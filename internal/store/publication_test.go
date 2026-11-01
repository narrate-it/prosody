package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPublicationAtomicityAndMediaAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, kind                                          string
		duration                                            int64
		foreign, wrongDuration, noScript, failStage, repeat bool
		want                                                error
	}{
		{name: "narration at limit", kind: "narration", duration: 600000},
		{name: "narration over limit", kind: "narration", duration: 600001, want: ErrInvalid},
		{name: "aggregate narration over limit", kind: "narration", duration: 300001, repeat: true, want: ErrInvalid},
		{name: "supplied two hour audio", kind: "audio", duration: 7200000},
		{name: "foreign asset", kind: "narration", duration: 1000, foreign: true, want: ErrUnavailable},
		{name: "duration mismatch", kind: "narration", duration: 1000, wrongDuration: true, want: ErrInvalid},
		{name: "missing script", kind: "narration", duration: 1000, noScript: true, want: ErrInvalid},
		{name: "notification staging rollback", kind: "narration", duration: 1000, failStage: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, _ := openFixture(t)
			a := testAcceptance(t, db)
			a.SettingsSnapshot = []byte(`{"settingsRevision":1}`)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			initial, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer initial.Rollback()
			receipt, _, err := Accept(ctx, FromSQL(initial), a)
			if err != nil {
				t.Fatal(err)
			}
			if err = initial.Commit(); err != nil {
				t.Fatal(err)
			}
			assetWorkspace := a.Scope.WorkspaceID
			if tc.foreign {
				assetWorkspace = testID(t)
			}
			assetID := testID(t)
			// Metadata fixture only: these tests do not qualify media decoding.
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if _, err = tx.ExecContext(ctx, `INSERT INTO public.app_prosody_assets(workspace_id,id,mime,byte_size,sha256,duration_ms,generated) VALUES($1,$2,'audio/mpeg',1,$3,$4,$5)`, assetWorkspace, assetID, make([]byte, 32), tc.duration, tc.kind == "narration"); err != nil {
				t.Fatal(err)
			}
			duration := tc.duration
			if tc.wrongDuration {
				duration++
			}
			p := Publication{WorkspaceID: a.Scope.WorkspaceID, ItemID: receipt.ItemID, WorkID: receipt.WorkID, RenditionID: testID(t), SettingsRevision: 1, Script: []byte("spoken script"), Provenance: []byte("fixture"), Segments: []PublishedSegment{{ID: testID(t), Kind: tc.kind, AssetID: assetID, DurationMS: duration}}}
			if tc.repeat {
				second := p.Segments[0]
				second.ID = testID(t)
				p.Segments = append(p.Segments, second)
			}
			if tc.noScript {
				p.Script = nil
			}
			stageFailure := errors.New("fixture enqueue refusal")
			staged := 0
			p.StageReady = func(ctx context.Context, item, rendition string) error {
				staged++
				if item != p.ItemID || rendition != p.RenditionID {
					t.Fatal("wrong notification identity")
				}
				var status string
				if err := tx.QueryRowContext(ctx, `SELECT status FROM public.app_prosody_submissions WHERE id=$1`, item).Scan(&status); err != nil || status != "ready" {
					t.Fatalf("not ready inside transaction: %s %v", status, err)
				}
				if tc.failStage {
					return stageFailure
				}
				return nil
			}
			err = Publish(ctx, FromSQL(tx), p)
			want := tc.want
			if tc.failStage {
				want = stageFailure
			}
			if !errors.Is(err, want) {
				t.Fatalf("publish got %v want %v", err, want)
			}
			if want != nil {
				if err = tx.Rollback(); err != nil {
					t.Fatal(err)
				}
			} else {
				if err = tx.Commit(); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if _, err := db.Exec(`DELETE FROM public.app_prosody_assets WHERE workspace_id=$1 AND id=$2`, assetWorkspace, assetID); err != nil {
						t.Error(err)
					}
				})
			}
			var status string
			var count int
			if err = db.QueryRowContext(ctx, `SELECT status FROM public.app_prosody_submissions WHERE id=$1`, receipt.ItemID).Scan(&status); err != nil {
				t.Fatal(err)
			}
			if err = db.QueryRowContext(ctx, `SELECT count(*) FROM public.app_prosody_renditions WHERE item_id=$1`, receipt.ItemID).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if want != nil {
				if status != "processing" || count != 0 {
					t.Fatalf("partial publication survived rollback: %s %d", status, count)
				}
				if !tc.failStage && staged != 0 {
					t.Fatal("invalid media staged notification")
				}
			} else if status != "ready" || count != 1 || staged != 1 {
				t.Fatalf("incomplete publication: %s %d %d", status, count, staged)
			}
		})
	}
}
