package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/narrate-it/prosody/internal/playback"
)

// ProgressKey is resolved from the authorized human session, never an agent.
type ProgressKey struct{ WorkspaceID, PersonID, ItemID, RenditionID string }

func (k ProgressKey) valid() bool {
	return UUID(k.WorkspaceID) && UUID(k.PersonID) && UUID(k.ItemID) && UUID(k.RenditionID)
}

// UpdateProgress serializes one person's updates and rejects stale revisions.
// Durations come from the immutable published manifest, not from client input.
func UpdateProgress(ctx context.Context, tx Transaction, k ProgressKey, u playback.Update) (playback.Progress, error) {
	if !tx.valid() || !k.valid() {
		return playback.Progress{}, ErrInvalid
	}
	var initial playback.Progress
	var manifest []byte
	// Lock parent against tombstoning for the entire progress transaction.
	var id string
	err := tx.QueryRowContext(ctx, `SELECT s.id,r.manifest_bytes FROM public.app_prosody_submissions s JOIN public.app_prosody_renditions r ON r.workspace_id=s.workspace_id AND r.item_id=s.id WHERE s.workspace_id=$1 AND s.id=$2 AND r.id=$3 AND s.status='ready' AND s.deleted_at IS NULL FOR SHARE OF s`, k.WorkspaceID, k.ItemID, k.RenditionID).Scan(&id, &manifest)
	if errors.Is(err, sql.ErrNoRows) {
		return initial, ErrUnavailable
	}
	if err != nil {
		return initial, fmt.Errorf("lock progress item: %w", err)
	}
	var segments []PublishedSegment
	if err = json.Unmarshal(manifest, &segments); err != nil {
		return initial, fmt.Errorf("decode rendition durations: %w", err)
	}
	durations := make([]int64, len(segments))
	for i, segment := range segments {
		durations[i] = segment.DurationMS
	}
	initial, err = playback.New(durations)
	if err != nil {
		return initial, err
	}
	b, err := json.Marshal(initial)
	if err != nil {
		return initial, fmt.Errorf("encode initial progress: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO public.app_prosody_progress(workspace_id,person_id,item_id,rendition_id,revision,value_bytes) VALUES($1,$2,$3,$4,0,$5) ON CONFLICT DO NOTHING`, k.WorkspaceID, k.PersonID, k.ItemID, k.RenditionID, b)
	if err != nil {
		return initial, fmt.Errorf("initialize progress: %w", err)
	}
	err = tx.QueryRowContext(ctx, `SELECT value_bytes FROM public.app_prosody_progress WHERE workspace_id=$1 AND person_id=$2 AND item_id=$3 AND rendition_id=$4 FOR UPDATE`, k.WorkspaceID, k.PersonID, k.ItemID, k.RenditionID).Scan(&b)
	if err != nil {
		return initial, fmt.Errorf("read progress: %w", err)
	}
	var current playback.Progress
	if err = json.Unmarshal(b, &current); err != nil {
		return initial, fmt.Errorf("decode progress: %w", err)
	}
	if len(current.Segments) != len(durations) {
		return initial, ErrConflict
	}
	for i, d := range durations {
		if current.Segments[i].DurationMS != d {
			return initial, ErrConflict
		}
	}
	next, err := playback.Apply(current, u)
	if err != nil {
		return initial, err
	}
	if next.Revision > math.MaxInt64 {
		return initial, ErrInvalid
	}
	b, err = json.Marshal(next)
	if err != nil {
		return initial, fmt.Errorf("encode progress: %w", err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE public.app_prosody_progress SET revision=$5,value_bytes=$6,updated_at=clock_timestamp() WHERE workspace_id=$1 AND person_id=$2 AND item_id=$3 AND rendition_id=$4`, k.WorkspaceID, k.PersonID, k.ItemID, k.RenditionID, next.Revision, b)
	if err != nil {
		return initial, fmt.Errorf("persist progress: %w", err)
	}
	return next, nil
}
