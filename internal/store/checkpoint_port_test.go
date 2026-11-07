package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// This port double checks callback sequencing/arguments only. It is not a
// database, transactional rollback model or native committed-receipt fixture.
type checkpointPort struct {
	t                                           *testing.T
	binding, sourceBinding                      CheckpointBinding
	source, settings, artifact, encoded, digest []byte
	sourceErr, readErr, execErr                 error
	result                                      sql.Result
	queries, inserts                            int
	afterSource, afterRead, afterInsert         func()
}

type checkpointRow func(...any) error

func (r checkpointRow) Scan(dest ...any) error { return r(dest...) }

type checkpointRows struct {
	n   int64
	err error
}

func (r checkpointRows) RowsAffected() (int64, error) { return r.n, r.err }
func (r checkpointRows) LastInsertId() (int64, error) { return 0, errors.New("unused") }

func newCheckpointPort(t *testing.T) *checkpointPort {
	t.Helper()
	source, settings, artifact := []byte("persisted exact source"), []byte(`{"settingsRevision":1}`), []byte("Briefing.")
	b := checkpointBinding(t, source, settings, artifact)
	encoded, err := b.encode()
	if err != nil {
		t.Fatal(err)
	}
	return &checkpointPort{t: t, binding: b, sourceBinding: b, source: source, settings: settings, artifact: artifact, encoded: encoded, digest: append([]byte(nil), b.ArtifactDigest[:]...), result: checkpointRows{n: 1}}
}
func (p *checkpointPort) tx() Transaction {
	return Transaction{
		QueryRowContext: func(_ context.Context, q string, args ...any) Row {
			p.queries++
			switch q {
			case CheckpointSourceSQL:
				if !reflect.DeepEqual(args, []any{p.binding.WorkspaceID, p.binding.ItemID, p.binding.JobID}) {
					p.t.Fatalf("wrong source slots: %v", args)
				}
				return checkpointRow(func(dest ...any) error {
					if p.sourceErr != nil {
						return p.sourceErr
					}
					for i, v := range []string{p.sourceBinding.InstallationID, p.sourceBinding.ApplicationID, p.sourceBinding.EnvironmentID, p.sourceBinding.PersonID, p.sourceBinding.FeedID} {
						*dest[i].(*string) = v
					}
					*dest[5].(*[]byte) = p.source
					*dest[6].(*[]byte) = p.settings
					if p.afterSource != nil {
						p.afterSource()
					}
					return nil
				})
			case CheckpointReadSQL:
				if !reflect.DeepEqual(args, []any{p.binding.WorkspaceID, p.binding.JobID, p.binding.InvocationID, p.binding.Stage}) {
					p.t.Fatalf("wrong read slots: %v", args)
				}
				return checkpointRow(func(dest ...any) error {
					if p.readErr != nil {
						return p.readErr
					}
					*dest[0].(*[]byte) = p.encoded
					*dest[1].(*[]byte) = p.artifact
					*dest[2].(*[]byte) = p.digest
					if p.afterRead != nil {
						p.afterRead()
					}
					return nil
				})
			default:
				p.t.Fatalf("unadmitted query: %s", q)
				return nil
			}
		},
		ExecContext: func(_ context.Context, q string, args ...any) (sql.Result, error) {
			p.inserts++
			want := []any{p.binding.WorkspaceID, p.binding.JobID, p.binding.InvocationID, p.binding.Stage, p.binding.ItemID, p.binding.FeedID, p.encoded, p.artifact, p.digest}
			if q != CheckpointInsertSQL || !reflect.DeepEqual(args, want) {
				p.t.Fatal("unexpected statement or binding/artifact slots")
			}
			if p.afterInsert != nil {
				p.afterInsert()
			}
			return p.result, p.execErr
		},
	}
}

func TestCheckpointVersionedBinding(t *testing.T) {
	p := newCheckpointPort(t)
	var envelope struct {
		Version int               `json:"version"`
		Binding CheckpointBinding `json:"binding"`
	}
	if json.Unmarshal(p.encoded, &envelope) != nil || envelope.Version != CheckpointBindingEncodingVersion || envelope.Binding != p.binding {
		t.Fatal("binding envelope is not explicit and lossless")
	}
	again, err := EncodeCheckpointBinding(p.binding)
	if err != nil || !bytes.Equal(again, p.encoded) {
		t.Fatal("binding encoding is not stable")
	}
	invalid := p.binding
	invalid.Stage = "publication"
	if _, err := EncodeCheckpointBinding(invalid); !errors.Is(err, ErrInvalid) {
		t.Fatal("public codec accepted invalid binding")
	}
}

