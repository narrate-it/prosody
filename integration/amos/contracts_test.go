package amos

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ajent-social/amos/app/operation"
	"github.com/narrate-it/prosody/internal/store"
)

func TestAcceptanceReceiptRetainsPersistedIdentityAndTime(t *testing.T) {
	schema, err := operation.NewJSONSchema([]byte(OutputSchema))
	if err != nil {
		t.Fatal(err)
	}
	outputCodec, err := operation.NewJSONCodec[Output](schema)
	if err != nil {
		t.Fatal(err)
	}
	accepted := time.Date(2026, 11, 1, 12, 34, 56, 123456000, time.FixedZone("source", -7*60*60))
	receipt := store.Receipt{ItemID: "01900000-0000-7000-8000-000000000001", WorkID: "01900000-0000-7000-8000-000000000002", AcceptedAt: accepted}
	raw, err := outputCodec.EncodeCanonical(acceptedOutput(receipt))
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 || fields["itemId"] != receipt.ItemID || fields["workId"] != receipt.WorkID || fields["status"] != "accepted" || fields["acceptedAt"] != "2026-11-01T19:34:56.123456Z" {
		t.Fatalf("incorrect acceptance receipt: %s", raw)
	}
	// Equivalent persisted timestamps yield identical replay bytes; no current
	// time or later processing state participates in the receipt projection.
	receipt.AcceptedAt = accepted.UTC()
	replayed, err := outputCodec.EncodeCanonical(acceptedOutput(receipt))
	if err != nil || string(replayed) != string(raw) {
		t.Fatalf("replay receipt changed: %s, %v", replayed, err)
	}
	for _, invalid := range []string{
		strings.Replace(string(raw), `"accepted"`, `"processing"`, 1),
		strings.Replace(string(raw), `,"acceptedAt":"2026-11-01T19:34:56.123456Z"`, "", 1),
		strings.Replace(string(raw), "2026-11-01T19:34:56.123456Z", "not-a-timestamp", 1),
	} {
		if schema.Validate([]byte(invalid)) == nil {
			t.Fatalf("accepted invalid receipt: %s", invalid)
		}
	}
}

func TestFirstOperationDescriptor(t *testing.T) {
	registry, api, mapping, effects, err := Contracts(func(context.Context, operation.InvocationContext, string) ([]byte, error) {
		return []byte(`{"testOnly":true}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	definitions := registry.Definitions()
	if len(definitions) != 1 || len(effects) != 1 {
		t.Fatal("unexpected operation inventory")
	}
	m := definitions[0].Metadata()
	if m.OperationID != OperationID || m.IdempotencyField != "/idempotencyKey" || m.SuccessKind != operation.ResultAccepted || m.Resource == nil || m.Resource.Field != "/feedId" {
		t.Fatal("descriptor lost admission bindings")
	}
	if !strings.Contains(string(api), `"202"`) || !strings.Contains(string(mapping), `"workspaceSelection":"active_principal"`) {
		t.Fatal("transport mapping mismatch")
	}
}
func TestMissingAdmissionResolverFails(t *testing.T) {
	if _, _, _, _, err := Contracts(nil); err == nil {
		t.Fatal("accepted absent admission resolver")
	}
}
func TestBodyKeyUnicode(t *testing.T) {
	schema, err := operation.NewJSONSchema([]byte(InputSchema))
	if err != nil {
		t.Fatal(err)
	}
	inner, err := operation.NewJSONCodec[Input](schema)
	if err != nil {
		t.Fatal(err)
	}
	c := codec{inner}
	for _, tc := range []struct {
		key   string
		valid bool
	}{
		{`" \u0000\r\n界 "`, true}, {`"\ud800"`, false}, {`"` + strings.Repeat("界", 86) + `"`, false},
	} {
		raw := []byte(`{"feedId":"01900000-0000-7000-8000-000000000001","idempotencyKey":` + tc.key + `,"text":"source"}`)
		_, _, err = c.DecodeCanonical(operation.RawInput{Body: raw})
		if (err == nil) != tc.valid {
			t.Fatalf("valid=%t err=%v", tc.valid, err)
		}
	}
}
