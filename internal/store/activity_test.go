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

func TestActivityCursorClosedScope(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	c := activityCursor{Version: 1, Workspace: workspace, Feed: feed, Cutoff: now, AcceptedAt: now, ItemID: item}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	never := ReadTransaction{
		QueryContext: func(context.Context, string, ...any) (ReadRows, error) {
			t.Fatal("invalid cursor reached SQL")
			return nil, ErrInvalid
		},
		QueryRowContext: func(context.Context, string, ...any) Row { t.Fatal("invalid cursor reached SQL"); return nil },
	}
	for key := range fields {
		t.Run("missing_"+key, func(t *testing.T) {
			copyFields := map[string]json.RawMessage{}
			for k, v := range fields {
				if k != key {
					copyFields[k] = v
				}
			}
			b, err := json.Marshal(copyFields)
			if err != nil {
				t.Fatal(err)
			}
			token := base64.RawURLEncoding.EncodeToString(b)
			prefilled := c
			if !errors.Is(decodeActivityCursor(token, &prefilled), ErrInvalid) {
				t.Fatal("omission inherited prefilled value")
			}
			if _, err := ReadActivity(context.Background(), never, workspace, feed, token, 2); !errors.Is(err, ErrInvalid) {
				t.Fatalf("omission: %v", err)
			}
		})
	}
	valid := base64.RawURLEncoding.EncodeToString(raw)
	for _, tc := range []struct{ name, token, w, f string }{
		{"workspace", valid, installation, feed}, {"feed", valid, workspace, item},
		{"encoding", "%%%", workspace, feed}, {"oversized", strings.Repeat("a", 2049), workspace, feed},
		{"duplicate", base64.RawURLEncoding.EncodeToString(append([]byte(`{"v":1,`), raw[1:]...)), workspace, feed},
		{"unknown", base64.RawURLEncoding.EncodeToString(append([]byte(`{"extra":1,`), raw[1:]...)), workspace, feed},
		{"null", base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(string(raw), `"v":1`, `"v":null`, 1))), workspace, feed},
		{"case", base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(string(raw), `"w":`, `"W":`, 1))), workspace, feed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ReadActivity(context.Background(), never, tc.w, tc.f, tc.token, 2); !errors.Is(err, ErrInvalid) {
				t.Fatalf("invalid cursor: %v", err)
			}
		})
	}
}

func insertActivityFixture(t *testing.T, db *sql.DB, base Acceptance, at time.Time, status string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a := base
	a.ItemID = testID(t)
	a.Submission.IdempotencyKey = "private-key-" + a.ItemID
	a.Submission.Title = "Original title\x00retained"
	a.Submission.Text = "Original\x00text retained."
	a.StageWork = func(context.Context, string) (string, error) { return testID(t), nil }
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, _, err = Accept(ctx, FromSQL(tx), a); err != nil {
		t.Fatal(err)
	}
	// Seed lifecycle state directly: this fixture qualifies reads, not processing.
	if _, err = tx.ExecContext(ctx, `UPDATE public.app_prosody_submissions SET accepted_at=$2,status=$3,deleted_at=CASE WHEN $3='deleted' THEN clock_timestamp() ELSE NULL END WHERE id=$1`, a.ItemID, at, status); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return a.ItemID
}

func TestActivityStatusesPaginationAndScope(t *testing.T) {
	db, _ := openFixture(t)
	a := testAcceptance(t, db)
	foreign := testAcceptance(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	at := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	expected := map[string]string{}
	for _, status := range []string{"processing", "failed", "ready"} {
		expected[insertActivityFixture(t, db, a, at, status)] = status
	}
	insertActivityFixture(t, db, a, at, "deleted")
	insertActivityFixture(t, db, foreign, at, "ready")
	// A separate feed in the same workspace must not leak into this feed.
	other := a
	other.Submission.FeedID = testID(t)
	if _, err := db.ExecContext(ctx, `INSERT INTO public.app_prosody_feeds(workspace_id,id,kind,title) VALUES($1,$2,'team',$3)`, a.Scope.WorkspaceID, other.Submission.FeedID, []byte("Other")); err != nil {
		t.Fatal(err)
	}
	insertActivityFixture(t, db, other, at, "failed")
	var got []string
	cursor := ""
	for pageIndex := 0; pageIndex < 3; pageIndex++ {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		page, err := ReadActivity(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, a.Submission.FeedID, cursor, 2)
		rollbackErr := tx.Rollback()
		if err != nil {
			t.Fatal(err)
		}
		if rollbackErr != nil {
			t.Fatal(rollbackErr)
		}
		for _, summary := range page.Items {
			if summary.Status != expected[summary.ID] {
				t.Fatalf("unexpected item/status: %+v", summary)
			}
			if summary.Title != "Original title\x00retained" || summary.Source != "Original\x00text retained." || !summary.AcceptedAt.Equal(at) {
				t.Fatalf("source changed: %+v", summary)
			}
			got = append(got, summary.ID)
		}
		encoded, err := json.Marshal(page)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"idempotencyKey", "private-key-", "authority", "settings", "workId", "provenance", "segments", "renditionId"} {
			if strings.Contains(string(encoded), secret) {
				t.Fatalf("private projection %q", secret)
			}
		}
		cursor = page.NextCursor
		if cursor == "" {
			break
		}
	}
	want := make([]string, 0, len(expected))
	for id := range expected {
		want = append(want, id)
	}
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") || cursor != "" {
		t.Fatalf("ties/pagination: got %v want %v cursor %q", got, want, cursor)
	}
	// A non-deleted status with a tombstone is also excluded.
	if _, err := db.ExecContext(ctx, `UPDATE public.app_prosody_submissions SET deleted_at=clock_timestamp() WHERE id=$1`, want[0]); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	page, err := ReadActivity(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, a.Submission.FeedID, "", 50)
	tx.Rollback()
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("tombstone: %+v %v", page, err)
	}
	for _, entry := range page.Items {
		if entry.ID == want[0] {
			t.Fatal("tombstone returned")
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE public.app_prosody_feeds SET deleted_at=clock_timestamp() WHERE workspace_id=$1 AND id=$2`, a.Scope.WorkspaceID, a.Submission.FeedID); err != nil {
		t.Fatal(err)
	}
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := ReadActivity(ctx, FromSQLRead(tx), a.Scope.WorkspaceID, a.Submission.FeedID, "", 2); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("deleted feed: %v", err)
	}
	if _, err := ReadActivity(ctx, FromSQLRead(tx), foreign.Scope.WorkspaceID, a.Submission.FeedID, "", 2); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("cross workspace feed: %v", err)
	}
}
