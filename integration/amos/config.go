package amos

import (
	"context"
	"errors"

	"github.com/ajent-social/amos/app/operation"
	"github.com/ajent-social/amos/apphost"
	"github.com/ajent-social/amos/migrations"
	"github.com/narrate-it/prosody/internal/store"
)

// CurrentProviderSnapshot checks persisted current setup inside the invocation
// transaction. Finite budget admission must be composed by the host around it.
func CurrentProviderSnapshot(ctx context.Context, call operation.InvocationContext, feed string) ([]byte, error) {
	if call.DB == nil {
		return nil, errors.New("current transaction unavailable")
	}
	tx := store.Transaction{ExecContext: call.DB.ExecContext, QueryRowContext: func(ctx context.Context, q string, args ...any) store.Row {
		return call.DB.QueryRowContext(ctx, q, args...)
	}}
	return store.CurrentSettings(ctx, tx, call.Selection.Workspace.ID.String(), feed)
}

// MCPConfig supplies the first submission slice only. The integrator assigns
// firstMigration after its closed native profile and separately applies the
// returned migrations under operator authority. This function performs no I/O.
// Explicit enableCIMD is caller-owned; source composition is not qualification.
func MCPConfig(firstMigration uint64, enableCIMD bool, snapshot Snapshot) (*apphost.ProductionMCPConfig, error) {
	if firstMigration == 0 {
		return nil, errors.New("native migration boundary required")
	}
	registry, api, mapping, effects, err := Contracts(snapshot)
	if err != nil {
		return nil, err
	}
	return &apphost.ProductionMCPConfig{Registry: registry, OpenAPI: api, Mapping: mapping, Permissions: Permissions(), EnableCIMD: enableCIMD, ResourceTables: ResourceTables(), EffectDefinitions: effects, FirstMigration: firstMigration, ApplicationMigrations: []migrations.Migration{{Sequence: firstMigration + 2, Namespace: "application", Name: "prosody_domain", SQL: store.Migration}}, Tables: map[string][3]string{
		"app_prosody_feeds":       {"workspace_id,id,deleted_at", "", "id"},
		"app_prosody_settings":    {"workspace_id,feed_id,value_bytes,key_envelope,credential_id", "", "revision"},
		"app_prosody_submissions": {"installation_id,application_id,environment_id,workspace_id,person_id,operation_id,key_bytes,digest,id,feed_id,source_bytes,settings_bytes,authority_ref,status,accepted_at,deleted_at", "installation_id,application_id,environment_id,workspace_id,person_id,operation_id,key_bytes,digest,id,feed_id,source_bytes,settings_bytes,authority_ref", ""},
		"app_prosody_work":        {"item_id,work_id", "item_id,work_id", ""},
	}}, nil
}
