package store_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/narrate-it/prosody/internal/store"
)

const (
	prefWorkspace    = "01900000-0000-7000-8000-000000000001"
	prefFeed         = "01900000-0000-7000-8000-000000000002"
	prefPerson       = "01900000-0000-7000-8000-000000000003"
	prefOther        = "01900000-0000-7000-8000-000000000004"
	prefReadFeedSQL  = `SELECT id FROM public.app_prosody_feeds WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR SHARE`
	prefWriteFeedSQL = `SELECT id FROM public.app_prosody_feeds WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE`
	prefReadSQL      = `SELECT notifications FROM public.app_prosody_preferences WHERE workspace_id=$1 AND feed_id=$2 AND person_id=$3 FOR SHARE`
	prefWriteSQL     = `INSERT INTO public.app_prosody_preferences(workspace_id,feed_id,person_id,notifications) VALUES($1,$2,$3,$4) ON CONFLICT(workspace_id,feed_id,person_id) DO UPDATE SET notifications=EXCLUDED.notifications`
)

type preferenceRow func(...any) error

func (r preferenceRow) Scan(args ...any) error { return r(args...) }

type preferenceResult struct {
	affected int64
	err      error
}

func (r preferenceResult) RowsAffected() (int64, error) { return r.affected, r.err }
func (r preferenceResult) LastInsertId() (int64, error) { return 0, errors.New("not used") }

// This ephemeral port double checks exact SQL/argument contracts, not database
// locks, transaction rollback, current authority or membership qualification.
type preferenceFixture struct {
	t                                                *testing.T
	values                                           map[store.NotificationPreferenceKey]bool
	queryCount, writeCount                           int
	lockErr, readErr, writeErr                       error
	lockedFeed                                       string
	result                                           sql.Result
	writeArgs                                        []any
	duringFeedScan, duringPreferenceScan, duringExec func()
}

func newPreferenceFixture(t *testing.T) *preferenceFixture {
	return &preferenceFixture{t: t, values: map[store.NotificationPreferenceKey]bool{}, lockedFeed: prefFeed, result: preferenceResult{affected: 1}}
}
func preferenceKey() store.NotificationPreferenceKey {
	return store.NotificationPreferenceKey{WorkspaceID: prefWorkspace, FeedID: prefFeed, PersonID: prefPerson}
}
func (f *preferenceFixture) query(_ context.Context, q string, args ...any) store.Row {
	f.queryCount++
	if q == prefReadFeedSQL || q == prefWriteFeedSQL {
		if len(args) != 2 {
			f.t.Fatal("feed lock arity")
		}
		return preferenceRow(func(dest ...any) error {
			if f.duringFeedScan != nil {
				f.duringFeedScan()
			}
			if f.lockErr != nil {
				return f.lockErr
			}
			if args[0] != prefWorkspace || args[1] != prefFeed {
				return sql.ErrNoRows
			}
			*dest[0].(*string) = f.lockedFeed
			return nil
		})
	}
	if q != prefReadSQL || len(args) != 3 {
		f.t.Fatalf("unexpected preference query %q args=%v", q, args)
	}
	return preferenceRow(func(dest ...any) error {
		if f.duringPreferenceScan != nil {
			f.duringPreferenceScan()
		}
		if f.readErr != nil {
			return f.readErr
		}
		k := store.NotificationPreferenceKey{WorkspaceID: args[0].(string), FeedID: args[1].(string), PersonID: args[2].(string)}
		enabled, exists := f.values[k]
		if !exists {
			return sql.ErrNoRows
		}
		*dest[0].(*bool) = enabled
		return nil
	})
}
func (f *preferenceFixture) exec(_ context.Context, q string, args ...any) (sql.Result, error) {
	f.writeCount++
	f.writeArgs = append([]any(nil), args...)
	if q != prefWriteSQL || len(args) != 4 {
		f.t.Fatalf("unexpected preference write %q args=%v", q, args)
	}
	if f.duringExec != nil {
		f.duringExec()
	}
	if f.writeErr != nil {
		return nil, f.writeErr
	}
	k := store.NotificationPreferenceKey{WorkspaceID: args[0].(string), FeedID: args[1].(string), PersonID: args[2].(string)}
	f.values[k] = args[3].(bool)
	return f.result, nil
}
func (f *preferenceFixture) readTx() store.ReadTransaction {
	return store.ReadTransaction{QueryRowContext: func(ctx context.Context, q string, args ...any) store.Row {
		if q != prefReadFeedSQL && q != prefReadSQL {
			f.t.Fatalf("read used wrong lock/query: %q", q)
		}
		return f.query(ctx, q, args...)
	}, QueryContext: func(context.Context, string, ...any) (store.ReadRows, error) {
		f.t.Fatal("unexpected multirow query")
		return nil, store.ErrInvalid
	}}
}
func (f *preferenceFixture) writeTx() store.Transaction {
	return store.Transaction{QueryRowContext: func(ctx context.Context, q string, args ...any) store.Row {
		if q != prefWriteFeedSQL {
			f.t.Fatalf("write used wrong feed lock: %q", q)
		}
		return f.query(ctx, q, args...)
	}, ExecContext: f.exec}
}

