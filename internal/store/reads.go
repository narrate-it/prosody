package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/narrate-it/prosody/internal/content"
	"github.com/narrate-it/prosody/internal/strictjson"
)

// ReadRows is the callback-facing result set, compatible with AMOS Rows through
// a host adapter. No transaction lifecycle or unrestricted database is exposed.
type ReadRows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}

// ReadTransaction must wrap the caller's currently authorized transaction.
// Workspace parameters constrain SQL; they are not proof of current authority.
// The caller must admit human consumption and hold authority through the read.
type ReadTransaction struct {
	QueryContext    func(context.Context, string, ...any) (ReadRows, error)
	QueryRowContext func(context.Context, string, ...any) Row
}

func (t ReadTransaction) valid() bool { return t.QueryContext != nil && t.QueryRowContext != nil }

// FromSQLRead adapts an already-open transaction without exposing its lifecycle.
func FromSQLRead(tx *sql.Tx) ReadTransaction {
	if tx == nil {
		return ReadTransaction{}
	}
	return ReadTransaction{
		QueryContext: func(ctx context.Context, query string, args ...any) (ReadRows, error) {
			return tx.QueryContext(ctx, query, args...)
		},
		QueryRowContext: func(ctx context.Context, query string, args ...any) Row {
			return tx.QueryRowContext(ctx, query, args...)
		},
	}
}

const maxReadPage = 50

// FeedSummary intentionally has no key, provider settings, role or permission
// claims. A host may add canManage only from its current authorization decision.
type FeedSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type FeedReadPage struct {
	Feeds      []FeedSummary `json:"feeds"`
	NextCursor string        `json:"nextCursor,omitempty"`
}

type feedCursor struct {
	Version   int    `json:"v"`
	Workspace string `json:"w"`
	AfterID   string `json:"after"`
}

// ReadFeeds lists only nondeleted feeds in one currently authorized workspace.
// A caller enumerating several workspaces must authorize each separately. The
// cursor is a scope-bound location, not a signed access capability.
func ReadFeeds(ctx context.Context, tx ReadTransaction, workspace, cursor string, limit int) (FeedReadPage, error) {
	page := FeedReadPage{Feeds: []FeedSummary{}}
	if !tx.valid() || !UUID(workspace) || limit < 1 || limit > maxReadPage {
		return page, ErrInvalid
	}
	after := "00000000-0000-0000-0000-000000000000"
	if cursor != "" {
		var c feedCursor
		if decodeReadCursor(cursor, &c) != nil || c.Version != 1 || c.Workspace != workspace || !FeedID(c.AfterID) {
			return page, ErrInvalid
		}
		after = c.AfterID
	}
	rows, err := tx.QueryContext(ctx, readFeedsSQL, workspace, after, limit+1)
	if err != nil {
		return page, fmt.Errorf("read workspace feeds: %w", err)
	}
	// Success closes explicitly below; cleanup must not replace a read error.
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var f FeedSummary
		var title []byte
		if err = rows.Scan(&f.ID, &f.Kind, &title); err != nil {
			return FeedReadPage{}, fmt.Errorf("scan feed: %w", err)
		}
		if !FeedID(f.ID) || (f.Kind != "personal" && f.Kind != "team") || len(title) == 0 || !utf8.Valid(title) {
			return FeedReadPage{}, ErrInvalid
		}
		f.Name = string(title)
		page.Feeds = append(page.Feeds, f)
	}
	if err = rows.Err(); err != nil {
		return FeedReadPage{}, fmt.Errorf("iterate feeds: %w", err)
	}
	if err = rows.Close(); err != nil {
		return FeedReadPage{}, fmt.Errorf("close feed results: %w", err)
	}
	if len(page.Feeds) > limit {
		page.Feeds = page.Feeds[:limit]
		page.NextCursor, err = encodeReadCursor(feedCursor{Version: 1, Workspace: workspace, AfterID: page.Feeds[limit-1].ID})
		if err != nil {
			return FeedReadPage{}, err
		}
	}
	return page, nil
}