func TestCheckpointRejectsAmbiguousSettingsBeforeInsert(t *testing.T) {
	for _, raw := range []string{`{"settingsRevision":2,"settingsRevision":1}`, `{"SettingsRevision":1}`, `{"settingsRevision":1,"SettingsRevision":2}`, `{"settingsRevision":null}`, `{"settingsRevision":1.0}`, `{"settingsRevision":1,"other":"\ud800"}`, `[]`, `{`} {
		t.Run(raw, func(t *testing.T) {
			p := newCheckpointPort(t)
			p.settings = []byte(raw)
			p.binding.SettingsDigest = sha256.Sum256(p.settings)
			var encodeErr error
			p.encoded, encodeErr = p.binding.encode()
			if encodeErr != nil {
				t.Fatal(encodeErr)
			}
			fresh, err := WriteCheckpoint(context.Background(), p.tx(), p.binding, p.artifact)
			if !errors.Is(err, ErrConflict) || fresh || p.inserts != 0 {
				t.Fatal("malformed matching-hash settings advanced")
			}
		})
	}
}

func TestCheckpointInsertResultIsNotCommitProof(t *testing.T) {
	sentinel := errors.New("database result failure")
	for _, tc := range []struct {
		name                   string
		result                 sql.Result
		execErr, readErr, want error
		fresh                  bool
		queries                int
	}{
		{"fresh", checkpointRows{n: 1}, nil, nil, nil, true, 1},
		{"exact replay", checkpointRows{n: 0}, nil, nil, nil, false, 3},
		{"zero without row", checkpointRows{n: 0}, nil, sql.ErrNoRows, ErrUnavailable, false, 3},
		{"multiple", checkpointRows{n: 2}, nil, nil, ErrConflict, false, 1},
		{"negative", checkpointRows{n: -1}, nil, nil, ErrConflict, false, 1},
		{"nil result", nil, nil, nil, ErrInvalid, false, 1},
		{"count failure", checkpointRows{err: sentinel}, nil, nil, sentinel, false, 1},
		{"exec failure", nil, sentinel, nil, sentinel, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newCheckpointPort(t)
			p.result, p.execErr, p.readErr = tc.result, tc.execErr, tc.readErr
			fresh, err := WriteCheckpoint(context.Background(), p.tx(), p.binding, p.artifact)
			if !errors.Is(err, tc.want) || fresh != tc.fresh || p.inserts != 1 || p.queries != tc.queries {
				t.Fatalf("fresh=%v err=%v inserts=%d queries=%d", fresh, err, p.inserts, p.queries)
			}
		})
	}
}

func TestCheckpointSourceMismatchCannotReadOrWrite(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*checkpointPort)
		want   error
	}{
		{"tombstoned or foreign source", func(p *checkpointPort) { p.sourceErr = sql.ErrNoRows }, ErrUnavailable},
		{"installation", func(p *checkpointPort) { p.sourceBinding.InstallationID = testID(t) }, ErrConflict},
		{"application", func(p *checkpointPort) { p.sourceBinding.ApplicationID = "other" }, ErrConflict},
		{"environment", func(p *checkpointPort) { p.sourceBinding.EnvironmentID = "other" }, ErrConflict},
		{"person", func(p *checkpointPort) { p.sourceBinding.PersonID = testID(t) }, ErrConflict},
		{"feed", func(p *checkpointPort) { p.sourceBinding.FeedID = testID(t) }, ErrConflict},
		{"source digest", func(p *checkpointPort) { p.source = []byte("changed") }, ErrConflict},
		{"settings digest", func(p *checkpointPort) { p.settings = []byte(`{"settingsRevision":2}`) }, ErrConflict},
		{"settings revision", func(p *checkpointPort) {
			p.settings = []byte(`{"settingsRevision":2}`)
			p.binding.SettingsDigest = sha256.Sum256(p.settings)
		}, ErrConflict},
		{"missing source", func(p *checkpointPort) { p.source = nil }, ErrConflict},
		{"source error", func(p *checkpointPort) { p.sourceErr = errCheckpointFixtureIO }, errCheckpointFixtureIO},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newCheckpointPort(t)
			tc.mutate(p)
			fresh, err := WriteCheckpoint(context.Background(), p.tx(), p.binding, p.artifact)
			if !errors.Is(err, tc.want) || fresh || p.inserts != 0 {
				t.Fatal("mismatched source advanced")
			}
			p.queries = 0
			got, err := ReadCheckpoint(context.Background(), p.tx(), p.binding)
			if !errors.Is(err, tc.want) || len(got) != 0 || p.queries != 1 {
				t.Fatal("mismatched source exposed bytes")
			}
		})
	}
}

