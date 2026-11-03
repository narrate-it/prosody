package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
)

func checkpointBinding(t *testing.T, source, settings, payload []byte) CheckpointBinding {
	t.Helper()
	digest := sha256.Sum256([]byte("registered fixture digest"))
	return CheckpointBinding{InstallationID: testID(t), ApplicationID: testID(t), EnvironmentID: testID(t), PersonID: testID(t), WorkspaceID: testID(t), JobID: testID(t), InvocationID: testID(t), AttemptID: testID(t), ItemID: testID(t), FeedID: testID(t), Stage: "rewrite", SettingsRevision: 1, SourceDigest: sha256.Sum256(source), SettingsDigest: sha256.Sum256(settings), ArtifactDigest: sha256.Sum256(payload), RequestDigest: digest, ResultDigest: digest, RegistrationDigest: digest, ProfileDigest: digest, CodecDigest: digest, ManifestDigest: digest}
}
func TestCheckpointByteCodec(t *testing.T) {
	t.Parallel()
	payload := []byte("Briefing.")
	b := checkpointBinding(t, []byte("source"), []byte("settings"), payload)
	if err := ValidateCheckpointArtifact(b, payload); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*CheckpointBinding){func(b *CheckpointBinding) { b.Stage = "ready" }, func(b *CheckpointBinding) { b.ArtifactDigest = [32]byte{} }, func(b *CheckpointBinding) { b.InvocationID = "" }, func(b *CheckpointBinding) { b.SettingsRevision = 0 }, func(b *CheckpointBinding) { b.ResultDigest = [32]byte{} }} {
		changed := b
		mutate(&changed)
		if !errors.Is(ValidateCheckpointArtifact(changed, payload), ErrInvalid) {
			t.Fatal("invalid binding admitted")
		}
	}
	if !errors.Is(ValidateCheckpointArtifact(b, []byte("different")), ErrInvalid) {
		t.Fatal("different actual bytes admitted")
	}
	malformed := []byte{0xff}
	b.ArtifactDigest = sha256.Sum256(malformed)
	if !errors.Is(ValidateCheckpointArtifact(b, malformed), ErrInvalid) {
		t.Fatal("non UTF8 rewrite admitted")
	}
}

func TestPrivateCheckpointPostgres(t *testing.T) {
	db, _ := openFixture(t)
	if _, err := db.ExecContext(context.Background(), CheckpointMigration); err != nil {
		t.Fatal(err)
	}
	a := testAcceptance(t, db)
	a.Scope.ApplicationID, a.Scope.EnvironmentID = testID(t), testID(t)
	a.SettingsSnapshot = []byte(`{"settingsRevision":1}`)
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DELETE FROM public.app_prosody_stage_checkpoints WHERE workspace_id=$1`, a.Scope.WorkspaceID); err != nil {
			t.Error(err)
		}
	})
	tx := beginCheckpoint(t, db)
	receipt, _, err := Accept(context.Background(), FromSQL(tx), a)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(a.Submission)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("private spoken briefing")
	b := checkpointBinding(t, source, a.SettingsSnapshot, payload)
	b.InstallationID, b.ApplicationID, b.EnvironmentID, b.PersonID = a.Scope.InstallationID, a.Scope.ApplicationID, a.Scope.EnvironmentID, a.Scope.PersonID
	b.WorkspaceID, b.ItemID, b.FeedID, b.JobID = a.Scope.WorkspaceID, receipt.ItemID, a.Submission.FeedID, receipt.WorkID
	write := func(commit bool) (bool, error) {
		tx := beginCheckpoint(t, db)
		fresh, err := WriteCheckpoint(context.Background(), FromSQL(tx), b, payload)
		if err != nil || !commit {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				t.Fatal(rollbackErr)
			}
			return fresh, err
		}
		return fresh, tx.Commit()
	}
	if fresh, err := write(false); err != nil || !fresh {
		t.Fatalf("initial private insert: %v", err)
	}
	tx = beginCheckpoint(t, db)
	_, err = ReadCheckpoint(context.Background(), FromSQL(tx), b)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatal("rollback retained provisional bytes")
	}
	rollback(t, tx)
	if fresh, err := write(true); err != nil || !fresh {
		t.Fatalf("commit private insert: %v", err)
	}
	if fresh, err := write(true); err != nil || fresh {
		t.Fatalf("same identity replay: %v", err)
	}
	tx = beginCheckpoint(t, db)
	var status string
	if err := tx.QueryRowContext(context.Background(), `SELECT status FROM public.app_prosody_submissions WHERE workspace_id=$1 AND id=$2`, b.WorkspaceID, b.ItemID).Scan(&status); err != nil || status != "processing" {
		t.Fatal("checkpoint made source ready")
	}
	var count int
	if err := tx.QueryRowContext(context.Background(), `SELECT count(*) FROM public.app_prosody_renditions WHERE workspace_id=$1 AND item_id=$2`, b.WorkspaceID, b.ItemID).Scan(&count); err != nil || count != 0 {
		t.Fatal("checkpoint created final rendition")
	}
	rollback(t, tx)
	for _, mutate := range []func(*CheckpointBinding){func(b *CheckpointBinding) { b.WorkspaceID = testID(t) }, func(b *CheckpointBinding) { b.SourceDigest = sha256.Sum256([]byte("stale")) }, func(b *CheckpointBinding) { b.SettingsDigest = sha256.Sum256([]byte("stale")) }, func(b *CheckpointBinding) { b.ResultDigest = sha256.Sum256([]byte("changed result")) }, func(b *CheckpointBinding) { b.AttemptID = testID(t) }} {
		bad := b
		mutate(&bad)
		tx = beginCheckpoint(t, db)
		got, err := ReadCheckpoint(context.Background(), FromSQL(tx), bad)
		if err == nil || len(got) != 0 {
			t.Fatal("mismatched binding exposed private bytes")
		}
		rollback(t, tx)
	}
	tx = beginCheckpoint(t, db)
	changed := append([]byte(nil), payload...)
	changed[0] = 'X'
	bad := b
	bad.ArtifactDigest = sha256.Sum256(changed)
	if _, err := WriteCheckpoint(context.Background(), FromSQL(tx), bad, changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("same stage different artifact: %v", err)
	}
	rollback(t, tx)
	if _, err := db.ExecContext(context.Background(), `UPDATE public.app_prosody_feeds SET deleted_at=clock_timestamp() WHERE workspace_id=$1 AND id=$2`, b.WorkspaceID, b.FeedID); err != nil {
		t.Fatal(err)
	}
	tx = beginCheckpoint(t, db)
	if _, err := ReadCheckpoint(context.Background(), FromSQL(tx), b); !errors.Is(err, ErrUnavailable) {
		t.Fatal("tombstone read allowed")
	}
	rollback(t, tx)
}

func rollback(t *testing.T, tx *sql.Tx) {
	t.Helper()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
}

func beginCheckpoint(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rollbackFixture(t, tx) })
	return tx
}
