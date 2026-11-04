package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const (
	preferenceReadFeedSQL  = `SELECT id FROM public.app_prosody_feeds WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR SHARE`
	preferenceWriteFeedSQL = `SELECT id FROM public.app_prosody_feeds WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE`
	preferenceReadSQL      = `SELECT notifications FROM public.app_prosody_preferences WHERE workspace_id=$1 AND feed_id=$2 AND person_id=$3 FOR SHARE`
	preferenceWriteSQL     = `INSERT INTO public.app_prosody_preferences(workspace_id,feed_id,person_id,notifications) VALUES($1,$2,$3,$4) ON CONFLICT(workspace_id,feed_id,person_id) DO UPDATE SET notifications=EXCLUDED.notifications`
)

// NotificationPreferenceKey is resolved from the currently authorized human
// operation. IDs are descriptive bindings, not authority or membership proof.
type NotificationPreferenceKey struct{ WorkspaceID, FeedID, PersonID string }

// valid checks UUID syntax and feed version, not current authority or membership.
func (k NotificationPreferenceKey) valid() bool {
	return UUID(k.WorkspaceID) && FeedID(k.FeedID) && UUID(k.PersonID)
}

// ReadNotificationPreference returns false when no preference row exists, only
// after establishing a live scoped feed. It never creates a row. Feed and any
// existing preference row remain share-locked until the caller transaction ends.
// Current person/workspace/feed admission remains the caller's responsibility.
func ReadNotificationPreference(ctx context.Context, tx ReadTransaction, k NotificationPreferenceKey) (bool, error) {
	if !tx.valid() || !k.valid() {
		return false, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := lockPreferenceFeed(ctx, tx.QueryRowContext, preferenceReadFeedSQL, k); err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	var enabled bool
	err := tx.QueryRowContext(ctx, preferenceReadSQL, k.WorkspaceID, k.FeedID, k.PersonID).Scan(&enabled)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("read notification preference: %w", err)
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return false, ctxErr
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return enabled, nil
}

// SetNotificationPreference writes only this person's feed choice, including an
// explicit false. FOR UPDATE on the feed serializes these writes and conflicts
// with read-side feed locks even for the absent/default-false preference. This
// also excludes concurrent tombstoning. The caller owns commit/rollback and must
// roll back on any error; affected-row proof is not a committed receipt.
// Cancellation after a callback may follow an executed write: it does not prove
// rollback/no effect. The host must reconcile unknown transaction completion.
func SetNotificationPreference(ctx context.Context, tx Transaction, k NotificationPreferenceKey, enabled bool) error {
	if !tx.valid() || !k.valid() {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := lockPreferenceFeed(ctx, tx.QueryRowContext, preferenceWriteFeedSQL, k); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, preferenceWriteSQL, k.WorkspaceID, k.FeedID, k.PersonID, enabled)
	if err != nil {
		return fmt.Errorf("write notification preference: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if result == nil {
		return ErrInvalid
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count notification preference rows: %w", err)
	}
	if count != 1 {
		return ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

// lockPreferenceFeed requests the selected live-feed lock and rejects missing
// or mismatched feed projections before any preference access.
func lockPreferenceFeed(ctx context.Context, query func(context.Context, string, ...any) Row, statement string, k NotificationPreferenceKey) error {
	var feed string
	err := query(ctx, statement, k.WorkspaceID, k.FeedID).Scan(&feed)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnavailable
	}
	if err != nil {
		return fmt.Errorf("lock notification preference feed: %w", err)
	}
	if feed != k.FeedID {
		return ErrUnavailable
	}
	return nil
}
