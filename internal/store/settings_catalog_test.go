package store_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/narrate-it/prosody/internal/configuration"
	"github.com/narrate-it/prosody/internal/keyvault"
	"github.com/narrate-it/prosody/internal/openrouter"
	"github.com/narrate-it/prosody/internal/store"
)

type settingsCatalogRow func(...any) error

// Scan supplies the controlled SQL error once a valid selection reaches SQL.
func (r settingsCatalogRow) Scan(dest ...any) error { return r(dest...) }

// TestSaveSettingsRejectsAmbiguousCatalogBeforeSQL uses valid IDs, a real
// nonnil vault and complete ports. The unique positive control must reach SQL,
// preventing unrelated preflight rejection from making the negative test pass.
func TestSaveSettingsRejectsAmbiguousCatalogBeforeSQL(t *testing.T) {
	t.Parallel()
	for _, tc := range []string{"unique control", "duplicate text", "duplicate speech"} {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			var catalog []openrouter.Model
			if err := json.Unmarshal([]byte(`[{"id":"text","context_length":8192,"top_provider":{"context_length":8192,"max_completion_tokens":4096},"supported_parameters":["max_tokens"],"architecture":{"input_modalities":["text"],"output_modalities":["text"]}},{"id":"elevenlabs/eleven-v4-turbo","supported_voices":["voice"],"architecture":{"input_modalities":["text"],"output_modalities":["speech"]}}]`), &catalog); err != nil {
				t.Fatal(err)
			}
			switch tc {
			case "duplicate text":
				catalog = append(catalog, catalog[0])
			case "duplicate speech":
				catalog = append(catalog, catalog[1])
			}
			vault, err := keyvault.New(bytes.Repeat([]byte{42}, 32))
			if err != nil {
				t.Fatal(err)
			}
			queries := 0
			tx := store.Transaction{
				QueryRowContext: func(context.Context, string, ...any) store.Row {
					queries++
					return settingsCatalogRow(func(...any) error { return sql.ErrNoRows })
				},
				ExecContext: func(context.Context, string, ...any) (sql.Result, error) {
					t.Fatal("unexpected write")
					return nil, store.ErrInvalid
				},
			}
			change := store.SettingsChange{
				InstallationID: "01900000-0000-7000-8000-000000000001", WorkspaceID: "01900000-0000-7000-8000-000000000002", FeedID: "01900000-0000-7000-8000-000000000003",
				Selection: configuration.Selection{TextModel: "text", SpeechModel: "elevenlabs/eleven-v4-turbo", Voice: "voice"}, Catalog: catalog,
			}
			snapshot, err := store.SaveSettings(context.Background(), tx, vault, change)
			if snapshot != (configuration.Snapshot{}) {
				t.Fatal("rejected settings returned a usable snapshot")
			}
			if tc == "unique control" {
				if !errors.Is(err, store.ErrUnavailable) || queries != 1 {
					t.Fatalf("unique control did not reach feed-lock SQL: calls=%d err=%v", queries, err)
				}
			} else if !errors.Is(err, configuration.ErrInvalid) || queries != 0 {
				t.Fatalf("ambiguous selection reached SQL or lost cause: calls=%d err=%v", queries, err)
			}
		})
	}
}
