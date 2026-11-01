package store

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/narrate-it/prosody/internal/playback"
)

func TestProgressIsolationRevisionAndDeletion(t *testing.T) {
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
	rendition := testID(t)
	manifest, err := json.Marshal([]PublishedSegment{{ID: testID(t), Kind: "narration", DurationMS: 1000}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO public.app_prosody_renditions(workspace_id,item_id,id,settings_revision,manifest_bytes,script_bytes,provenance_bytes,narration_ms) VALUES($1,$2,$3,1,$4,$5,$6,1000)`, a.Scope.WorkspaceID, receipt.ItemID, rendition, manifest, []byte("script"), []byte("fixture")); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE public.app_prosody_submissions SET status='ready' WHERE id=$1`, receipt.ItemID); err != nil {
		t.Fatal(err)
	}
	key := ProgressKey{a.Scope.WorkspaceID, a.Scope.PersonID, receipt.ItemID, rendition}
	update := playback.Update{ExpectedRevision: 0, Segment: 0, PositionMS: 1000, Played: []playback.Interval{{StartMS: 0, EndMS: 1000}}}
	first, err := UpdateProgress(ctx, FromSQL(tx), key, update)
	if err != nil || !first.Complete() {
		t.Fatalf("complete=%t err=%v", first.Complete(), err)
	}
	if _, err = UpdateProgress(ctx, FromSQL(tx), key, update); !errors.Is(err, playback.ErrConflict) {
		t.Fatalf("stale update: %v", err)
	}
	key.PersonID = testID(t)
	update.Played = nil
	other, err := UpdateProgress(ctx, FromSQL(tx), key, update)
	if err != nil || other.Complete() {
		t.Fatalf("shared another person's coverage: %v", err)
	}
	if err = Tombstone(ctx, FromSQL(tx), a.Scope.WorkspaceID, a.ItemID, func(context.Context, string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	update.ExpectedRevision = other.Revision
	if _, err = UpdateProgress(ctx, FromSQL(tx), key, update); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("progress after deletion: %v", err)
	}
}
