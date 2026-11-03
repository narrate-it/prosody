package media_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/narrate-it/prosody/internal/media"
)

const cancelWorkspace = "01900000-0000-7000-8000-000000000001"
const cancelAsset = "01900000-0000-7000-8000-000000000002"

type cancelAtEOF struct {
	cancel context.CancelFunc
	calls  int
}

func (r *cancelAtEOF) Read(p []byte) (int, error) {
	r.calls++
	n := copy(p, []byte("original bytes"))
	r.cancel()
	return n, io.EOF
}

func TestPutCancellationDuringFinalReadDoesNotPublish(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	files, err := media.NewFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := &cancelAtEOF{cancel: cancel}
	object, err := files.Put(ctx, cancelWorkspace, cancelAsset, reader, 100)
	if !errors.Is(err, context.Canceled) || object != (media.Object{}) {
		t.Errorf("late cancellation returned object=%+v error=%v", object, err)
	}
	if reader.calls != 1 {
		t.Errorf("final-byte reader calls=%d", reader.calls)
	}
	if _, err = os.Stat(filepath.Join(root, cancelWorkspace, cancelAsset)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("canceled put published target: %v", err)
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			t.Errorf("canceled put left file %s", filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPutAlreadyCanceledHasNoFilesystemEffects(t *testing.T) {
	t.Parallel()
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "canceled", true: "deadline"}[deadline], func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			files, err := media.NewFiles(root)
			if err != nil {
				t.Fatal(err)
			}
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
			object, err := files.Put(ctx, cancelWorkspace, cancelAsset, bytes.NewReader([]byte("original bytes")), 100)
			if !errors.Is(err, want) || object != (media.Object{}) {
				t.Errorf("canceled put returned object=%+v error=%v", object, err)
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Errorf("pre-canceled put changed filesystem: %v", entries)
			}
		})
	}
}
