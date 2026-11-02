package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"testing"
)

func TestReconcileImmutableWrite(t *testing.T) {
	f, err := NewFiles(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w, a := "01900000-0000-7000-8000-000000000001", "01900000-0000-7000-8000-000000000002"
	original := []byte("original audio bytes")
	digest := sha256.Sum256(original)
	expected := Object{Size: int64(len(original)), SHA256: hex.EncodeToString(digest[:])}
	ctx := context.Background()
	if found, err := f.Reconcile(ctx, w, a, expected); err != nil || found {
		t.Fatalf("missing: %t %v", found, err)
	}
	if _, err = f.Put(ctx, w, a, bytes.NewReader(original), 100); err != nil {
		t.Fatal(err)
	}
	// Ignore the write receipt, as if it was lost after the file became durable.
	if found, err := f.Reconcile(ctx, w, a, expected); err != nil || !found {
		t.Fatalf("committed: %t %v", found, err)
	}
	changed := expected
	other := sha256.Sum256([]byte("different"))
	changed.SHA256 = hex.EncodeToString(other[:])
	if found, err := f.Reconcile(ctx, w, a, changed); found || !errors.Is(err, ErrObjectMismatch) {
		t.Fatalf("digest conflict: %t %v", found, err)
	}
	changed = expected
	changed.Size++
	if found, err := f.Reconcile(ctx, w, a, changed); found || !errors.Is(err, ErrObjectMismatch) {
		t.Fatalf("size conflict: %t %v", found, err)
	}
	changed.SHA256 = "not-a-digest"
	if _, err := f.Reconcile(ctx, w, a, changed); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad metadata: %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := f.Reconcile(canceled, w, a, expected); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	file, err := f.Open(w, a)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("close fixture file: %v", err)
		}
	}()
	saved, err := io.ReadAll(file)
	if err != nil || !bytes.Equal(saved, original) {
		t.Fatal("reconciliation changed original bytes")
	}
}