func encodeReadCursor(value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode read cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeReadCursor(value string, out any) error {
	if len(value) == 0 || len(value) > 2048 {
		return ErrInvalid
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || strictjson.Validate(b) != nil {
		return ErrInvalid
	}
	// Cursors are small closed objects. Reject duplicate decoded field names
	// before struct decoding, rather than allowing last-field-wins ambiguity.
	keys := json.NewDecoder(strings.NewReader(string(b)))
	token, err := keys.Token()
	if err != nil || token != json.Delim('{') {
		return ErrInvalid
	}
	allowed := map[string]bool{"v": true, "w": true}
	switch out.(type) {
	case *feedCursor:
		allowed["after"] = true
	case *briefingCursor:
		for _, key := range []string{"f", "cutoff", "at", "item", "rendition"} {
			allowed[key] = true
		}
	default:
		return ErrInvalid
	}
	seen := map[string]bool{}
	for keys.More() {
		token, err = keys.Token()
		if err != nil {
			return ErrInvalid
		}
		key, ok := token.(string)
		if !ok || !allowed[key] || seen[key] {
			return ErrInvalid
		}
		seen[key] = true
		var value json.RawMessage
		if keys.Decode(&value) != nil || string(value) == "null" {
			return ErrInvalid
		}
	}
	// Every declared field is required; no caller-populated value may stand in
	// for an omitted cursor field. Unknown and duplicate fields were rejected above.
	if len(seen) != len(allowed) {
		return ErrInvalid
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return ErrInvalid
	}
	if _, err = d.Token(); err != io.EOF {
		return ErrInvalid
	}
	return nil
}

var ErrAmbiguousRendition = errors.New("item has more than one published rendition")

// ReadyBriefing is a consumption projection: source text and published script,
// with private asset IDs for separately authorized delivery. It never exposes
// idempotency keys, authority references, work IDs, credentials or settings.
type ReadyBriefing struct {
	ID          string             `json:"id"`
	RenditionID string             `json:"renditionId"`
	Title       string             `json:"title"`
	Status      string             `json:"status"`
	Source      string             `json:"source"`
	Script      string             `json:"script"`
	Segments    []PublishedSegment `json:"segments"`
	PublishedAt time.Time          `json:"publishedAt"`
}

type BriefingReadPage struct {
	Items      []ReadyBriefing `json:"items"`
	NextCursor string          `json:"nextCursor,omitempty"`
}

type briefingCursor struct {
	Version     int       `json:"v"`
	Workspace   string    `json:"w"`
	Feed        string    `json:"f"`
	Cutoff      time.Time `json:"cutoff"`
	PublishedAt time.Time `json:"at"`
	ItemID      string    `json:"item"`
	RenditionID string    `json:"rendition"`
}

// ReadReadyBriefings lists v1's single published rendition per ready item, in
// ascending publication order with deterministic identity tie-breakers. It
// rejects ambiguous multiple renditions rather than inventing a current version.
//
// The cursor cutoff bounds publication timestamps; it is NOT a database snapshot
// across requests. A later commit/backdated publication may require refreshing
// from an empty cursor. This method does not promise gap-free arrival delivery.
// Existing cursor results are not retained content access grants: every call
// requires current workspace/feed authority and excludes tombstones afresh.
// Returned source rows stay share-locked until the caller closes its transaction;
// this is no guarantee about response delivery after transaction/authority release.
func ReadReadyBriefings(ctx context.Context, tx ReadTransaction, workspace, feed, cursor string, limit int) (BriefingReadPage, error) {
	page := BriefingReadPage{Items: []ReadyBriefing{}}
	if !tx.valid() || !UUID(workspace) || !FeedID(feed) || limit < 1 || limit > maxReadPage {
		return page, ErrInvalid
	}
	var c briefingCursor
	if cursor != "" {
		if decodeReadCursor(cursor, &c) != nil || c.Version != 1 || c.Workspace != workspace || c.Feed != feed || !UUID(c.ItemID) || !UUID(c.RenditionID) || c.Cutoff.IsZero() || c.PublishedAt.IsZero() || c.PublishedAt.After(c.Cutoff) {
			return page, ErrInvalid
		}
	} else {
		c = briefingCursor{Version: 1, Workspace: workspace, Feed: feed, ItemID: "00000000-0000-0000-0000-000000000000", RenditionID: "00000000-0000-0000-0000-000000000000"}
	}
	if err := lockReadableFeed(ctx, tx, workspace, feed); err != nil {
		return page, err
	}
	var now time.Time
	if err := tx.QueryRowContext(ctx, readClockSQL).Scan(&now); err != nil {
		return page, fmt.Errorf("read publication cutoff: %w", err)
	}
	if cursor == "" {
		c.Cutoff = now
	} else if c.Cutoff.After(now) {
		return page, ErrInvalid
	}
	rows, err := tx.QueryContext(ctx, readReadyBriefingsSQL, workspace, feed, c.Cutoff, cursor != "", c.PublishedAt, c.ItemID, c.RenditionID, limit+1)
	if err != nil {
		return page, fmt.Errorf("read ready briefings: %w", err)
	}
	// Success closes explicitly below; cleanup must not replace a read error.
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		briefing, err := scanReadyBriefing(rows, workspace, feed)
		if err != nil {
			return BriefingReadPage{}, err
		}
		page.Items = append(page.Items, briefing)
	}
	if err = rows.Err(); err != nil {
		return BriefingReadPage{}, fmt.Errorf("iterate ready briefings: %w", err)
	}
	if err = rows.Close(); err != nil {
		return BriefingReadPage{}, fmt.Errorf("close briefing results: %w", err)
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[limit-1]
		c.PublishedAt, c.ItemID, c.RenditionID = last.PublishedAt, last.ID, last.RenditionID
		page.NextCursor, err = encodeReadCursor(c)
		if err != nil {
			return BriefingReadPage{}, err
		}
	}
	return page, nil
}

// ReadReadyBriefing retrieves one currently readable ready item from the selected
// feed. Processing, failed, deleted, cross-feed and cross-workspace records all
// return ErrUnavailable; no partial source or private submission record leaks.
func ReadReadyBriefing(ctx context.Context, tx ReadTransaction, workspace, feed, item string) (ReadyBriefing, error) {
	if !tx.valid() || !UUID(workspace) || !FeedID(feed) || !UUID(item) {
		return ReadyBriefing{}, ErrInvalid
	}
	if err := lockReadableFeed(ctx, tx, workspace, feed); err != nil {
		return ReadyBriefing{}, err
	}
	row := tx.QueryRowContext(ctx, readReadyBriefingSQL, workspace, feed, item)
	return scanReadyBriefing(row, workspace, feed)
}

func lockReadableFeed(ctx context.Context, tx ReadTransaction, workspace, feed string) error {
	var id string
	err := tx.QueryRowContext(ctx, readableFeedSQL, workspace, feed).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnavailable
	}
	if err != nil {
		return fmt.Errorf("lock readable feed: %w", err)
	}
	return nil
}

