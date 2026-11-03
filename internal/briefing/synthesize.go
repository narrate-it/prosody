package briefing

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/narrate-it/prosody/internal/content"
	"github.com/narrate-it/prosody/internal/media"
	"github.com/narrate-it/prosody/internal/openrouter"
)

type SpeechProvider interface {
	Speech(context.Context, string, string, string, string) (openrouter.SpeechResult, error)
}

type Synthesized struct {
	Outcome  openrouter.AttemptOutcome
	Audio    []byte
	Duration time.Duration
	SHA256   [32]byte
	// Retained even when returned audio fails validation, for host reconciliation.
	GenerationID string
	RequestID    string
}

// Synthesize performs exactly one speech request after caller-owned authority and
// financial admission. It is not a worker, retry journal or publication boundary.
// The caller persists/reconciles the provider outcome before any retry. remaining
// is the rendition's remaining aggregate narration budget, not a per-part reset.
func Synthesize(ctx context.Context, p SpeechProvider, key, model, voice, script string, remaining time.Duration, maxBytes int) (result Synthesized, failure error) {
	state := openrouter.NotDispatched
	defer func() {
		observation := result.Outcome
		observation.State = state
		observation.RequestID, observation.GenerationID = result.RequestID, result.GenerationID
		result.Outcome, failure = openrouter.FinishAttempt(observation, failure)
		result.RequestID, result.GenerationID = result.Outcome.RequestID, result.Outcome.GenerationID
	}()
	if err := ctx.Err(); err != nil {
		return Synthesized{}, err
	}
	if p == nil || strings.TrimSpace(key) == "" || strings.TrimSpace(model) == "" || strings.TrimSpace(voice) == "" || !utf8.ValidString(script) || strings.TrimSpace(script) == "" || len(script) > 64000 || len(strings.Fields(script)) > 1100 || strings.Contains(script, "```") || remaining <= 0 || remaining > content.MaxNarrationDuration || maxBytes <= 0 || maxBytes > 64<<20 {
		return Synthesized{}, errors.New("invalid narration synthesis request")
	}
	state = openrouter.MayHaveActed
	out, err := p.Speech(ctx, key, model, voice, script)
	result = Synthesized{Outcome: out.Outcome, GenerationID: out.GenerationID, RequestID: out.RequestID}
	if err != nil {
		return result, err
	}
	state = openrouter.ResponseReceived
	duration, err := media.MeasureMP3(ctx, out.Audio, maxBytes, remaining)
	if err != nil {
		return result, err
	}
	result.Audio = append([]byte(nil), out.Audio...)
	result.Duration = duration
	result.SHA256 = sha256.Sum256(result.Audio)
	return result, nil
}
