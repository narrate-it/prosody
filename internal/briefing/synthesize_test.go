package briefing

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"errors"
	"testing"
	"time"

	"github.com/narrate-it/prosody/internal/media"
	"github.com/narrate-it/prosody/internal/openrouter"
)

//go:embed testdata/silence.mp3
var silence []byte

type speechStub struct {
	calls int
	audio []byte
	err   error
}

func (p *speechStub) Speech(_ context.Context, key, model, voice, script string) (openrouter.SpeechResult, error) {
	p.calls++
	return openrouter.SpeechResult{Audio: p.audio, GenerationID: "fixture-request", RequestID: "fixture-transport", ContentType: "audio/mpeg"}, p.err
}
func TestSynthesisMeasuredOutput(t *testing.T) {
	p := &speechStub{audio: append([]byte(nil), silence...)}
	got, err := Synthesize(context.Background(), p, "test-key", "model", "voice", "A short briefing.", time.Second, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if p.calls != 1 || got.Duration < 100*time.Millisecond || got.Duration > 200*time.Millisecond || !bytes.Equal(got.Audio, silence) || got.SHA256 != sha256.Sum256(silence) {
		t.Fatal("invalid measured result")
	}
	p.audio[0] ^= 1
	if !bytes.Equal(got.Audio, silence) {
		t.Fatal("provider buffer aliases accepted audio")
	}
}
func TestSynthesisRejectsOutputWithoutRetry(t *testing.T) {
	for _, tc := range []struct {
		name     string
		audio    []byte
		budget   time.Duration
		maxBytes int
	}{
		{"malformed", []byte("not audio"), time.Second, 1 << 20},
		{"truncated", silence[:len(silence)-1], time.Second, 1 << 20},
		{"remaining budget", silence, 50 * time.Millisecond, 1 << 20},
		{"byte bound", silence, time.Second, len(silence) - 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &speechStub{audio: tc.audio}
			got, err := Synthesize(context.Background(), p, "test-key", "model", "voice", "Briefing.", tc.budget, tc.maxBytes)
			if err == nil || p.calls != 1 || len(got.Audio) != 0 || got.GenerationID != "fixture-request" {
				t.Fatal("invalid output accepted or receipt lost")
			}
			if tc.name == "remaining budget" && !errors.Is(err, media.ErrDuration) {
				t.Fatalf("duration error: %v", err)
			}
		})
	}
}
func TestSynthesisUncertainOutcomeAndPreflight(t *testing.T) {
	uncertain := errors.New("uncertain provider outcome")
	p := &speechStub{err: uncertain}
	got, err := Synthesize(context.Background(), p, "test-key", "model", "voice", "Briefing.", time.Second, 1<<20)
	if !errors.Is(err, uncertain) || p.calls != 1 || got.GenerationID != "fixture-request" || got.RequestID != "fixture-transport" || len(got.Audio) != 0 {
		t.Fatal("uncertain result retried or changed")
	}
	for _, budget := range []time.Duration{0, 11 * time.Minute} {
		_, err = Synthesize(context.Background(), p, "test-key", "model", "voice", "Briefing.", budget, 1<<20)
		if err == nil || p.calls != 1 {
			t.Fatal("invalid budget dispatched")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Synthesize(ctx, p, "test-key", "model", "voice", "Briefing.", time.Second, 1<<20)
	if !errors.Is(err, context.Canceled) || p.calls != 1 {
		t.Fatal("canceled call dispatched")
	}
}
