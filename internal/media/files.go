package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/narrate-it/prosody/internal/store"
)

// Files is a local private object adapter for owned development fixtures or a
// separately qualified persistent volume. Nanos production still requires an
// external durable object-store adapter. It never fetches caller-supplied URLs.
type Files struct{ root string }

func NewFiles(root string) (*Files, error) {
	if !filepath.IsAbs(root) {
		return nil, errors.New("absolute asset root required")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, fmt.Errorf("create asset root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve asset root: %w", err)
	}
	return &Files{resolved}, nil
}
func (f *Files) path(workspace, asset string) (string, error) {
	if f == nil || !store.UUID(workspace) || !store.UUID(asset) {
		return "", ErrInvalid
	}
	return filepath.Join(f.root, workspace, asset), nil
}

type Object struct {
	Size   int64
	SHA256 string
}

// Put writes original bytes once. Existing IDs never overwrite another object.
// The caller validates the content and commits metadata only after this succeeds.
func (f *Files) Put(ctx context.Context, workspace, asset string, r io.Reader, maxBytes int64) (Object, error) {
	path, err := f.path(workspace, asset)
	if err != nil || r == nil || maxBytes <= 0 || maxBytes > 1<<30 {
		return Object{}, ErrInvalid
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return Object{}, fmt.Errorf("create asset directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".upload-")
	if err != nil {
		return Object{}, fmt.Errorf("create upload: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	defer tmp.Close()
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(contextReader{ctx, r}, maxBytes+1))
	if err != nil {
		return Object{}, fmt.Errorf("write upload: %w", err)
	}
	if n == 0 || n > maxBytes {
		return Object{}, ErrInvalid
	}
	if err = tmp.Sync(); err != nil {
		return Object{}, fmt.Errorf("sync upload: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return Object{}, fmt.Errorf("close upload: %w", err)
	}
	// Link is atomic no-replace on the same filesystem; unlike Rename it cannot
	// silently overwrite an existing immutable asset.
	if err = os.Link(name, path); err != nil {
		return Object{}, fmt.Errorf("publish original asset: %w", err)
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return Object{}, fmt.Errorf("open asset directory: %w", err)
	}
	syncErr := dir.Sync()
	closeErr := dir.Close()
	if syncErr != nil {
		return Object{}, fmt.Errorf("sync asset directory: %w", syncErr)
	}
	if closeErr != nil {
		return Object{}, fmt.Errorf("close asset directory: %w", closeErr)
	}
	return Object{n, hex.EncodeToString(hash.Sum(nil))}, nil
}
func (f *Files) Open(workspace, asset string) (*os.File, error) {
	path, err := f.path(workspace, asset)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open original asset: %w", err)
	}
	return file, nil
}

// ErrObjectMismatch means an immutable ID exists with different content. It
// must not be overwritten or treated as a safe retry of the intended write.
var ErrObjectMismatch = errors.New("immutable asset content mismatch")

// Reconcile checks a prior uncertain Put against the expected size and SHA-256
// recorded by its owner. Missing returns false,nil; a match returns true,nil.
// Errors leave the outcome unresolved (or conflicting), never permission to retry.
// This local development adapter does not supply authority or a recovery journal.
func (f *Files) Reconcile(ctx context.Context, workspace, asset string, expected Object) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	digest, err := hex.DecodeString(expected.SHA256)
	if err != nil || len(digest) != sha256.Size || expected.Size <= 0 || expected.Size > 1<<30 {
		return false, ErrInvalid
	}
	file, err := f.Open(workspace, asset)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return false, fmt.Errorf("inspect original asset: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() != expected.Size {
		return false, ErrObjectMismatch
	}
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(contextReader{ctx, file}, expected.Size+1))
	if err != nil {
		return false, fmt.Errorf("verify original asset: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return false, err
	}
	if n != expected.Size || !bytes.Equal(hash.Sum(nil), digest) {
		return false, ErrObjectMismatch
	}
	return true, nil
}

func (f *Files) Delete(workspace, asset string) error {
	path, err := f.path(workspace, asset)
	if err != nil {
		return err
	}
	if err = os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove asset: %w", err)
	}
	return nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
