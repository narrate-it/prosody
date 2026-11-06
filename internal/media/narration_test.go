package media_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/narrate-it/prosody/internal/media"
)

// silentFrames makes synthetic zero-content MPEG1 LayerIII decoder inputs.
// These tests require real successful PCM decoding, not just frame-header checks.
// They do not qualify external providers or every supported MP3 encoding.
func silentFrames(frames int, frequency44100 bool) []byte {
	size, header := 384, uint32(0xfffb9400) // 128kb/s, 48000Hz, 1152 samples/frame.
	if frequency44100 {
		size, header = 417, 0xfffb9000
	}
	frame := make([]byte, size)
	binary.BigEndian.PutUint32(frame, header)
	return bytes.Repeat(frame, frames)
}

// TestMeasureNarrationAggregate checks complete-item and exact rational timing
// instead of accepting each part against a fresh independent duration budget.
func TestMeasureNarrationAggregate(t *testing.T) {
	a, b := silentFrames(3, false), silentFrames(2, false)
	for _, tc := range []struct {
		name     string
		parts    [][]byte
		maxBytes int
		budget   time.Duration
		want     time.Duration
		wantErr  error
	}{
		{"exact shared budget", [][]byte{a, b}, len(a) + len(b), 120 * time.Millisecond, 120 * time.Millisecond, nil},
		{"one nanosecond over shared budget", [][]byte{a, b}, len(a) + len(b), 120*time.Millisecond - 1, 0, media.ErrDuration},
		{"each fits but sum exceeds", [][]byte{a, b}, len(a) + len(b), 100 * time.Millisecond, 0, media.ErrDuration},
		{"combined byte overflow", [][]byte{a, b}, len(a) + len(b) - 1, time.Second, 0, media.ErrInvalid},
		{"late malformed part", [][]byte{a, []byte("not audio")}, len(a) + 100, time.Second, 0, media.ErrInvalid},
		{"late truncated part", [][]byte{a, b[:len(b)-1]}, len(a) + len(b), time.Second, 0, media.ErrInvalid},
		{"empty late part", [][]byte{a, nil}, len(a), time.Second, 0, media.ErrInvalid},
		{"no parts", nil, 1024, time.Second, 0, media.ErrInvalid},
		{"missing byte budget", [][]byte{a}, 0, time.Second, 0, media.ErrInvalid},
		{"raised byte codec ceiling", [][]byte{a}, 64<<20 + 1, time.Second, 0, media.ErrInvalid},
		{"missing duration budget", [][]byte{a}, len(a), 0, 0, media.ErrInvalid},
		{"raised ten minute ceiling", [][]byte{a}, len(a), 10*time.Minute + 1, 0, media.ErrInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := media.MeasureNarration(context.Background(), tc.parts, tc.maxBytes, tc.budget)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) || got.Duration != 0 || len(got.Parts) != 0 {
					t.Fatal("invalid aggregate accepted, error changed or partial measurement returned")
				}
				return
			}
			if err != nil || got.Duration != tc.want || len(got.Parts) != 2 || got.Parts[0] != 72*time.Millisecond || got.Parts[1] != 48*time.Millisecond {
				t.Fatalf("complete measured timing lost: %+v, %v", got, err)
			}
		})
	}
}

// TestMeasureNarrationRationalBoundary rejects sub-nanosecond excess that would
// disappear if each decoded part were rounded before applying the shared cap.
func TestMeasureNarrationRationalBoundary(t *testing.T) {
	part := silentFrames(1, true)
	parts := [][]byte{part, part, part}
	floor := time.Duration(int64(3*1152) * int64(time.Second) / 44100)
	got, err := media.MeasureNarration(context.Background(), parts, 3*len(part), floor)
	if !errors.Is(err, media.ErrDuration) || got.Duration != 0 || len(got.Parts) != 0 {
		t.Fatal("rounded individual parts hid an aggregate overrun")
	}
	got, err = media.MeasureNarration(context.Background(), parts, 3*len(part), floor+1)
	if err != nil || got.Duration != floor || len(got.Parts) != 3 || got.Duration == got.Parts[0]+got.Parts[1]+got.Parts[2] {
		t.Fatalf("exact total not rounded once: %+v, %v", got, err)
	}
}

// TestMeasureNarrationMixedRates compares the shared clock across independently
// valid recordings with different sample rates; an individual file cannot vary.
func TestMeasureNarrationMixedRates(t *testing.T) {
	a, b := silentFrames(1, false), silentFrames(2, true)
	floor := 24*time.Millisecond + time.Duration(int64(2*1152)*int64(time.Second)/44100)
	got, err := media.MeasureNarration(context.Background(), [][]byte{a, b}, len(a)+len(b), floor+1)
	if err != nil || got.Duration != floor || len(got.Parts) != 2 || got.Parts[0] != 24*time.Millisecond {
		t.Fatalf("mixed-rate timing lost: %+v, %v", got, err)
	}
	got, err = media.MeasureNarration(context.Background(), [][]byte{a, b}, len(a)+len(b), floor)
	if !errors.Is(err, media.ErrDuration) || got.Duration != 0 || len(got.Parts) != 0 {
		t.Fatal("mixed-rate fractional excess accepted")
	}
}

// TestMeasureNarrationTenMinutes exercises real decoding at the RFC ceiling.
func TestMeasureNarrationTenMinutes(t *testing.T) {
	part := silentFrames(12500, false) // 300 seconds of decoded PCM.
	parts := [][]byte{part, part}
	got, err := media.MeasureNarration(context.Background(), parts, 2*len(part), 10*time.Minute)
	if err != nil || got.Duration != 10*time.Minute || len(got.Parts) != 2 {
		t.Fatalf("exact ten minutes refused: %+v, %v", got, err)
	}
	parts = append(parts, silentFrames(1, false))
	got, err = media.MeasureNarration(context.Background(), parts, 2*len(part)+384, 10*time.Minute)
	if !errors.Is(err, media.ErrDuration) || got.Duration != 0 || len(got.Parts) != 0 {
		t.Fatal("over-ten-minute item accepted or a partial result escaped")
	}
}

// TestMeasureNarrationFiniteParts checks the existing 100-segment domain bound.
func TestMeasureNarrationFiniteParts(t *testing.T) {
	part := silentFrames(1, false)
	parts := make([][]byte, 100)
	for i := range parts {
		parts[i] = part
	}
	got, err := media.MeasureNarration(context.Background(), parts, 101*len(part), 3*time.Second)
	if err != nil || len(got.Parts) != 100 || got.Duration != 2400*time.Millisecond {
		t.Fatal("valid bounded part list refused")
	}
	got, err = media.MeasureNarration(context.Background(), append(parts, part), 101*len(part), 3*time.Second)
	if !errors.Is(err, media.ErrInvalid) || len(got.Parts) != 0 || got.Duration != 0 {
		t.Fatal("unbounded part list accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err = media.MeasureNarration(ctx, parts, 100*len(part), 3*time.Second)
	if !errors.Is(err, context.Canceled) || len(got.Parts) != 0 || got.Duration != 0 {
		t.Fatal("cancelled measurement returned a result")
	}
}