func TestNotificationPreferenceDefaultOnOffAndPersonIsolation(t *testing.T) {
	t.Parallel()
	f := newPreferenceFixture(t)
	k := preferenceKey()
	ctx := context.Background()
	if enabled, err := store.ReadNotificationPreference(ctx, f.readTx(), k); err != nil || enabled || len(f.values) != 0 {
		t.Fatalf("default must be false without insertion: %t %v", enabled, err)
	}
	for _, enabled := range []bool{true, false} {
		if err := store.SetNotificationPreference(ctx, f.writeTx(), k, enabled); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(f.writeArgs, []any{k.WorkspaceID, k.FeedID, k.PersonID, enabled}) {
			t.Fatal("preference write scope/boolean changed")
		}
		got, err := store.ReadNotificationPreference(ctx, f.readTx(), k)
		if err != nil || got != enabled {
			t.Fatalf("choice=%t got=%t err=%v", enabled, got, err)
		}
		other := k
		other.PersonID = prefOther
		if got, err := store.ReadNotificationPreference(ctx, f.readTx(), other); err != nil || got {
			t.Fatal("one person's preference leaked to another")
		}
	}
	if f.writeCount != 2 || len(f.values) != 1 {
		t.Fatal("explicit off was not persisted in the exact key")
	}
}

func TestNotificationPreferenceRejectsMalformedInputsBeforeSQL(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		change func(*store.NotificationPreferenceKey)
	}{
		{"workspace", func(k *store.NotificationPreferenceKey) { k.WorkspaceID = "invalid" }},
		{"feed", func(k *store.NotificationPreferenceKey) { k.FeedID = "invalid" }},
		{"non v7 feed", func(k *store.NotificationPreferenceKey) { k.FeedID = "01900000-0000-4000-8000-000000000002" }},
		{"person", func(k *store.NotificationPreferenceKey) { k.PersonID = "invalid" }},
		{"oversized person", func(k *store.NotificationPreferenceKey) { k.PersonID = string(make([]byte, 4096)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newPreferenceFixture(t)
			k := preferenceKey()
			tc.change(&k)
			if enabled, err := store.ReadNotificationPreference(context.Background(), f.readTx(), k); !errors.Is(err, store.ErrInvalid) || enabled {
				t.Fatal("invalid read accepted")
			}
			if err := store.SetNotificationPreference(context.Background(), f.writeTx(), k, true); !errors.Is(err, store.ErrInvalid) {
				t.Fatal("invalid write accepted")
			}
			if f.queryCount != 0 || f.writeCount != 0 {
				t.Fatal("invalid identity reached SQL")
			}
		})
	}
	k := preferenceKey()
	if _, err := store.ReadNotificationPreference(context.Background(), store.ReadTransaction{}, k); !errors.Is(err, store.ErrInvalid) {
		t.Fatal("incomplete read port accepted")
	}
	if err := store.SetNotificationPreference(context.Background(), store.Transaction{}, k, true); !errors.Is(err, store.ErrInvalid) {
		t.Fatal("incomplete write port accepted")
	}
}

