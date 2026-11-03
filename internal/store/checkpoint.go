package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/narrate-it/prosody/internal/strictjson"
	"unicode/utf8"
)

// CheckpointMigration is separate and deliberately not enabled in the AMOS
// adapter. Future native registration must admit the exact private query catalog.
//
//go:embed migrations/002_stage_checkpoints.sql
var CheckpointMigration string

// CheckpointBindingEncodingVersion freezes the private binding envelope separately
// from CodecDigest, which identifies the native-accepted stage result codec.
// Changing the envelope shape requires a new version and native registration.
const CheckpointBindingEncodingVersion = 1

// CheckpointBinding is data supplied inside a future native Checkpoint callback,
// NOT an authority object or native receipt. Source/settings hashes refer to
// exact persisted bytes. ArtifactDigest covers the actual private bytes below;
// ResultDigest and ManifestDigest bind the native-accepted codec result/ordered
// artifact manifest, whose proof and canonicalization remain native obligations.
type CheckpointBinding struct {
	InstallationID, ApplicationID, EnvironmentID, PersonID                         string
	WorkspaceID, JobID, InvocationID, AttemptID, ItemID, FeedID                    string
	Stage                                                                          string
	SettingsRevision                                                               int64
	SourceDigest, SettingsDigest, RequestDigest, ResultDigest                      [32]byte
	RegistrationDigest, ProfileDigest, CodecDigest, ManifestDigest, ArtifactDigest [32]byte
}

// EncodeCheckpointBinding returns the exact versioned private envelope for a
// future registered byte-argument codec. It validates descriptive data only;
// possession of these bytes confers no native candidate, receipt or authority.
func EncodeCheckpointBinding(b CheckpointBinding) ([]byte, error) {
	return b.encode()
}

func (b CheckpointBinding) encode() ([]byte, error) {
	for _, id := range []string{b.InstallationID, b.PersonID, b.WorkspaceID, b.JobID, b.InvocationID, b.AttemptID, b.ItemID} {
		if !UUID(id) {
			return nil, ErrInvalid
		}
	}
	if !safeLabel(b.ApplicationID) || !safeLabel(b.EnvironmentID) || !FeedID(b.FeedID) || b.SettingsRevision <= 0 || (b.Stage != "rewrite" && b.Stage != "speech") {
		return nil, ErrInvalid
	}
	for _, h := range [][32]byte{b.SourceDigest, b.SettingsDigest, b.RequestDigest, b.ResultDigest, b.RegistrationDigest, b.ProfileDigest, b.CodecDigest, b.ManifestDigest, b.ArtifactDigest} {
		if h == [32]byte{} {
			return nil, ErrInvalid
		}
	}
	encoded, err := json.Marshal(struct {
		Version int               `json:"version"`
		Binding CheckpointBinding `json:"binding"`
	}{CheckpointBindingEncodingVersion, b})
	if err != nil || len(encoded) > 8192 {
		return nil, ErrInvalid
	}
	return encoded, nil
}

// ValidateCheckpointArtifact is a bounded local byte codec/binding check. Speech
// MP3 decoding/duration and native accepted-result proof are separate; this does
// not promote arbitrary bytes to a published or remotely settled media asset.
func ValidateCheckpointArtifact(b CheckpointBinding, payload []byte) error {
	if _, err := b.encode(); err != nil {
		return err
	}
	max := 67108864
	if b.Stage == "rewrite" {
		max = 256000
		if !utf8.Valid(payload) {
			return ErrInvalid
		}
	}
	if len(payload) == 0 || len(payload) > max || sha256.Sum256(payload) != b.ArtifactDigest {
		return ErrInvalid
	}
	return nil
}

const CheckpointSourceSQL = `SELECT s.installation_id,s.application_id,s.environment_id,s.person_id,s.feed_id,CASE WHEN octet_length(s.source_bytes)<=65536 THEN s.source_bytes ELSE NULL END,CASE WHEN octet_length(s.settings_bytes)<=65536 THEN s.settings_bytes ELSE NULL END FROM public.app_prosody_submissions s JOIN public.app_prosody_work w ON w.item_id=s.id JOIN public.app_prosody_feeds f ON f.workspace_id=s.workspace_id AND f.id=s.feed_id WHERE s.workspace_id=$1 AND s.id=$2 AND w.work_id=$3 AND s.status='processing' AND s.deleted_at IS NULL AND f.deleted_at IS NULL FOR SHARE OF s,w,f`
const CheckpointInsertSQL = `INSERT INTO public.app_prosody_stage_checkpoints(workspace_id,job_id,invocation_id,stage_id,item_id,feed_id,binding_bytes,artifact_bytes,artifact_digest) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(workspace_id,job_id,invocation_id,stage_id) DO NOTHING`
const CheckpointReadSQL = `SELECT CASE WHEN octet_length(binding_bytes)<=8192 THEN binding_bytes ELSE NULL END,CASE WHEN octet_length(artifact_bytes)<=CASE WHEN stage_id='rewrite' THEN 256000 ELSE 67108864 END THEN artifact_bytes ELSE NULL END,artifact_digest FROM public.app_prosody_stage_checkpoints WHERE workspace_id=$1 AND job_id=$2 AND invocation_id=$3 AND stage_id=$4 FOR SHARE`

