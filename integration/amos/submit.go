// Package amos binds Prosody's first text operation to the inspected candidate
// public interfaces. The candidate is unmerged and this adapter is unqualified.
package amos

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ajent-social/amos/app/operation"
	"github.com/ajent-social/amos/identity"
	"github.com/ajent-social/amos/jobs"
	"github.com/ajent-social/amos/policy"
	"github.com/ajent-social/amos/workspace/authorization"
	wc "github.com/ajent-social/amos/workspace/context"
	"github.com/google/uuid"
	"github.com/narrate-it/prosody/internal/content"
	"github.com/narrate-it/prosody/internal/store"
	"github.com/narrate-it/prosody/internal/strictjson"
)

const OperationID = "prosody.messages.submit-text"
const EffectKind = "prosody.narration.prepare"
const InputSchema = `{"type":"object","properties":{"feedId":{"type":"string","description":"Canonical UUIDv7 destination feed","format":"uuid","minLength":36,"maxLength":36},"idempotencyKey":{"type":"string","description":"Globally unique submission key; 1–256 decoded UTF-8 bytes, retained exactly on retries","minLength":1,"maxLength":256},"text":{"type":"string","description":"Original source to condense into a briefing","minLength":1,"maxLength":4096},"title":{"type":"string","description":"Optional briefing title","minLength":1,"maxLength":160}},"required":["feedId","idempotencyKey","text"],"additionalProperties":false}`
const OutputSchema = `{"type":"object","properties":{"itemId":{"type":"string","description":"Durably accepted item","format":"uuid","minLength":36,"maxLength":36},"workId":{"type":"string","description":"Durable work receipt","format":"uuid","minLength":36,"maxLength":36},"status":{"type":"string","description":"Acceptance state","enum":["accepted"],"minLength":1,"maxLength":32},"acceptedAt":{"type":"string","description":"Original durable acceptance time, retained on replay","format":"date-time","minLength":20,"maxLength":35}},"required":["itemId","workId","status","acceptedAt"],"additionalProperties":false}`
const PayloadSchema = `{"type":"object","properties":{"workspace_id":{"type":"string","description":"Current workspace","format":"uuid","minLength":36,"maxLength":36},"item_id":{"type":"string","description":"Immutable source item","format":"uuid","minLength":36,"maxLength":36}},"required":["workspace_id","item_id"],"additionalProperties":false}`

type Input struct {
	FeedID string `json:"feedId"`
	Key    string `json:"idempotencyKey"`
	Text   string `json:"text"`
	Title  string `json:"title,omitempty"`
}
type Output struct {
	ItemID     string    `json:"itemId"`
	WorkID     string    `json:"workId"`
	Status     string    `json:"status"`
	AcceptedAt time.Time `json:"acceptedAt"`
}

// Acceptance is immutable; a replay must not claim the item is still processing.
func acceptedOutput(receipt store.Receipt) Output {
	return Output{ItemID: receipt.ItemID, WorkID: receipt.WorkID, Status: "accepted", AcceptedAt: receipt.AcceptedAt.UTC()}
}

type Payload struct {
	WorkspaceID string `json:"workspace_id"`
	ItemID      string `json:"item_id"`
}

// Snapshot resolves current provider setup and finite job admission from the
// held invocation context. It must return references/configuration, no secrets.
// This required host boundary is deliberately not replaced by default success.
type Snapshot func(context.Context, operation.InvocationContext, string) ([]byte, error)

type codec struct{ inner *operation.JSONCodec[Input] }

func (c codec) SchemaDigest() [32]byte { return c.inner.SchemaDigest() }
func (c codec) SchemaDocument() []byte { return c.inner.SchemaDocument() }
func (c codec) DecodeCanonical(raw operation.RawInput) (Input, []byte, error) {
	if len(raw.Body) > 65536 || strictjson.Validate(raw.Body) != nil {
		return Input{}, nil, content.ErrInvalid
	}
	in, canonical, err := c.inner.DecodeCanonical(raw)
	if err != nil {
		return in, nil, err
	}
	if !store.FeedID(in.FeedID) {
		return Input{}, nil, content.ErrInvalid
	}
	// Workspace is supplied only after policy selection; decode validates payload bounds.
	check := content.TextSubmission{FeedID: in.FeedID, IdempotencyKey: in.Key, Text: in.Text, Title: in.Title}
	if err = check.ValidatePayload(); err != nil {
		return Input{}, nil, err
	}
	return in, canonical, nil
}

type resolver struct{}

func (resolver) Resolve(_ identity.Principal, s wc.Selection, in Input) (operation.ResolvedResources, error) {
	id, err := uuid.Parse(in.FeedID)
	if err != nil || !store.FeedID(in.FeedID) {
		return operation.ResolvedResources{}, content.ErrInvalid
	}
	feed := policy.Resource{Type: "prosody_feed", ID: id, WorkspaceID: s.Workspace.ID}
	return operation.ResolvedResources{AuthorizationResource: feed, LockSet: []policy.Resource{{Type: "workspace", ID: s.Workspace.ID, WorkspaceID: s.Workspace.ID}, feed}}, nil
}
func Permissions() []authorization.AppPermission {
	return []authorization.AppPermission{{Name: "prosody.messages.submit", Roles: []string{"owner", "admin", "member"}}}
}
func ResourceTables() []operation.ResourceTable {
	return []operation.ResourceTable{{Type: "prosody_feed", Table: "app_prosody_feeds", IDColumn: "id", WorkspaceColumn: "workspace_id"}}
}

