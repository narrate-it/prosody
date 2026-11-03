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
	data   []byte
}

func (r *cancelAtEOF) Read(p []byte) (int, error) {
	r.calls++
	data := r.data
	if data == nil {
		data = []byte("original bytes")
	}
	n := copy(p, data)
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

func TestCanceledReplacementPreservesPriorImmutableObject(t *testing.T) {
	t.Parallel()
	for _, late := range []bool{false, true} {
		t.Run(map[bool]string{false: "already canceled", true: "final read canceled"}[late], func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			files, err := media.NewFiles(root)
			if err != nil {
				t.Fatal(err)
			}
			originalBytes := []byte("prior immutable bytes")
			original, err := files.Put(context.Background(), cancelWorkspace, cancelAsset, bytes.NewReader(originalBytes), 100)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var replacement io.Reader = bytes.NewReader([]byte("replacement bytes"))
			if late {
				replacement = &cancelAtEOF{cancel: cancel, data: []byte("replacement bytes")}
			} else {
				cancel()
			}
			object, err := files.Put(ctx, cancelWorkspace, cancelAsset, replacement, 100)
			if !errors.Is(err, context.Canceled) || object != (media.Object{}) {
				t.Errorf("canceled replacement returned object=%+v error=%v", object, err)
			}
			file, err := files.Open(cancelWorkspace, cancelAsset)
			if err != nil {
				t.Fatal(err)
			}
			saved, readErr := io.ReadAll(file)
			closeErr := file.Close()
			if readErr != nil || closeErr != nil {
				t.Fatalf("read preserved object: %v %v", readErr, closeErr)
			}
			if !bytes.Equal(saved, originalBytes) {
				t.Error("canceled replacement changed prior immutable bytes")
			}
			matches, err := files.Reconcile(context.Background(), cancelWorkspace, cancelAsset, original)
			if err != nil || !matches {
				t.Errorf("prior immutable size/hash changed: %v", err)
			}
			uploads, err := filepath.Glob(filepath.Join(root, cancelWorkspace, ".upload-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(uploads) != 0 {
				t.Error("owned fixture retained replacement upload temp")
			}
		})
	}
}
