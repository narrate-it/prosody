package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"

	mp3 "github.com/hajimehoshi/go-mp3"
)

// TestMeasureMP3PreflightCancellation covers early cancellation with malformed
// bytes, plus active valid/invalid and already-cancelled valid controls.
func TestMeasureMP3PreflightCancellation(t *testing.T) {
	fixture, err := os.ReadFile("../briefing/testdata/silence.mp3")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		bytes []byte
	}{
		{"invalid_header", []byte("not MP3")},
		{"truncated_ID3", []byte("ID3\x04\x00\x00\x00\x00\x7f\x7f")},
		{"valid_recording", fixture},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			duration, err := MeasureMP3(ctx, tc.bytes, len(tc.bytes), time.Minute)
			if duration != 0 || !errors.Is(err, context.Canceled) {
				t.Fatalf("duration=%v error=%v; want cancelled with no measurement", duration, err)
			}
		})
	}
	t.Run("expired_deadline", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Unix(1, 0))
		defer cancel()
		duration, err := MeasureMP3(ctx, []byte("not MP3"), 7, time.Minute)
		if duration != 0 || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("duration=%v error=%v", duration, err)
		}
	})
	t.Run("active_valid_control", func(t *testing.T) {
		duration, err := MeasureMP3(context.Background(), fixture, len(fixture), time.Minute)
		if err != nil || duration <= 0 {
			t.Fatalf("duration=%v error=%v", duration, err)
		}
	})
	t.Run("active_invalid_control", func(t *testing.T) {
		duration, err := MeasureMP3(context.Background(), []byte("not MP3"), 7, time.Minute)
		if duration != 0 || !errors.Is(err, ErrInvalid) {
			t.Fatalf("duration=%v error=%v", duration, err)
		}
	})
}

// syntheticMP3Frames supplies frame structure only, never validated audio.
func syntheticMP3Frames(count int) []byte {
	result := make([]byte, count*417)
	for i := 0; i < count; i++ {
		binary.BigEndian.PutUint32(result[i*417:], 0xfffb9000)
	}
	return result
}

// cancelOnErrCheck triggers a real cancellation at a deterministic validation
// checkpoint, without sleep-based scheduling or a production test hook.
type cancelOnErrCheck struct {
	context.Context
	cancel context.CancelFunc
	checks atomic.Int32
	at     int32
}

// Err closes the underlying context at the selected validation checkpoint.
func (c *cancelOnErrCheck) Err() error {
	if c.checks.Add(1) == c.at {
		c.cancel()
	}
	return c.Context.Err()
}

// TestMeasureMP3CancellationDuringPreflight exercises the public no-result path.
func TestMeasureMP3CancellationDuringPreflight(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	controlled := &cancelOnErrCheck{Context: ctx, cancel: cancel, at: 3}
	frames := syntheticMP3Frames(4)
	duration, err := MeasureMP3(controlled, frames, len(frames), time.Minute)
	if duration != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("duration=%v error=%v; want cancelled without partial result", duration, err)
	}
	if ctx.Err() != context.Canceled {
		t.Fatal("underlying context was not cancelled")
	}
}

// TestMP3CompressedReaderCancellation checks the compressed-reader adapter
// directly, not the complete public MeasureMP3 construction path.
func TestMP3CompressedReaderCancellation(t *testing.T) {
	fixture, err := os.ReadFile("../briefing/testdata/silence.mp3")
	if err != nil {
		t.Fatal(err)
	}
	// Structural preflight is separate from the decoder-input cancellation check.
	// This Reader-level control exercises the same non-seeking compressed adapter.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reader := contextReader{ctx, bytes.NewReader(fixture)}
	if _, seekable := any(reader).(io.Seeker); seekable {
		t.Fatal("decoder input must not trigger an eager seek-index pass")
	}
	if _, err := mp3.NewDecoder(reader); !errors.Is(err, context.Canceled) {
		t.Fatalf("decoder setup error=%v", err)
	}
}

// TestMP3FramePreflightCancellation isolates cancellation inside header scanning,
// before any decoder reads can observe it.
func TestMP3FramePreflightCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	controlled := &cancelOnErrCheck{Context: ctx, cancel: cancel, at: 2}
	rate, samples, err := mp3Frames(controlled, syntheticMP3Frames(4))
	if rate != 0 || samples != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("rate=%d samples=%d error=%v; want no partial frame measurement", rate, samples, err)
	}
	if ctx.Err() != context.Canceled {
		t.Fatal("underlying context was not cancelled")
	}
}