func Contracts(snapshot Snapshot) (*operation.FrozenRegistry, []byte, []byte, []operation.EffectDefinition, error) {
	if snapshot == nil {
		return nil, nil, nil, nil, errors.New("current provider/job admission resolver required")
	}
	input, err := operation.NewJSONSchema([]byte(InputSchema))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	output, err := operation.NewJSONSchema([]byte(OutputSchema))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	payload, err := operation.NewJSONSchema([]byte(PayloadSchema))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	ic, err := operation.NewJSONCodec[Input](input)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	oc, err := operation.NewJSONCodec[Output](output)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	pc, err := operation.NewJSONCodec[Payload](payload)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	effect, err := operation.BindEffect(EffectKind, pc)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	metadata := operation.Metadata{OperationID: OperationID, Method: "POST", Path: "/prosody/v1/messages/submit-text", InputType: "ProsodyTextInput", OutputType: "ProsodyReceipt", Features: []string{"typed-json"}, Revision: "prosody.text.v1", InputSchemaDigest: input.SchemaDigest(), OutputSchemaDigest: output.SchemaDigest(), Requirements: policy.Requirements{Permissions: []string{"prosody.messages.submit"}, Entitlements: []string{}, Assurance: identity.AAL1, MCPExposure: policy.MCPEligible}, SideEffect: operation.SideEffectWrite, Idempotency: operation.IdempotencyRequired, IdempotencyField: "/idempotencyKey", SuccessKind: operation.ResultAccepted, OutputClass: operation.OutputReplaySafe, MaxReplayBytes: 1024, Resource: &operation.ResourceSelector{Type: "prosody_feed", Field: "/feedId"}, Effects: []operation.EffectMetadata{{Kind: EffectKind, SchemaDigest: payload.SchemaDigest(), MaxBytes: 256, Privacy: "internal", WorkspaceField: "/workspace_id", MaxDeadlineMillis: 1800000}}}
	handler := func(ctx context.Context, call operation.InvocationContext, in Input) (operation.Result[Output], error) {
		if call.DB == nil || call.Effects == nil {
			return operation.Result[Output]{}, errors.New("transactional operation participants unavailable")
		}
		settings, err := snapshot(ctx, call, in.FeedID)
		if err != nil {
			return operation.Result[Output]{}, err
		}
		item, err := uuid.NewV7()
		if err != nil {
			return operation.Result[Output]{}, fmt.Errorf("allocate item: %w", err)
		}
		p := call.Principal
		scope := store.Scope{InstallationID: p.InstallationID().String(), ApplicationID: p.ApplicationID().String(), EnvironmentID: p.EnvironmentID().String(), WorkspaceID: call.Selection.Workspace.ID.String(), PersonID: p.Actor().PersonID().String(), OperationID: OperationID}
		tx := store.Transaction{ExecContext: call.DB.ExecContext, QueryRowContext: func(ctx context.Context, q string, args ...any) store.Row {
			return call.DB.QueryRowContext(ctx, q, args...)
		}}
		receipt, _, err := store.Accept(ctx, tx, store.Acceptance{Scope: scope, Submission: content.TextSubmission{WorkspaceID: scope.WorkspaceID, FeedID: in.FeedID, IdempotencyKey: in.Key, Text: in.Text, Title: in.Title}, ItemID: item.String(), SettingsSnapshot: settings, AuthorityReference: []byte(call.InvocationID.String()), StageWork: func(ctx context.Context, item string) (string, error) {
			var now time.Time
			if err := call.DB.QueryRowContext(ctx, `SELECT pg_catalog.clock_timestamp()`).Scan(&now); err != nil {
				return "", err
			}
			body, err := json.Marshal(Payload{scope.WorkspaceID, item})
			if err != nil {
				return "", err
			}
			job, err := call.Effects.Enqueue(ctx, 0, jobs.Intent{Kind: EffectKind, Payload: body, Deadline: now.Add(25 * time.Minute)})
			if err != nil {
				return "", err
			}
			return job.ID.String(), nil
		}})
		if err != nil {
			return operation.Result[Output]{}, err
		}
		return operation.Result[Output]{Kind: operation.ResultAccepted, Value: acceptedOutput(receipt)}, nil
	}
	definition, err := operation.Bind(metadata, codec{ic}, resolver{}, oc, handler)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	registry, err := operation.NewRegistry(definition)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	digest := definition.DescriptorDigest()
	policyJSON := map[string]any{"operationId": OperationID, "permissions": metadata.Requirements.Permissions, "entitlements": []string{}, "assurance": "aal1", "mcp": "eligible", "sideEffect": "write", "idempotency": "required"}
	api, err := json.Marshal(map[string]any{"openapi": "3.1.1", "info": map[string]any{"title": "Prosody", "version": "1"}, "paths": map[string]any{metadata.Path: map[string]any{"post": map[string]any{"operationId": OperationID, "x-amos-policy": policyJSON, "requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": json.RawMessage(InputSchema)}}}, "responses": map[string]any{"202": map[string]any{"description": "Durable source and work accepted", "content": map[string]any{"application/json": map[string]any{"schema": json.RawMessage(OutputSchema)}}}}}}}})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	mapping, err := json.Marshal(map[string]any{"contract": "amos-mcp-bound-mapping-v1", "operations": []any{map[string]any{"operationId": OperationID, "toolName": "prosody_submit_text", "description": "Send text to a permitted feed for a condensed spoken briefing", "descriptorDigest": hex.EncodeToString(digest[:]), "inputSchema": json.RawMessage(InputSchema), "outputSchema": json.RawMessage(OutputSchema), "workspaceSelection": "active_principal", "resourceSelection": "request_bound", "resourceType": "prosody_feed", "resourceField": "/feedId", "resourceDelete": false}}})
	return registry, api, mapping, []operation.EffectDefinition{effect}, err
}