func TestNotificationPreferenceUnavailableFeedNeverDefaultsOrWrites(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		change func(*preferenceFixture, *store.NotificationPreferenceKey)
	}{
		{"foreign workspace", func(_ *preferenceFixture, k *store.NotificationPreferenceKey) { k.WorkspaceID = prefOther }},
		{"foreign feed", func(_ *preferenceFixture, k *store.NotificationPreferenceKey) { k.FeedID = prefOther }},
		{"missing or tombstoned", func(f *preferenceFixture, _ *store.NotificationPreferenceKey) { f.lockErr = sql.ErrNoRows }},
		{"mismatched feed row", func(f *preferenceFixture, _ *store.NotificationPreferenceKey) { f.lockedFeed = prefOther }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newPreferenceFixture(t)
			k := preferenceKey()
			tc.change(f, &k)
			if enabled, err := store.ReadNotificationPreference(context.Background(), f.readTx(), k); !errors.Is(err, store.ErrUnavailable) || enabled {
				t.Fatal("unavailable feed became default false success")
			}
			if err := store.SetNotificationPreference(context.Background(), f.writeTx(), k, true); !errors.Is(err, store.ErrUnavailable) {
				t.Fatal("unavailable feed accepted write")
			}
			if f.queryCount != 2 || f.writeCount != 0 {
				t.Fatal("preference access continued after failed feed lock")
			}
		})
	}
}

func TestNotificationPreferencePropagatesErrorsAndExactAffectedCount(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("fixture SQL failure")
	for _, tc := range []struct {
		name  string
		setup func(*preferenceFixture)
		want  error
	}{
		{"write error", func(f *preferenceFixture) { f.writeErr = sentinel }, sentinel},
		{"nil result", func(f *preferenceFixture) { f.result = nil }, store.ErrInvalid},
		{"zero rows", func(f *preferenceFixture) { f.result = preferenceResult{affected: 0} }, store.ErrConflict},
		{"multiple rows", func(f *preferenceFixture) { f.result = preferenceResult{affected: 2} }, store.ErrConflict},
		{"row count error", func(f *preferenceFixture) { f.result = preferenceResult{err: sentinel} }, sentinel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newPreferenceFixture(t)
			tc.setup(f)
			if err := store.SetNotificationPreference(context.Background(), f.writeTx(), preferenceKey(), true); !errors.Is(err, tc.want) {
				t.Fatalf("write error=%v", err)
			}
			if f.queryCount != 1 || f.writeCount != 1 {
				t.Fatal("write retried or feed lock missing")
			}
		})
	}
	for _, lock := range []bool{true, false} {
		f := newPreferenceFixture(t)
		if lock {
			f.lockErr = sentinel
		} else {
			f.readErr = sentinel
		}
		if enabled, err := store.ReadNotificationPreference(context.Background(), f.readTx(), preferenceKey()); !errors.Is(err, sentinel) || enabled {
			t.Fatal("SQL read failure defaulted to false success")
		}
	}
	f := newPreferenceFixture(t)
	f.lockErr = sentinel
	if err := store.SetNotificationPreference(context.Background(), f.writeTx(), preferenceKey(), true); !errors.Is(err, sentinel) || f.writeCount != 0 {
		t.Fatal("lock failure continued to write")
	}
}