func checkpointSource(ctx context.Context, tx Transaction, b CheckpointBinding) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var installation, application, environment, person, feed string
	var source, settings []byte
	err := tx.QueryRowContext(ctx, CheckpointSourceSQL, b.WorkspaceID, b.ItemID, b.JobID).Scan(&installation, &application, &environment, &person, &feed, &source, &settings)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnavailable
	}
	if err != nil {
		return fmt.Errorf("read checkpoint source: %w", err)
	}
	revision, revisionErr := checkpointSettingsRevision(settings)
	if len(source) == 0 || len(source) > 65536 || len(settings) == 0 || len(settings) > 65536 || installation != b.InstallationID || application != b.ApplicationID || environment != b.EnvironmentID || person != b.PersonID || feed != b.FeedID || sha256.Sum256(source) != b.SourceDigest || sha256.Sum256(settings) != b.SettingsDigest || revisionErr != nil || revision != b.SettingsRevision {
		return ErrConflict
	}
	return nil
}

// checkpointSettingsRevision accepts the exact spelling once, without Go's
// case-insensitive field matching or last-duplicate-wins behavior. Other snapshot
// fields remain owned by their registered configuration codec.
func checkpointSettingsRevision(settings []byte) (int64, error) {
	if len(settings) == 0 || len(settings) > 65536 || strictjson.Validate(settings) != nil {
		return 0, ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(settings))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return 0, ErrInvalid
	}
	seen := map[string]bool{}
	var revision int64
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return 0, ErrInvalid
		}
		key, ok := token.(string)
		if !ok || seen[key] || (strings.EqualFold(key, "settingsRevision") && key != "settingsRevision") {
			return 0, ErrInvalid
		}
		seen[key] = true
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return 0, ErrInvalid
		}
		if key == "settingsRevision" && json.Unmarshal(value, &revision) != nil {
			return 0, ErrInvalid
		}
	}
	if !seen["settingsRevision"] || revision <= 0 {
		return 0, ErrInvalid
	}
	return revision, nil
}

// WriteCheckpoint stages immutable private bytes, returning whether inserted.
// The host MUST atomically stage its own native checkpoint receipt in this SAME
// transaction and roll back on any error. Returning true/false does not prove
// commit or permit the next stage. Uncertain commit reconciles the same identity.
func WriteCheckpoint(ctx context.Context, tx Transaction, b CheckpointBinding, payload []byte) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !tx.valid() {
		return false, ErrInvalid
	}
	if err := ValidateCheckpointArtifact(b, payload); err != nil {
		return false, err
	}
	if err := checkpointSource(ctx, tx, b); err != nil {
		return false, err
	}
	binding, err := b.encode()
	if err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	result, err := tx.ExecContext(ctx, CheckpointInsertSQL, b.WorkspaceID, b.JobID, b.InvocationID, b.Stage, b.ItemID, b.FeedID, binding, payload, b.ArtifactDigest[:])
	if err != nil {
		return false, fmt.Errorf("insert private checkpoint: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if result == nil {
		return false, ErrInvalid
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count checkpoint rows: %w", err)
	}
	if count != 0 && count != 1 {
		return false, ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if count == 1 {
		return true, nil
	}
	_, err = ReadCheckpoint(ctx, tx, b)
	return false, err
}

// ReadCheckpoint returns only matching private bytes after rechecking current
// source/feed tombstones and exact input bindings. Native receipt reconciliation
// and current authority/fence admission still enclose this read; it is not an
// AMOS receipt lookup or permission to reuse a previous fence.
func ReadCheckpoint(ctx context.Context, tx Transaction, b CheckpointBinding) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !tx.valid() {
		return nil, ErrInvalid
	}
	expected, err := b.encode()
	if err != nil {
		return nil, err
	}
	if err := checkpointSource(ctx, tx, b); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var binding, payload, digest []byte
	err = tx.QueryRowContext(ctx, CheckpointReadSQL, b.WorkspaceID, b.JobID, b.InvocationID, b.Stage).Scan(&binding, &payload, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("read private checkpoint: %w", err)
	}
	if !bytes.Equal(expected, binding) || !bytes.Equal(b.ArtifactDigest[:], digest) || ValidateCheckpointArtifact(b, payload) != nil {
		return nil, ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return payload, nil
}
