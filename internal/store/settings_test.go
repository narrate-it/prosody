package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/narrate-it/prosody/internal/configuration"
	"github.com/narrate-it/prosody/internal/keyvault"
	"github.com/narrate-it/prosody/internal/openrouter"
)

func TestProviderSettingsPersistenceAndRevocation(t *testing.T) {
	db, _ := openFixture(t)
	a := testAcceptance(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// This fixture key has no external provider value.
	secret := []byte("test-only-provider-credential")
	vault, err := keyvault.New(bytes.Repeat([]byte{42}, 32))
	if err != nil {
		t.Fatal(err)
	}
	var catalog []openrouter.Model
	if err = json.Unmarshal([]byte(`[{"id":"text","context_length":8192,"architecture":{"input_modalities":["text"],"output_modalities":["text"]}},{"id":"speech","supported_voices":["voice"],"architecture":{"input_modalities":["text"],"output_modalities":["audio"]}}]`), &catalog); err != nil {
		t.Fatal(err)
	}
	change := SettingsChange{InstallationID: a.Scope.InstallationID, WorkspaceID: a.Scope.WorkspaceID, FeedID: a.Submission.FeedID, Selection: configuration.Selection{TextModel: "text", SpeechModel: "speech", Voice: "voice"}, Catalog: catalog, NewKey: secret, CredentialID: testID(t)}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM public.app_prosody_settings WHERE workspace_id=$1`, a.Scope.WorkspaceID); err != nil {
			t.Error(err)
		}
	})
	save := func(c SettingsChange, want error) configuration.Snapshot {
		t.Helper()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer rollbackFixture(t, tx)
		snapshot, err := SaveSettings(ctx, FromSQL(tx), vault, c)
		if !errors.Is(err, want) {
			t.Fatalf("settings got %v want %v", err, want)
		}
		if want == nil {
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
		}
		return snapshot
	}
	first := save(change, nil)
	if first.SettingsRevision != 1 || first.CredentialID != change.CredentialID {
		t.Fatal("incorrect initial snapshot")
	}
	var value, envelope []byte
	if err = db.QueryRowContext(ctx, `SELECT value_bytes,key_envelope FROM public.app_prosody_settings WHERE workspace_id=$1 AND feed_id=$2`, change.WorkspaceID, change.FeedID).Scan(&value, &envelope); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(value, secret) || bytes.Contains(envelope, secret) {
		t.Fatal("plaintext credential persisted")
	}
	scope := keyvault.Scope{Installation: change.InstallationID, Workspace: change.WorkspaceID, Credential: change.CredentialID, Revision: "1"}
	decoded, err := vault.Open(scope, envelope)
	if err != nil || !bytes.Equal(decoded, secret) {
		t.Fatal("encrypted credential did not round trip")
	}
	scope.Workspace = testID(t)
	if _, err = vault.Open(scope, envelope); err == nil {
		t.Fatal("credential opened in different workspace")
	}
	// A stale update must not replace either the choices or the encrypted key.
	stale := change
	stale.NewKey = []byte("stale-replacement")
	stale.CredentialID = testID(t)
	save(stale, ErrConflict)
	change.ExpectedRevision = 1
	change.NewKey = nil
	preserved := save(change, nil)
	if preserved.SettingsRevision != 2 || preserved.CredentialID != first.CredentialID {
		t.Fatal("model-only edit changed key identity")
	}
	var retained []byte
	if err = db.QueryRowContext(ctx, `SELECT key_envelope FROM public.app_prosody_settings WHERE workspace_id=$1 AND feed_id=$2`, change.WorkspaceID, change.FeedID).Scan(&retained); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retained, envelope) {
		t.Fatal("model-only edit rewrote key")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rollbackFixture(t, tx)
	snapshot, err := CurrentSettings(ctx, FromSQL(tx), change.WorkspaceID, change.FeedID)
	if err != nil || bytes.Contains(snapshot, secret) {
		t.Fatalf("admission snapshot: %v", err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	change.ExpectedRevision = 2
	change.NewKey = []byte("replacement-test-credential")
	change.CredentialID = testID(t)
	rotated := save(change, nil)
	if rotated.SettingsRevision != 3 || rotated.CredentialID == preserved.CredentialID {
		t.Fatal("rotation did not advance identity")
	}
	var rotatedEnvelope []byte
	if err = db.QueryRowContext(ctx, `SELECT key_envelope FROM public.app_prosody_settings WHERE workspace_id=$1 AND feed_id=$2`, change.WorkspaceID, change.FeedID).Scan(&rotatedEnvelope); err != nil {
		t.Fatal(err)
	}
	scope.Workspace = change.WorkspaceID
	scope.Credential = rotated.CredentialID
	decoded, err = vault.Open(scope, rotatedEnvelope)
	if err != nil || !bytes.Equal(decoded, change.NewKey) {
		t.Fatal("rotated credential did not round trip")
	}
	scope.Credential = preserved.CredentialID
	if _, err = vault.Open(scope, rotatedEnvelope); err == nil {
		t.Fatal("new credential opened with old identity")
	}
	change.ExpectedRevision = 3
	change.NewKey = nil
	change.RevokeKey = true
	revoked := save(change, nil)
	if revoked.SettingsRevision != 4 || revoked.CredentialID != "" || revoked.CredentialRevision != 0 {
		t.Fatal("revocation retained credential reference")
	}
	check, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rollbackFixture(t, check)
	if _, err = CurrentSettings(ctx, FromSQL(check), change.WorkspaceID, change.FeedID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("revoked settings admitted: %v", err)
	}
	var removed bool
	if err = check.QueryRowContext(ctx, `SELECT key_envelope IS NULL AND credential_id IS NULL AND credential_revision IS NULL FROM public.app_prosody_settings WHERE workspace_id=$1 AND feed_id=$2`, change.WorkspaceID, change.FeedID).Scan(&removed); err != nil || !removed {
		t.Fatalf("revocation did not clear credential: %v", err)
	}
}