func TestNotificationPreferenceCanceledContextDoesNotQuery(t *testing.T) {
	t.Parallel()
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "canceled", true: "deadline"}[deadline], func(t *testing.T) {
			t.Parallel()
			var ctx context.Context
			var cancel context.CancelFunc
			want := context.Canceled
			if deadline {
				ctx, cancel = context.WithDeadline(context.Background(), time.Unix(1, 0))
				want = context.DeadlineExceeded
			} else {
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			defer cancel()
			f := newPreferenceFixture(t)
			if enabled, err := store.ReadNotificationPreference(ctx, f.readTx(), preferenceKey()); !errors.Is(err, want) || enabled {
				t.Fatal("read cancellation lost")
			}
			if err := store.SetNotificationPreference(ctx, f.writeTx(), preferenceKey(), true); !errors.Is(err, want) {
				t.Fatal("write cancellation lost")
			}
			if f.queryCount != 0 || f.writeCount != 0 {
				t.Fatal("canceled preference call queried SQL")
			}
		})
	}
}

type preferenceCountCallback struct {
	calls  *int
	during func()
	err    error
}

func (r preferenceCountCallback) RowsAffected() (int64, error) {
	*r.calls++
	if r.during != nil {
		r.during()
	}
	return 1, r.err
}
func (r preferenceCountCallback) LastInsertId() (int64, error) { return 0, errors.New("not used") }

func TestNotificationPreferenceCancellationDuringCallbacks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, phase             string
		write, present          bool
		queries, writes, counts int
	}{
		{"read feed scan", "feed", false, false, 1, 0, 0},
		{"read absent preference scan", "preference", false, false, 2, 0, 0},
		{"read enabled preference scan", "preference", false, true, 2, 0, 0},
		{"write feed scan", "feed", true, false, 1, 0, 0},
		{"write exec", "exec", true, false, 1, 1, 0},
		{"write affected count", "count", true, false, 1, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newPreferenceFixture(t)
			k := preferenceKey()
			if tc.present {
				f.values[k] = true
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			counts := 0
			result := preferenceCountCallback{calls: &counts}
			switch tc.phase {
			case "feed":
				f.duringFeedScan = cancel
			case "preference":
				f.duringPreferenceScan = cancel
			case "exec":
				f.duringExec = cancel
			case "count":
				result.during = cancel
			}
			f.result = result
			if tc.write {
				if err := store.SetNotificationPreference(ctx, f.writeTx(), k, true); !errors.Is(err, context.Canceled) {
					t.Fatalf("write returned success or lost cancellation: %v", err)
				}
			} else {
				if enabled, err := store.ReadNotificationPreference(ctx, f.readTx(), k); !errors.Is(err, context.Canceled) || enabled {
					t.Fatalf("read/default returned success after cancellation: %t %v", enabled, err)
				}
			}
			if f.queryCount != tc.queries || f.writeCount != tc.writes || counts != tc.counts {
				t.Fatalf("continued callback sequence: queries=%d writes=%d counts=%d", f.queryCount, f.writeCount, counts)
			}
			if tc.writes == 1 && !f.values[k] {
				t.Fatal("fixture lost executed write; cancellation is not rollback proof")
			}
		})
	}
}

func TestNotificationPreferenceSQLErrorCauseSurvivesConcurrentCancellation(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("synthetic SQL cause")
	for _, phase := range []string{"feed", "read", "exec", "count"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			f := newPreferenceFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			counts := 0
			switch phase {
			case "feed":
				f.duringFeedScan = cancel
				f.lockErr = sentinel
			case "read":
				f.duringPreferenceScan = cancel
				f.readErr = sentinel
			case "exec":
				f.duringExec = cancel
				f.writeErr = sentinel
			case "count":
				f.result = preferenceCountCallback{calls: &counts, during: cancel, err: sentinel}
			}
			var err error
			if phase == "feed" || phase == "read" {
				_, err = store.ReadNotificationPreference(ctx, f.readTx(), preferenceKey())
			} else {
				err = store.SetNotificationPreference(ctx, f.writeTx(), preferenceKey(), true)
			}
			if !errors.Is(err, sentinel) {
				t.Fatalf("SQL cause masked by cancellation: %v", err)
			}
		})
	}
}
