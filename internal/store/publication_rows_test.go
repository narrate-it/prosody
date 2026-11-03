package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

type publicationRow func(...any) error

func (r publicationRow) Scan(args ...any) error { return r(args...) }

type publicationCount int64

func (r publicationCount) LastInsertId() (int64, error) { return 0, errors.New("unused") }
func (r publicationCount) RowsAffected() (int64, error) { return int64(r), nil }

type publicationResultError struct{ err error }

func (r publicationResultError) LastInsertId() (int64, error) { return 0, r.err }
func (r publicationResultError) RowsAffected() (int64, error) { return 0, r.err }

func TestPublicationRequiresExactlyOneAffectedRow(t *testing.T) {
	sentinel := errors.New("row count unavailable")
	for _, tc := range []struct {
		name       string
		results    [2]sql.Result
		want       error
		executions int
	}{
		{"zero insert", [2]sql.Result{publicationCount(0), publicationCount(1)}, ErrConflict, 1},
		{"multiple insert", [2]sql.Result{publicationCount(2), publicationCount(1)}, ErrConflict, 1},
		{"zero update", [2]sql.Result{publicationCount(1), publicationCount(0)}, ErrConflict, 2},
		{"multiple update", [2]sql.Result{publicationCount(1), publicationCount(2)}, ErrConflict, 2},
		{"nil insert", [2]sql.Result{nil, publicationCount(1)}, ErrInvalid, 1},
		{"nil update", [2]sql.Result{publicationCount(1), nil}, ErrInvalid, 2},
		{"insert count error", [2]sql.Result{publicationResultError{sentinel}, publicationCount(1)}, sentinel, 1},
		{"update count error", [2]sql.Result{publicationCount(1), publicationResultError{sentinel}}, sentinel, 2},
		{"one each", [2]sql.Result{publicationCount(1), publicationCount(1)}, nil, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			executed, staged := 0, 0
			tx := Transaction{
				QueryRowContext: func(_ context.Context, q string, _ ...any) Row {
					if strings.HasPrefix(q, "SELECT s.status") {
						return publicationRow(func(args ...any) error {
							*args[0].(*string) = "processing"
							*args[1].(*[]byte) = []byte(`{"settingsRevision":1}`)
							return nil
						})
					}
					return publicationRow(func(args ...any) error {
						*args[0].(*string) = "audio/mpeg"
						*args[1].(*int64) = 1000
						*args[2].(*bool) = true
						return nil
					})
				},
				ExecContext: func(context.Context, string, ...any) (sql.Result, error) {
					result := tc.results[executed]
					executed++
					return result, nil
				},
			}
			err := Publish(context.Background(), tx, Publication{WorkspaceID: testID(t), ItemID: testID(t), WorkID: testID(t), RenditionID: testID(t), SettingsRevision: 1, Segments: []PublishedSegment{{ID: testID(t), Kind: "narration", AssetID: testID(t), DurationMS: 1000}}, Script: []byte("Briefing"), Provenance: []byte("fixture"), StageReady: func(context.Context, string, string) error { staged++; return nil }})
			if executed != tc.executions {
				t.Fatal("unexpected write after row-count failure")
			}
			if tc.want == nil {
				if err != nil || staged != 1 {
					t.Fatalf("exact count failed: %v", err)
				}
			} else if !errors.Is(err, tc.want) || staged != 0 {
				t.Fatalf("count mismatch staged readiness: %v", err)
			}
		})
	}
}
