package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"strings"
	"testing"
)

func rangeObject(b []byte) Object {
	h := sha256.Sum256(b)
	return Object{Size: int64(len(b)), SHA256: hex.EncodeToString(h[:])}
}

type countReader struct {
	r    io.Reader
	read int
}

func (r *countReader) Read(b []byte) (int, error) { n, e := r.r.Read(b); r.read += n; return n, e }

func TestVerifiedRange(t *testing.T) {
	t.Parallel()
	data := bytes.Repeat([]byte("01234567"), 200000)
	object := rangeObject(data)
	for _, bounds := range [][2]int64{{0, 0}, {32760, 32800}, {0, MaxRangeBytes - 1}, {int64(len(data)) - 1, int64(len(data)) - 1}} {
		got, err := ReadVerifiedRange(context.Background(), bytes.NewReader(data), object, bounds[0], bounds[1])
		if err != nil || !bytes.Equal(got.Bytes, data[bounds[0]:bounds[1]+1]) || got.First != bounds[0] || got.TotalBytes != int64(len(data)) || got.ContentDigest != sha256.Sum256(data) {
			t.Fatalf("invalid slice: %v", err)
		}
	}
}

func TestRangeBoundsRejectBeforeReading(t *testing.T) {
	t.Parallel()
	data := bytes.Repeat([]byte{1}, int(MaxRangeBytes+2))
	object := rangeObject(data)
	for _, bounds := range [][2]int64{{-1, 0}, {1, 0}, {0, int64(len(data))}, {0, MaxRangeBytes}, {0, math.MaxInt64}, {math.MaxInt64, math.MaxInt64}} {
		r := &countReader{r: bytes.NewReader(data)}
		got, err := ReadVerifiedRange(context.Background(), r, object, bounds[0], bounds[1])
		if !errors.Is(err, ErrInvalid) || len(got.Bytes) != 0 || r.read != 0 {
			t.Fatalf("invalid bounds read data: %v", err)
		}
	}
}

func TestRangeVerifiesWholeObjectBeforeReturning(t *testing.T) {
	t.Parallel()
	data := []byte("requested bytes and unrequested bytes")
	for _, bad := range [][]byte{append([]byte("X"), data[1:]...), append(append([]byte(nil), data[:len(data)-1]...), 'X'), data[:len(data)-1], append(append([]byte(nil), data...), 'X')} {
		got, err := ReadVerifiedRange(context.Background(), bytes.NewReader(bad), rangeObject(data), 0, 3)
		if !errors.Is(err, ErrObjectMismatch) || len(got.Bytes) != 0 {
			t.Fatalf("mismatch exposed data: %v", err)
		}
	}
}

type failingRangeReader struct{ done bool }

func (r *failingRangeReader) Read(p []byte) (int, error) {
	if !r.done {
		r.done = true
		return copy(p, "bytes"), io.ErrUnexpectedEOF
	}
	return 0, io.EOF
}

type emptyRangeReader struct{}

func (emptyRangeReader) Read([]byte) (int, error) { return 0, nil }

func TestRangeFailuresExposeNothing(t *testing.T) {
	t.Parallel()
	for _, r := range []io.Reader{&failingRangeReader{}, emptyRangeReader{}} {
		got, err := ReadVerifiedRange(context.Background(), r, rangeObject([]byte("bytes")), 0, 2)
		if err == nil || len(got.Bytes) != 0 {
			t.Fatal("read failure exposed bytes")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &countReader{r: bytes.NewReader([]byte("bytes"))}
	got, err := ReadVerifiedRange(ctx, r, rangeObject([]byte("bytes")), 0, 2)
	if !errors.Is(err, context.Canceled) || len(got.Bytes) != 0 || r.read != 0 {
		t.Fatal("canceled read exposed bytes")
	}
}

type terminalRangeReader struct {
	data    []byte
	failure error
	cancel  context.CancelFunc
}

func (r terminalRangeReader) Read(p []byte) (int, error) {
	if r.cancel != nil {
		r.cancel()
	}
	return copy(p, r.data), r.failure
}
func TestRangeTerminalReadsAndSafeErrors(t *testing.T) {
	data := []byte("bytes")
	object := rangeObject(data)
	got, err := ReadVerifiedRange(context.Background(), terminalRangeReader{data: data, failure: io.EOF}, object, 1, 3)
	if err != nil || string(got.Bytes) != "yte" {
		t.Fatal("data with EOF rejected")
	}
	private := errors.New("private source /private/path")
	got, err = ReadVerifiedRange(context.Background(), terminalRangeReader{data: data, failure: private}, object, 1, 3)
	if !errors.Is(err, private) || strings.Contains(err.Error(), "private") || len(got.Bytes) != 0 {
		t.Fatal("reader error leaked or identity lost")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got, err = ReadVerifiedRange(ctx, terminalRangeReader{data: data, failure: io.EOF, cancel: cancel}, object, 1, 3)
	if !errors.Is(err, context.Canceled) || len(got.Bytes) != 0 {
		t.Fatal("cancellation during read exposed data")
	}
	r := &countReader{r: bytes.NewReader(data)}
	object.SHA256 = strings.Repeat("a", 4096)
	got, err = ReadVerifiedRange(context.Background(), r, object, 0, 0)
	if !errors.Is(err, ErrInvalid) || r.read != 0 || len(got.Bytes) != 0 {
		t.Fatal("unbounded digest admitted")
	}
}