var errCheckpointFixtureIO = errors.New("fixture I/O failure")

func TestCheckpointReadRejectsStoredMismatch(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*checkpointPort)
	}{
		{"accepted result", func(p *checkpointPort) { p.binding.ResultDigest = sha256.Sum256([]byte("other result")) }},
		{"original attempt", func(p *checkpointPort) { p.binding.AttemptID = testID(t) }},
		{"binding", func(p *checkpointPort) { p.encoded = []byte("different binding") }},
		{"digest column", func(p *checkpointPort) { p.digest = make([]byte, 32) }},
		{"actual bytes", func(p *checkpointPort) { p.artifact = []byte("changed bytes") }},
		{"missing artifact", func(p *checkpointPort) { p.artifact = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newCheckpointPort(t)
			tc.mutate(p)
			got, err := ReadCheckpoint(context.Background(), p.tx(), p.binding)
			if !errors.Is(err, ErrConflict) || len(got) != 0 || p.inserts != 0 {
				t.Fatal("stored mismatch exposed bytes")
			}
		})
	}
}

func TestCheckpointCancellationDoesNotReportAdvance(t *testing.T) {
	for _, phase := range []string{"before", "after source", "after insert", "after read"} {
		t.Run(phase, func(t *testing.T) {
			p := newCheckpointPort(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch phase {
			case "before":
				cancel()
			case "after source":
				p.afterSource = cancel
			case "after insert":
				p.afterInsert = cancel
			case "after read":
				p.afterRead = cancel
			}
			if phase == "after read" {
				got, err := ReadCheckpoint(ctx, p.tx(), p.binding)
				if !errors.Is(err, context.Canceled) || len(got) != 0 {
					t.Fatal("canceled read exposed bytes")
				}
			} else {
				fresh, err := WriteCheckpoint(ctx, p.tx(), p.binding, p.artifact)
				if !errors.Is(err, context.Canceled) || fresh {
					t.Fatal("canceled write reported advance")
				}
				if phase != "after insert" && p.inserts != 0 {
					t.Fatal("canceled preflight dispatched insert")
				}
				if phase == "before" && p.queries != 0 {
					t.Fatal("canceled preflight dispatched read")
				}
			}
		})
	}
}

func TestCheckpointArtifactPreflightNeverCallsTransaction(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*checkpointPort)
	}{
		{"empty artifact", func(p *checkpointPort) { p.artifact = nil }},
		{"changed bytes", func(p *checkpointPort) { p.artifact = []byte("changed") }},
		{"oversized rewrite", func(p *checkpointPort) {
			p.artifact = bytes.Repeat([]byte{'a'}, 256001)
			p.binding.ArtifactDigest = sha256.Sum256(p.artifact)
		}},
		{"missing job", func(p *checkpointPort) { p.binding.JobID = "" }},
		{"invalid application", func(p *checkpointPort) { p.binding.ApplicationID = "bad app" }},
		{"unknown stage", func(p *checkpointPort) { p.binding.Stage = "publish" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newCheckpointPort(t)
			tc.mutate(p)
			fresh, err := WriteCheckpoint(context.Background(), p.tx(), p.binding, p.artifact)
			if !errors.Is(err, ErrInvalid) || fresh || p.queries != 0 || p.inserts != 0 {
				t.Fatal("invalid artifact/binding reached transaction")
			}
		})
	}
}
