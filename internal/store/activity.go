package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/narrate-it/prosody/internal/content"
	"github.com/narrate-it/prosody/internal/strictjson"
)

// SubmissionSummary exposes immutable source and current preparation status.
// Playback, private submission metadata and unsupported failure reasons are absent.
type SubmissionSummary struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Source     string    `json:"source"`
	Status     string    `json:"status"`
	AcceptedAt time.Time `json:"acceptedAt"`
}

type ActivityReadPage struct {
	Items      []SubmissionSummary `json:"items"`
	NextCursor string              `json:"nextCursor,omitempty"`
}

type activityCursor struct {
	Version    int       `json:"v"`
	Workspace  string    `json:"w"`
	Feed       string    `json:"f"`
	Cutoff     time.Time `json:"cutoff"`
	AcceptedAt time.Time `json:"at"`
	ItemID     string    `json:"item"`
}

// ReadActivity lists source summaries oldest first, ordered by acceptance time
// and item ID. The caller must supply a currently authorized transaction and
// hold authority through the read. Workspace/feed parameters are not authority.
// Returned source rows and the feed remain share-locked until that transaction
// closes. A cursor cutoff is a pagination bound, not a cross-transaction snapshot;
// late commits, status changes and backdated arrivals require a refresh.
func ReadActivity(ctx context.Context, tx ReadTransaction, workspace, feed, cursor string, limit int) (ActivityReadPage, error) {
	page := ActivityReadPage{Items: []SubmissionSummary{}}
	if !tx.valid() || !UUID(workspace) || !FeedID(feed) || limit < 1 || limit > maxReadPage {
		return page, ErrInvalid
	}
	var c activityCursor
	if cursor != "" {
		if decodeActivityCursor(cursor, &c) != nil || c.Version != 1 || c.Workspace != workspace || c.Feed != feed || !UUID(c.ItemID) || c.Cutoff.IsZero() || c.AcceptedAt.IsZero() || c.AcceptedAt.After(c.Cutoff) {
			return page, ErrInvalid
		}
	} else {
		c = activityCursor{Version: 1, Workspace: workspace, Feed: feed, ItemID: "00000000-0000-0000-0000-000000000000"}
	}
	if err := lockReadableFeed(ctx, tx, workspace, feed); err != nil {
		return page, err
	}
	var now time.Time
	if err := tx.QueryRowContext(ctx, `SELECT pg_catalog.clock_timestamp()`).Scan(&now); err != nil {
		return page, fmt.Errorf("read activity cutoff: %w", err)
	}
	if cursor == "" {
		c.Cutoff = now
	} else if c.Cutoff.After(now) {
		return page, ErrInvalid
	}
	rows, err := tx.QueryContext(ctx, `SELECT s.id,s.status,s.accepted_at,
 CASE WHEN octet_length(s.source_bytes)<=65536 THEN s.source_bytes ELSE NULL END
 FROM public.app_prosody_submissions s
 WHERE s.workspace_id=$1 AND s.feed_id=$2 AND s.deleted_at IS NULL
 AND s.status IN ('processing','failed','ready') AND s.accepted_at<=$3
 AND ($4::boolean=false OR (s.accepted_at,s.id)>($5::timestamptz,$6::uuid))
 ORDER BY s.accepted_at,s.id LIMIT $7 FOR SHARE OF s`, workspace, feed, c.Cutoff, cursor != "", c.AcceptedAt, c.ItemID, limit+1)
	if err != nil {
		return page, fmt.Errorf("read activity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item SubmissionSummary
		var source []byte
		if err := rows.Scan(&item.ID, &item.Status, &item.AcceptedAt, &source); err != nil {
			return ActivityReadPage{}, fmt.Errorf("scan activity: %w", err)
		}
		if !UUID(item.ID) || item.AcceptedAt.IsZero() || (item.Status != "processing" && item.Status != "failed" && item.Status != "ready") {
			return ActivityReadPage{}, ErrInvalid
		}
		submission, err := content.DecodeText(strings.NewReader(string(source)))
		if err != nil || submission.WorkspaceID != workspace || submission.FeedID != feed {
			return ActivityReadPage{}, ErrInvalid
		}
		item.Title, item.Source = submission.Title, submission.Text
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ActivityReadPage{}, fmt.Errorf("iterate activity: %w", err)
	}
	if err := rows.Close(); err != nil {
		return ActivityReadPage{}, fmt.Errorf("close activity: %w", err)
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[limit-1]
		c.AcceptedAt, c.ItemID = last.AcceptedAt, last.ID
		page.NextCursor, err = encodeReadCursor(c)
		if err != nil {
			return ActivityReadPage{}, err
		}
	}
	return page, nil
}

// Keep this closed inventory separate from the existing feed/playback cursors.
// All fields must be present exactly once; decoding cannot inherit request scope.
func decodeActivityCursor(value string, out *activityCursor) error {
	if out == nil || len(value) == 0 || len(value) > 2048 {
		return ErrInvalid
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || strictjson.Validate(b) != nil {
		return ErrInvalid
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return ErrInvalid
	}
	allowed := map[string]bool{"v": true, "w": true, "f": true, "cutoff": true, "at": true, "item": true}
	seen := map[string]bool{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return ErrInvalid
		}
		key, ok := token.(string)
		if !ok || !allowed[key] || seen[key] {
			return ErrInvalid
		}
		seen[key] = true
		var value json.RawMessage
		if d.Decode(&value) != nil || string(value) == "null" {
			return ErrInvalid
		}
	}
	if len(seen) != len(allowed) {
		return ErrInvalid
	}
	var decoded activityCursor
	d = json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if d.Decode(&decoded) != nil {
		return ErrInvalid
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrInvalid
	}
	*out = decoded
	return nil
}
