package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
)

const MaxRangeBytes int64 = 1 << 20

// RangeBytes is buffered evidence for a host-owned media response, not an access
// capability. ContentDigest covers the full object, not just Bytes. Native code
// must separately bind asset/workspace/feed/item/rendition, MIME and live grant.
type RangeBytes struct {
	First, TotalBytes int64
	ContentDigest     [sha256.Size]byte
	Bytes             []byte
}

// ReadVerifiedRange reads a closed inclusive range only after verifying the full
// object's expected size/digest. The selected bytes are captured during that
// same hash pass, avoiding a hash-then-reopen/read race. Memory is bounded by
// 1 MiB plus a small read buffer; I/O is bounded by expected size plus one byte.
// No content escapes on mismatch, cancellation or read failure. The caller owns
// the reader's deadline/close: context cannot interrupt an arbitrary blocking
// Reader. This helper is not a route, publication check or native authorization.
func ReadVerifiedRange(ctx context.Context, r io.Reader, expected Object, first, last int64) (RangeBytes, error) {
	if err := ctx.Err(); err != nil {
		return RangeBytes{}, &rangeReadError{cause: err}
	}
	if len(expected.SHA256) != sha256.Size*2 {
		return RangeBytes{}, ErrInvalid
	}
	digest, err := hex.DecodeString(expected.SHA256)
	if r == nil || err != nil || len(digest) != sha256.Size || expected.Size <= 0 || expected.Size > 1<<30 || first < 0 || last < first || last >= expected.Size || last-first >= MaxRangeBytes {
		return RangeBytes{}, ErrInvalid
	}
	hash := sha256.New()
	selected := make([]byte, 0, int(last-first+1))
	buffer := make([]byte, 32<<10)
	var position int64
	emptyReads := 0
	for {
		if err := ctx.Err(); err != nil {
			return RangeBytes{}, &rangeReadError{cause: err}
		}
		remaining := expected.Size + 1 - position
		chunk := buffer
		if remaining < int64(len(chunk)) {
			chunk = chunk[:int(remaining)]
		}
		n, readErr := r.Read(chunk)
		if n < 0 || n > len(chunk) {
			return RangeBytes{}, ErrInvalid
		}
		if n > 0 {
			emptyReads = 0
			// hash.Hash.Write cannot fail; no untrusted writer is involved.
			_, _ = hash.Write(chunk[:n])
			start, end := first-position, last+1-position
			if start < 0 {
				start = 0
			}
			if end > int64(n) {
				end = int64(n)
			}
			if start < end {
				selected = append(selected, chunk[int(start):int(end)]...)
			}
			position += int64(n)
			if position > expected.Size {
				return RangeBytes{}, ErrObjectMismatch
			}
		} else {
			emptyReads++
		}
		if readErr != nil {
			if readErr != io.EOF {
				return RangeBytes{}, &rangeReadError{cause: readErr}
			}
			break
		}
		if emptyReads >= 100 {
			return RangeBytes{}, io.ErrNoProgress
		}
	}
	if err := ctx.Err(); err != nil {
		return RangeBytes{}, &rangeReadError{cause: err}
	}
	if position != expected.Size || !bytes.Equal(hash.Sum(nil), digest) {
		return RangeBytes{}, ErrObjectMismatch
	}
	result := RangeBytes{First: first, TotalBytes: expected.Size, Bytes: selected}
	copy(result.ContentDigest[:], digest)
	return result, nil
}

// Rendering is safe for a response boundary; unwrap is diagnostic-only and must
// not be logged because an object adapter's cause may contain private paths/data.
type rangeReadError struct{ cause error }

func (e *rangeReadError) Error() string { return "media range read failed" }
func (e *rangeReadError) Unwrap() error { return e.cause }