func scanReadyBriefing(row Row, workspace, feed string) (ReadyBriefing, error) {
	var result ReadyBriefing
	var source, script, manifest []byte
	var renditions int64
	err := row.Scan(&result.ID, &result.RenditionID, &result.PublishedAt, &source, &script, &manifest, &renditions)
	if errors.Is(err, sql.ErrNoRows) {
		return result, ErrUnavailable
	}
	if err != nil {
		return result, fmt.Errorf("scan ready briefing: %w", err)
	}
	if renditions != 1 {
		return ReadyBriefing{}, ErrAmbiguousRendition
	}
	if !UUID(result.ID) || !UUID(result.RenditionID) || result.PublishedAt.IsZero() || script == nil || !utf8.Valid(script) || len(manifest) == 0 || strictjson.Validate(manifest) != nil {
		return ReadyBriefing{}, ErrInvalid
	}
	submission, err := content.DecodeText(strings.NewReader(string(source)))
	if err != nil || submission.WorkspaceID != workspace || submission.FeedID != feed {
		return ReadyBriefing{}, ErrInvalid
	}
	d := json.NewDecoder(strings.NewReader(string(manifest)))
	d.DisallowUnknownFields()
	if d.Decode(&result.Segments) != nil || len(result.Segments) == 0 || len(result.Segments) > 100 {
		return ReadyBriefing{}, ErrInvalid
	}
	seen := map[string]bool{}
	var narrationMS int64
	for _, segment := range result.Segments {
		if !UUID(segment.ID) || seen[segment.ID] || len(segment.ImageIDs) > 20 || segment.DurationMS < 0 {
			return ReadyBriefing{}, ErrInvalid
		}
		seen[segment.ID] = true
		if segment.Kind == "image" {
			if segment.AssetID != "" || segment.DurationMS != 0 || len(segment.ImageIDs) == 0 {
				return ReadyBriefing{}, ErrInvalid
			}
		} else if (segment.Kind != "narration" && segment.Kind != "audio" && segment.Kind != "video") || !UUID(segment.AssetID) || segment.DurationMS == 0 {
			return ReadyBriefing{}, ErrInvalid
		}
		if segment.Kind == "narration" {
			if segment.DurationMS > 600000-narrationMS {
				return ReadyBriefing{}, ErrInvalid
			}
			narrationMS += segment.DurationMS
		}
		for _, image := range segment.ImageIDs {
			if !UUID(image) {
				return ReadyBriefing{}, ErrInvalid
			}
		}
	}
	if narrationMS > 0 && len(script) == 0 {
		return ReadyBriefing{}, ErrInvalid
	}
	result.Title, result.Source, result.Script, result.Status = submission.Title, submission.Text, string(script), "ready"
	return result, nil
}
