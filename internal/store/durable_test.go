package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/narrate-it/prosody/internal/content"
)

func testID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	b[6] = b[6]&15 | 0x70
	b[8] = b[8]&63 | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func openFixture(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn := os.Getenv("PROSODY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("owned disposable PostgreSQL fixture required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err = db.ExecContext(ctx, Migration); err != nil {
		t.Fatal(err)
	}
	return db, dsn
}
func testAcceptance(t *testing.T, db *sql.DB) Acceptance {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	w, f := testID(t), testID(t)
	if _, err := db.ExecContext(ctx, `INSERT INTO public.app_prosody_feeds(workspace_id,id,kind,title) VALUES($1,$2,'personal',$3)`, w, f, []byte("inbox")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, q := range []string{
			`DELETE FROM public.app_prosody_work WHERE item_id IN(SELECT id FROM public.app_prosody_submissions WHERE workspace_id=$1)`,
			`DELETE FROM public.app_prosody_progress WHERE workspace_id=$1`,
			`DELETE FROM public.app_prosody_renditions WHERE workspace_id=$1`,
			`DELETE FROM public.app_prosody_submissions WHERE workspace_id=$1`,
			`DELETE FROM public.app_prosody_feeds WHERE workspace_id=$1`,
		} {
			if _, err := db.ExecContext(ctx, q, w); err != nil {
				t.Errorf("clean owned fixture rows: %v", err)
			}
		}
	})
	work := testID(t)
	return Acceptance{Scope: Scope{testID(t), "prosody", "test", w, testID(t), "prosody.messages.submit-text"}, Submission: content.TextSubmission{WorkspaceID: w, FeedID: f, IdempotencyKey: " \x00\r\n界 ", Text: "original\x00source"}, ItemID: testID(t), StageWork: func(context.Context, string) (string, error) { return work, nil }, SettingsSnapshot: []byte(`{"model":"chosen"}`), AuthorityReference: []byte("fixture-authority-not-AMOS")}
}
func TestCommittedReplayNewConnection(t *testing.T) {
	db, dsn := openFixture(t)
	a := testAcceptance(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	receipt, fresh, err := Accept(ctx, FromSQL(tx), a)
	if err != nil || !fresh {
		t.Fatalf("fresh=%t err=%v", fresh, err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// Simulate losing the successful response, then reconnecting with the same key.
	other, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	retry, err := other.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer retry.Rollback()
	a.ItemID = testID(t)
	a.StageWork = func(context.Context, string) (string, error) {
		t.Error("replay enqueued another job")
		return "", ErrInvalid
	}
	replay, newRecord, err := Accept(ctx, FromSQL(retry), a)
	if err != nil || newRecord || replay != receipt {
		t.Fatalf("replay=%+v fresh=%t err=%v", replay, newRecord, err)
	}
	if err = retry.Commit(); err != nil {
		t.Fatal(err)
	}
}
func TestConcurrentKeyStagesOnlyOneWorkIntent(t *testing.T) {
	db, _ := openFixture(t)
	a := testAcceptance(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var staged atomic.Int32
	work := testID(t)
	a.StageWork = func(context.Context, string) (string, error) { staged.Add(1); return work, nil }
	b := a
	b.ItemID = testID(t)
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, input := range []Acceptance{a, b} {
		go func(input Acceptance) {
			<-start
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				results <- err
				return
			}
			defer tx.Rollback()
			_, _, err = Accept(ctx, FromSQL(tx), input)
			if err == nil {
				err = tx.Commit()
			}
			results <- err
		}(input)
	}
	close(start)
	var failures []error
	for range 2 {
		if err := <-results; err != nil {
			failures = append(failures, err)
		}
	}
	for _, err := range failures {
		t.Error(err)
	}
	if staged.Load() != 1 {
		t.Fatalf("staged %d work intents", staged.Load())
	}
}
func TestEnqueueFailureRollsBackSource(t *testing.T) {
	db, _ := openFixture(t)
	a := testAcceptance(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	a.StageWork = func(context.Context, string) (string, error) { return "", fmt.Errorf("effect admission refused") }
	if _, _, err = Accept(ctx, FromSQL(tx), a); err == nil {
		t.Fatal("accepted failed work intent")
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRowContext(ctx, `SELECT count(*) FROM public.app_prosody_submissions WHERE id=$1`, a.ItemID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("source survived rollback: %d %v", count, err)
	}
}

func TestDistinctSubmissionCannotReuseWorkID(t *testing.T) {
	db, _ := openFixture(t)
	a := testAcceptance(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	receipt, _, err := Accept(ctx, FromSQL(tx), a)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	a.ItemID = testID(t)
	a.Submission.IdempotencyKey = "distinct-submission"
	a.StageWork = func(context.Context, string) (string, error) { return receipt.WorkID, nil }
	second, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Rollback()
	if _, _, err = Accept(ctx, FromSQL(second), a); err == nil {
		t.Fatal("accepted duplicate work identity")
	}
	if err = second.Rollback(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRowContext(ctx, `SELECT count(*) FROM public.app_prosody_submissions WHERE id=$1`, a.ItemID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed binding survived rollback: %d %v", count, err)
	}
}
