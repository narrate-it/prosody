package media

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Public Put always passes real Close. This test exercises its identical private
// implementation core, substituting only a real-close-then-cancel operation.
func TestPutCoreCancellationAfterRealCloseDoesNotPublish(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	files, err := NewFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	closes := 0
	closeThenCancel := func(file *os.File) error {
		closes++
		if err := file.Close(); err != nil {
			return err
		}
		if _, err := file.Stat(); !errors.Is(err, os.ErrClosed) {
			t.Errorf("upload descriptor not closed: %v", err)
		}
		cancel()
		return nil
	}
	workspace := "01900000-0000-7000-8000-000000000001"
	asset := "01900000-0000-7000-8000-000000000002"
	object, err := files.put(ctx, workspace, asset, bytes.NewReader([]byte("original bytes")), 100, closeThenCancel)
	if !errors.Is(err, context.Canceled) || object != (Object{}) {
		t.Errorf("post-close cancellation returned object=%+v error=%v", object, err)
	}
	if closes != 1 {
		t.Errorf("successful upload close calls=%d", closes)
	}
	if _, err := os.Stat(filepath.Join(root, workspace, asset)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("post-close cancellation published target: %v", err)
	}
	uploads, err := filepath.Glob(filepath.Join(root, workspace, ".upload-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(uploads) != 0 {
		t.Error("owned fixture retained canceled upload temp")
	}
}
