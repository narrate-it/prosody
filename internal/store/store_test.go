package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/narrate-it/prosody/internal/content"
)

const (
	installation = "01900000-0000-7000-8000-000000000001"
	workspace    = "01900000-0000-7000-8000-000000000002"
	person       = "01900000-0000-7000-8000-000000000003"
	feed         = "01900000-0000-7000-8000-000000000004"
	item         = "01900000-0000-7000-8000-000000000005"
	work         = "01900000-0000-7000-8000-000000000006"
)

func TestCanonicalFeed(t *testing.T) {
	for _, s := range []string{"", strings.ToUpper("01900000-0000-7000-8000-00000000000a"), "01900000-0000-4000-8000-000000000004", "01900000-0000-7000-0000-000000000004"} {
		if s != feed && FeedID(s) {
			t.Fatalf("accepted %q", s)
		}
	}
	if !FeedID(feed) {
		t.Fatal("rejected UUIDv7")
	}
}

// This test only uses an explicitly provided disposable database. No network
// fixture or shared AMOS database is discovered from ambient configuration.
func TestPostgresAcceptance(t *testing.T) {
	dsn := os.Getenv("PROSODY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PROSODY_TEST_DATABASE_URL to an owned disposable database")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer closeFixture(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err = db.ExecContext(ctx, Migration); err != nil {
		t.Fatal(err)
	}
	// Test changes are always rolled back; migrations apply only to owned fixture.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rollbackFixture(t, tx)
	if _, err = tx.ExecContext(ctx, `INSERT INTO public.app_prosody_feeds(workspace_id,id,kind,title) VALUES($1,$2,'personal',$3)`, workspace, feed, []byte("inbox")); err != nil {
		t.Fatal(err)
	}
	a := Acceptance{Scope: Scope{installation, "prosody", "test", workspace, person, "prosody.messages.submit-text"}, Submission: content.TextSubmission{WorkspaceID: workspace, FeedID: feed, IdempotencyKey: " \x00\r\n界 ", Text: "source\x00unchanged"}, ItemID: item, StageWork: func(context.Context, string) (string, error) { return work, nil }, SettingsSnapshot: []byte(`{"model":"chosen"}`), AuthorityReference: []byte("held-authority")}
	r, fresh, err := Accept(ctx, FromSQL(tx), a)
	if err != nil || !fresh {
		t.Fatalf("accept fresh=%t: %v", fresh, err)
	}
	replay, newRecord, err := Accept(ctx, FromSQL(tx), a)
	if err != nil || newRecord || replay != r {
		t.Fatalf("replay=%+v fresh=%t err=%v", replay, newRecord, err)
	}
	var raw, key []byte
	if err = tx.QueryRowContext(ctx, `SELECT source_bytes,key_bytes FROM public.app_prosody_submissions WHERE id=$1`, item).Scan(&raw, &key); err != nil {
		t.Fatal(err)
	}
	var decoded content.TextSubmission
	if err = json.Unmarshal(raw, &decoded); err != nil || decoded != a.Submission || string(key) != a.Submission.IdempotencyKey {
		t.Fatalf("lossy roundtrip: %v", err)
	}
	otherFeed := "01900000-0000-7000-8000-000000000007"
	if _, err = tx.ExecContext(ctx, `INSERT INTO public.app_prosody_feeds(workspace_id,id,kind,title) VALUES($1,$2,'team',$3)`, workspace, otherFeed, []byte("other")); err != nil {
		t.Fatal(err)
	}
	a.Submission.FeedID = otherFeed
	if _, _, err = Accept(ctx, FromSQL(tx), a); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected cross-feed conflict: %v", err)
	}
	a.Submission.FeedID = feed
	a.Submission.Text = "changed"
	if _, _, err = Accept(ctx, FromSQL(tx), a); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict: %v", err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRowContext(ctx, `SELECT count(*) FROM public.app_prosody_submissions WHERE id=$1`, item).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rollback count=%d err=%v", count, err)
	}
}

// A transaction may already have committed, rolled back, or been canceled.
func rollbackFixture(t *testing.T, tx *sql.Tx) {
	t.Helper()
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		t.Errorf("rollback fixture: %v", err)
	}
}

func closeFixture(t *testing.T, resource interface{ Close() error }) {
	t.Helper()
	if err := resource.Close(); err != nil {
		t.Errorf("close fixture: %v", err)
	}
}
