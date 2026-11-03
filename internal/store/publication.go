package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/narrate-it/prosody/internal/configuration"
)

// PublishedSegment contains asset IDs, not signed delivery URLs. Asset URLs are
// minted for a currently authorized human at read time and never persisted here.
type PublishedSegment struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	AssetID    string   `json:"assetId,omitempty"`
	ImageIDs   []string `json:"imageIds,omitempty"`
	DurationMS int64    `json:"durationMs"`
}
type Publication struct {
	WorkspaceID, ItemID, WorkID, RenditionID string
	SettingsRevision                         int64
	Segments                                 []PublishedSegment
	Script, Provenance                       []byte
	// StageReady stages a schema-bound notification intent through AMOS in the
	// same transaction. Dispatch still checks per-person opt-in/current authority.
	StageReady func(context.Context, string, string) error
}

// Publish atomically admits a complete rendition. Host-held AMOS job fences and
// current source/grant/credential authority remain required around this call.
func Publish(ctx context.Context, tx Transaction, p Publication) error {
	if !tx.valid() || !UUID(p.WorkspaceID) || !UUID(p.ItemID) || !UUID(p.WorkID) || !UUID(p.RenditionID) || p.SettingsRevision <= 0 || len(p.Segments) == 0 || len(p.Segments) > 100 || len(p.Script) > 256000 || len(p.Provenance) == 0 || len(p.Provenance) > 65536 || p.StageReady == nil {
		return ErrInvalid
	}
	var status string
	var snapshotBytes []byte
	err := tx.QueryRowContext(ctx, `SELECT s.status,s.settings_bytes FROM public.app_prosody_submissions s JOIN public.app_prosody_work w ON w.item_id=s.id WHERE s.workspace_id=$1 AND s.id=$2 AND w.work_id=$3 AND s.deleted_at IS NULL FOR UPDATE OF s`, p.WorkspaceID, p.ItemID, p.WorkID).Scan(&status, &snapshotBytes)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnavailable
	}
	if err != nil {
		return fmt.Errorf("lock publication source: %w", err)
	}
	if status != "processing" {
		return ErrConflict
	}
	var snapshot configuration.Snapshot
	if err = json.Unmarshal(snapshotBytes, &snapshot); err != nil || snapshot.SettingsRevision != p.SettingsRevision {
		return ErrConflict
	}
	total := int64(0)
	seen := map[string]bool{}
	for _, segment := range p.Segments {
		if !UUID(segment.ID) || seen[segment.ID] || len(segment.ImageIDs) > 20 {
			return ErrInvalid
		}
		seen[segment.ID] = true
		if segment.Kind != "image" {
			if !UUID(segment.AssetID) {
				return ErrInvalid
			}
			var mime string
			var duration int64
			var generated bool
			err = tx.QueryRowContext(ctx, `SELECT mime,duration_ms,generated FROM public.app_prosody_assets WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR SHARE`, p.WorkspaceID, segment.AssetID).Scan(&mime, &duration, &generated)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUnavailable
			}
			if err != nil {
				return fmt.Errorf("lock playback asset: %w", err)
			}
			if duration <= 0 || duration != segment.DurationMS {
				return ErrInvalid
			}
			switch segment.Kind {
			case "narration":
				if !generated || mime != "audio/mpeg" || duration > 600000-total {
					return ErrInvalid
				}
				total += duration
			case "audio":
				if generated || mime != "audio/mpeg" {
					return ErrInvalid
				}
			case "video":
				if generated || mime != "video/mp4" {
					return ErrInvalid
				}
			default:
				return ErrInvalid
			}
		} else if segment.AssetID != "" || segment.DurationMS != 0 || len(segment.ImageIDs) == 0 {
			return ErrInvalid
		}
		for _, id := range segment.ImageIDs {
			if !UUID(id) {
				return ErrInvalid
			}
			var mime string
			err = tx.QueryRowContext(ctx, `SELECT mime FROM public.app_prosody_assets WHERE workspace_id=$1 AND id=$2 AND generated=false AND deleted_at IS NULL FOR SHARE`, p.WorkspaceID, id).Scan(&mime)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUnavailable
			}
			if err != nil {
				return fmt.Errorf("lock image: %w", err)
			}
			if mime != "image/jpeg" && mime != "image/png" {
				return ErrInvalid
			}
		}
	}
	if total > 0 && len(p.Script) == 0 {
		return ErrInvalid
	}
	manifest, err := json.Marshal(p.Segments)
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	inserted, err := tx.ExecContext(ctx, `INSERT INTO public.app_prosody_renditions(workspace_id,item_id,id,settings_revision,manifest_bytes,script_bytes,provenance_bytes,narration_ms) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, p.WorkspaceID, p.ItemID, p.RenditionID, p.SettingsRevision, manifest, p.Script, p.Provenance, total)
	if err != nil {
		return fmt.Errorf("persist rendition: %w", err)
	}
	if err = onePublicationRow(inserted); err != nil {
		return err
	}
	updated, err := tx.ExecContext(ctx, `UPDATE public.app_prosody_submissions SET status='ready' WHERE workspace_id=$1 AND id=$2`, p.WorkspaceID, p.ItemID)
	if err != nil {
		return fmt.Errorf("publish item: %w", err)
	}
	if err = onePublicationRow(updated); err != nil {
		return err
	}
	if err = p.StageReady(ctx, p.ItemID, p.RenditionID); err != nil {
		return fmt.Errorf("stage ready notification: %w", err)
	}
	return nil
}

// Tombstone denies subsequent reads and publication while retaining the key and
// digest needed to reconcile retries. Cleanup must use AMOS's durable job path.
func Tombstone(ctx context.Context, tx Transaction, workspace, item string, stageCleanup func(context.Context, string) error) error {
	if !tx.valid() || !UUID(workspace) || !UUID(item) || stageCleanup == nil {
		return ErrInvalid
	}
	var id string
	err := tx.QueryRowContext(ctx, `UPDATE public.app_prosody_submissions SET status='deleted',deleted_at=clock_timestamp() WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL RETURNING id`, workspace, item).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnavailable
	}
	if err != nil {
		return fmt.Errorf("tombstone source: %w", err)
	}
	if err = stageCleanup(ctx, id); err != nil {
		return fmt.Errorf("stage deletion: %w", err)
	}
	return nil
}

// DurationCeilingMS rounds up so serialization cannot undercount generated audio.
func DurationCeilingMS(nanoseconds int64) (int64, error) {
	if nanoseconds <= 0 || nanoseconds > math.MaxInt64-999999 {
		return 0, ErrInvalid
	}
	return (nanoseconds + 999999) / 1000000, nil
}

func onePublicationRow(result sql.Result) error {
	if result == nil {
		return ErrInvalid
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count publication rows: %w", err)
	}
	if count != 1 {
		return ErrConflict
	}
	return nil
}
