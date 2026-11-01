package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestReadCursorRejectsMalformedAndCrossScope(t *testing.T) {
	ctx := context.Background()
	never := ReadTransaction{
		QueryContext: func(context.Context, string, ...any) (ReadRows, error) {
			t.Fatal("invalid cursor reached SQL")
			return nil, ErrInvalid
		},
		QueryRowContext: func(context.Context, string, ...any) Row {
			t.Fatal("invalid cursor reached SQL")
			return nil
		},
	}
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	c := briefingCursor{Version: 1, Workspace: workspace, Feed: feed, Cutoff: now, PublishedAt: now.Add(-time.Minute), ItemID: item, RenditionID: work}
	valid, err := encodeReadCursor(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, cursor, workspace, feed string }{
		{"invalid encoding", "%%%", workspace, feed},
		{"oversize", strings.Repeat("a", 2049), workspace, feed},
		{"wrong workspace", valid, installation, feed},
		{"wrong feed", valid, workspace, item},
		{"duplicate field", base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"v":1}`)), workspace, feed},
		{"unknown field", base64.RawURLEncoding.EncodeToString([]byte(`{"extra":true}`)), workspace, feed},
		{"null field", base64.RawURLEncoding.EncodeToString([]byte(`{"v":null}`)), workspace, feed},
		{"not object", base64.RawURLEncoding.EncodeToString([]byte(`[]`)), workspace, feed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ReadReadyBriefings(ctx, never, tc.workspace, tc.feed, tc.cursor, 2); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid cursor, got %v", err)
			}
		})
	}
	feedToken, err := encodeReadCursor(feedCursor{Version: 1, Workspace: workspace, AfterID: feed})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ReadFeeds(ctx, never, installation, feedToken, 2); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-workspace feed cursor: %v", err)
	}
}

// insertReadFixture deliberately seeds persisted ready metadata directly. It
// exercises consumption SQL, not publication admission, codecs or job finality.
func insertReadFixture(t *testing.T, db *sql.DB, base Acceptance, at time.Time, status string) (string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a := base
	a.ItemID = testID(t)
	a.Submission.IdempotencyKey = "private-key-" + a.ItemID
	a.Submission.Title = "Source title"
	a.Submission.Text = "Original\x00source retained."
	a.AuthorityReference = []byte("private-authority-reference")
	a.SettingsSnapshot = []byte(`{"privateSettingsReference":true}`)
	workID := testID(t)
	a.StageWork = func(context.Context, string) (string, error) { return workID, nil }
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, _, err = Accept(ctx, FromSQL(tx), a); err != nil {
		t.Fatal(err)
	}
	rid := testID(t)
	if status != "processing" {
		manifest, err := json.Marshal([]PublishedSegment{{ID: testID(t), Kind: "image", ImageIDs: []string{testID(t)}}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO public.app_prosody_renditions(workspace_id,item_id,id,settings_revision,manifest_bytes,script_bytes,provenance_bytes,narration_ms,published_at) VALUES($1,$2,$3,1,$4,$5,$6,0,$7)`, a.Scope.WorkspaceID, a.ItemID, rid, manifest, []byte("Condensed script."), []byte("private-provenance"), at); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE public.app_prosody_submissions SET status=$2,deleted_at=CASE WHEN $2='deleted' THEN clock_timestamp() ELSE NULL END WHERE id=$1`, a.ItemID, status); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return a.ItemID, rid
}

func TestReadReadyPaginationAndSecretProjection(t *testing.T) {
	db, _ := openFixture(t)
	a := testAcceptance(t, db)
	foreign := testAcceptance(t, db)
	at := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	ids := make([]string, 0, 3)
	for range 3 {
		id, _ := insertReadFixture(t, db, a, at, "ready")
		ids = append(ids, id)
	}
	sort.Strings(ids)
	insertReadFixture(t, db, a, at, "processing")
	insertReadFixture(t, db, a, at, "deleted")
	foreignID, _ := insertReadFixture(t, db, foreign, at, "ready")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	page, err := ReadReadyBriefings(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, a.Submission.FeedID, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.NextCursor == "" || page.Items[0].ID != ids[0] || page.Items[1].ID != ids[1] {
		t.Fatalf("unstable first page: %+v", page)
	}
	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-key-", "private-authority-reference", "privateSettingsReference", "private-provenance", "idempotencyKey", "workId", "credential"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("private field leaked: %s", secret)
		}
	}
	if page.Items[0].Source != "Original\x00source retained." || page.Items[0].Script != "Condensed script." {
		t.Fatal("source/script not preserved separately")
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var c briefingCursor
	if err = decodeReadCursor(page.NextCursor, &c); err != nil {
		t.Fatal(err)
	}
	newID, _ := insertReadFixture(t, db, a, c.Cutoff.Add(time.Microsecond), "ready")
	nextTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer nextTx.Rollback()
	next, err := ReadReadyBriefings(ctx, FromSQLRead(nextTx), a.Scope.WorkspaceID, a.Submission.FeedID, page.NextCursor, 2)
	if err != nil || len(next.Items) != 1 || next.Items[0].ID != ids[2] || next.NextCursor != "" {
		t.Fatalf("next page=%+v err=%v", next, err)
	}
	refreshed, err := ReadReadyBriefings(ctx, FromSQLRead(nextTx), a.Scope.WorkspaceID, a.Submission.FeedID, "", 10)
	if err != nil || len(refreshed.Items) != 4 {
		t.Fatalf("refresh page=%+v err=%v", refreshed, err)
	}
	if refreshed.Items[3].ID != newID {
		t.Fatal("new publication did not append by timestamp")
	}
	if _, err = ReadReadyBriefing(ctx, FromSQLRead(nextTx), a.Scope.WorkspaceID, a.Submission.FeedID, foreignID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("cross-workspace item: %v", err)
	}
}

func TestReadDeniesDeletedAndAmbiguousItems(t *testing.T) {
	db, _ := openFixture(t)
	a := testAcceptance(t, db)
	at := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	id, rid := insertReadFixture(t, db, a, at, "ready")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = ReadReadyBriefing(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, a.Submission.FeedID, id); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO public.app_prosody_renditions(workspace_id,item_id,id,settings_revision,manifest_bytes,script_bytes,provenance_bytes,narration_ms,published_at) SELECT workspace_id,item_id,$3,settings_revision,manifest_bytes,script_bytes,provenance_bytes,narration_ms,published_at FROM public.app_prosody_renditions WHERE workspace_id=$1 AND id=$2`, a.Scope.WorkspaceID, rid, testID(t)); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadReadyBriefing(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, a.Submission.FeedID, id); !errors.Is(err, ErrAmbiguousRendition) {
		t.Fatalf("ambiguous detail: %v", err)
	}
	if _, err = ReadReadyBriefings(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, a.Submission.FeedID, "", 2); !errors.Is(err, ErrAmbiguousRendition) {
		t.Fatalf("ambiguous feed: %v", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE public.app_prosody_submissions SET status='deleted',deleted_at=clock_timestamp() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadReadyBriefing(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, a.Submission.FeedID, id); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("deleted detail: %v", err)
	}
	page, err := ReadReadyBriefings(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, a.Submission.FeedID, "", 2)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("deleted item visible: %+v %v", page, err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE public.app_prosody_feeds SET deleted_at=clock_timestamp() WHERE workspace_id=$1 AND id=$2`, a.Scope.WorkspaceID, a.Submission.FeedID); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadReadyBriefings(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, a.Submission.FeedID, "", 2); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("deleted feed readable: %v", err)
	}
	feeds, err := ReadFeeds(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, "", 2)
	if err != nil || len(feeds.Feeds) != 0 {
		t.Fatalf("deleted feed listed: %+v %v", feeds, err)
	}
}

func TestReadFeedsWorkspaceAndCursor(t *testing.T) {
	db, _ := openFixture(t)
	a := testAcceptance(t, db)
	other := testAcceptance(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	second := testID(t)
	if _, err = tx.ExecContext(ctx, `INSERT INTO public.app_prosody_feeds(workspace_id,id,kind,title) VALUES($1,$2,'team',$3)`, a.Scope.WorkspaceID, second, []byte("Team\x00name")); err != nil {
		t.Fatal(err)
	}
	page, err := ReadFeeds(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, "", 1)
	if err != nil || len(page.Feeds) != 1 || page.NextCursor == "" {
		t.Fatalf("feed page=%+v err=%v", page, err)
	}
	next, err := ReadFeeds(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, page.NextCursor, 1)
	if err != nil || len(next.Feeds) != 1 || next.NextCursor != "" || next.Feeds[0].ID == page.Feeds[0].ID {
		t.Fatalf("next feed page=%+v err=%v", next, err)
	}
	for _, f := range append(page.Feeds, next.Feeds...) {
		if f.ID == other.Submission.FeedID {
			t.Fatal("foreign feed leaked")
		}
	}
	if _, err = ReadFeeds(ctx, FromSQLRead(tx), other.Scope.WorkspaceID, page.NextCursor, 1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("foreign cursor accepted: %v", err)
	}
}

func TestReadCursorsRequireEveryField(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	feedValue := feedCursor{Version: 1, Workspace: workspace, AfterID: feed}
	briefingValue := briefingCursor{Version: 1, Workspace: workspace, Feed: feed, Cutoff: now, PublishedAt: now.Add(-time.Minute), ItemID: item, RenditionID: work}
	for _, tc := range []struct {
		name   string
		value  any
		fields []string
	}{
		{"feeds", feedValue, []string{"v", "w", "after"}},
		{"briefings", briefingValue, []string{"v", "w", "f", "cutoff", "at", "item", "rendition"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			for _, omitted := range tc.fields {
				t.Run("missing_"+omitted, func(t *testing.T) {
					var fields map[string]json.RawMessage
					if err := json.Unmarshal(body, &fields); err != nil {
						t.Fatal(err)
					}
					delete(fields, omitted)
					token, err := encodeReadCursor(fields)
					if err != nil {
						t.Fatal(err)
					}
					// Deliberately prepopulate every field: decoding must reject
					// omissions rather than accepting inherited request defaults.
					if tc.name == "feeds" {
						out := feedValue
						err = decodeReadCursor(token, &out)
					} else {
						out := briefingValue
						err = decodeReadCursor(token, &out)
					}
					if !errors.Is(err, ErrInvalid) {
						t.Fatalf("omitted %s accepted: %v", omitted, err)
					}
					never := ReadTransaction{
						QueryContext: func(context.Context, string, ...any) (ReadRows, error) {
							t.Fatal("incomplete cursor reached SQL")
							return nil, ErrInvalid
						},
						QueryRowContext: func(context.Context, string, ...any) Row {
							t.Fatal("incomplete cursor reached SQL")
							return nil
						},
					}
					if tc.name == "feeds" {
						_, err = ReadFeeds(context.Background(), never, workspace, token, 2)
					} else {
						_, err = ReadReadyBriefings(context.Background(), never, workspace, feed, token, 2)
					}
					if !errors.Is(err, ErrInvalid) {
						t.Fatalf("public read accepted omitted %s: %v", omitted, err)
					}
				})
			}
		})
	}
}

func TestReadyListHoldsTombstoneUntilTransactionCloses(t *testing.T) {
	db, _ := openFixture(t)
	a := testAcceptance(t, db)
	id, _ := insertReadFixture(t, db, a, time.Now().UTC().Add(-time.Hour), "ready")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	readTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer readTx.Rollback()
	page, err := ReadReadyBriefings(ctx, FromSQLRead(readTx), a.Scope.WorkspaceID, a.Submission.FeedID, "", 2)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("initial page=%+v err=%v", page, err)
	}

	writerPID := make(chan int, 1)
	finished := make(chan error, 1)
	go func() {
		defer close(finished)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			finished <- err
			return
		}
		defer tx.Rollback()
		var pid int
		if err = tx.QueryRowContext(ctx, `SELECT pg_catalog.pg_backend_pid()`).Scan(&pid); err != nil {
			finished <- err
			return
		}
		writerPID <- pid
		err = Tombstone(ctx, FromSQL(tx), a.Scope.WorkspaceID, id, func(context.Context, string) error { return nil })
		if err == nil {
			err = tx.Commit()
		}
		finished <- err
	}()
	// Always release/cancel and drain the bounded worker before fixture cleanup,
	// including assertion failures; no goroutine keeps a transaction alive.
	defer func() {
		cancel()
		if err := readTx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Errorf("release read transaction: %v", err)
		}
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("tombstone transaction did not exit after cancellation")
		}
	}()
	var pid int
	select {
	case pid = <-writerPID:
	case err = <-finished:
		t.Fatalf("writer exited before lock attempt: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	// Observe an actual PostgreSQL lock wait instead of inferring a fence from
	// scheduler timing or a sleep. The fixture user owns both connections.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting bool
		err = db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_stat_activity WHERE pid=$1 AND wait_event_type='Lock')`, pid).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case err = <-finished:
			t.Fatalf("tombstone passed the active read transaction: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
	if err = readTx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	fresh, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Rollback()
	page, err = ReadReadyBriefings(ctx, FromSQLRead(fresh), a.Scope.WorkspaceID, a.Submission.FeedID, "", 2)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("fresh read includes tombstone: %+v %v", page, err)
	}
}
