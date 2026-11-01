package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/narrate-it/prosody/internal/configuration"
	"github.com/narrate-it/prosody/internal/keyvault"
	"github.com/narrate-it/prosody/internal/openrouter"
)

type SettingsChange struct {
	InstallationID, WorkspaceID, FeedID string
	ExpectedRevision                    int64
	Selection                           configuration.Selection
	Catalog                             []openrouter.Model
	// NewKey is never marshaled into settings, snapshots, audit or job payloads.
	NewKey       []byte
	CredentialID string
	RevokeKey    bool
}

// SaveSettings requires current admin authority held by the caller transaction.
// A nil/empty NewKey preserves the existing key; RevokeKey explicitly removes it.
func SaveSettings(ctx context.Context, tx Transaction, vault *keyvault.Vault, c SettingsChange) (configuration.Snapshot, error) {
	if !tx.valid() || vault == nil || !UUID(c.InstallationID) || !UUID(c.WorkspaceID) || !FeedID(c.FeedID) || c.ExpectedRevision < 0 || c.RevokeKey && len(c.NewKey) > 0 {
		return configuration.Snapshot{}, ErrInvalid
	}
	if err := c.Selection.Validate(c.Catalog); err != nil {
		return configuration.Snapshot{}, err
	}
	var id string
	err := tx.QueryRowContext(ctx, `SELECT id FROM public.app_prosody_feeds WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE`, c.WorkspaceID, c.FeedID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return configuration.Snapshot{}, ErrUnavailable
	}
	if err != nil {
		return configuration.Snapshot{}, fmt.Errorf("lock settings feed: %w", err)
	}
	var revision int64
	var envelope []byte
	var credential sql.NullString
	var keyRevision sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT revision,key_envelope,credential_id,credential_revision FROM public.app_prosody_settings WHERE workspace_id=$1 AND feed_id=$2 FOR UPDATE`, c.WorkspaceID, c.FeedID).Scan(&revision, &envelope, &credential, &keyRevision)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return configuration.Snapshot{}, fmt.Errorf("read provider settings: %w", err)
	}
	if revision != c.ExpectedRevision || revision == int64(^uint64(0)>>1) {
		return configuration.Snapshot{}, ErrConflict
	}
	if c.RevokeKey {
		envelope = nil
		credential = sql.NullString{}
		keyRevision = sql.NullInt64{}
	}
	if len(c.NewKey) > 0 {
		if !UUID(c.CredentialID) || strings.TrimSpace(string(c.NewKey)) == "" || strings.ContainsAny(string(c.NewKey), "\x00\r\n") {
			return configuration.Snapshot{}, ErrInvalid
		}
		if credential.Valid && credential.String == c.CredentialID {
			return configuration.Snapshot{}, ErrInvalid
		}
		credential = sql.NullString{String: c.CredentialID, Valid: true}
		keyRevision = sql.NullInt64{Int64: 1, Valid: true}
		envelope, err = vault.Seal(keyvault.Scope{Installation: c.InstallationID, Workspace: c.WorkspaceID, Credential: credential.String, Revision: strconv.FormatInt(keyRevision.Int64, 10)}, c.NewKey)
		if err != nil {
			return configuration.Snapshot{}, err
		}
	}
	snapshot := configuration.Snapshot{Selection: c.Selection, SettingsRevision: revision + 1, CredentialID: credential.String, CredentialRevision: keyRevision.Int64}
	value, err := json.Marshal(snapshot)
	if err != nil {
		return configuration.Snapshot{}, fmt.Errorf("encode provider settings: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO public.app_prosody_settings(workspace_id,feed_id,revision,value_bytes,key_envelope,credential_id,credential_revision) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(workspace_id,feed_id) DO UPDATE SET revision=EXCLUDED.revision,value_bytes=EXCLUDED.value_bytes,key_envelope=EXCLUDED.key_envelope,credential_id=EXCLUDED.credential_id,credential_revision=EXCLUDED.credential_revision`, c.WorkspaceID, c.FeedID, revision+1, value, envelope, credential, keyRevision)
	if err != nil {
		return configuration.Snapshot{}, fmt.Errorf("save provider settings: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE public.app_prosody_feeds SET settings_revision=$3 WHERE workspace_id=$1 AND id=$2`, c.WorkspaceID, c.FeedID, revision+1); err != nil {
		return configuration.Snapshot{}, fmt.Errorf("advance settings revision: %w", err)
	}
	return snapshot, nil
}

// CurrentSettings returns only non-secret references for admission snapshots.
func CurrentSettings(ctx context.Context, tx Transaction, workspace, feed string) ([]byte, error) {
	if !tx.valid() || !UUID(workspace) || !FeedID(feed) {
		return nil, ErrInvalid
	}
	var value []byte
	err := tx.QueryRowContext(ctx, `SELECT s.value_bytes FROM public.app_prosody_settings s JOIN public.app_prosody_feeds f ON f.workspace_id=s.workspace_id AND f.id=s.feed_id WHERE s.workspace_id=$1 AND s.feed_id=$2 AND f.deleted_at IS NULL AND s.key_envelope IS NOT NULL AND s.credential_id IS NOT NULL FOR SHARE OF s,f`, workspace, feed).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("read current configuration: %w", err)
	}
	var snapshot configuration.Snapshot
	if err = json.Unmarshal(value, &snapshot); err != nil || !UUID(snapshot.CredentialID) || snapshot.CredentialRevision <= 0 || snapshot.SettingsRevision <= 0 {
		return nil, ErrInvalid
	}
	return value, nil
}
