package briefing_test

import (
	"context"
	"errors"
	"github.com/narrate-it/prosody/internal/briefing"
	"github.com/narrate-it/prosody/internal/content"
	"github.com/narrate-it/prosody/internal/openrouter"
	"testing"
	"time"
)

type malformedProvider struct{ rewrite, speech int }

func (p *malformedProvider) Rewrite(context.Context, string, string, string, string, int) (openrouter.RewriteResult, error) {
	p.rewrite++
	var diagnostic *openrouter.AttemptError
	return openrouter.RewriteResult{Text: "private rejected script", RequestID: "request", GenerationID: "generation", PromptTokens: 20, CompletionTokens: 10}, diagnostic
}
func (p *malformedProvider) Speech(context.Context, string, string, string, string) (openrouter.SpeechResult, error) {
	p.speech++
	var diagnostic *openrouter.AttemptError
	return openrouter.SpeechResult{Audio: []byte("private rejected audio"), RequestID: "request", GenerationID: "generation"}, diagnostic
}
func TestBriefingTypedNilProviderDiagnostic(t *testing.T) {
	p := &malformedProvider{}
	rewritten, err := briefing.Rewrite(context.Background(), p, "key", selectedModel(), content.TextSubmission{WorkspaceID: "w", FeedID: "f", IdempotencyKey: "k", Text: "source"}, admittedBudget())
	var failure *openrouter.AttemptError
	if !errors.As(err, &failure) || failure == nil || rewritten.Provider.Outcome.State != openrouter.MayHaveActed || rewritten.Script != "" || rewritten.Provider.Text != "" || rewritten.Provider.RequestID != "request" || rewritten.Provider.GenerationID != "generation" || rewritten.Provider.PromptTokens != 20 || p.rewrite != 1 {
		t.Fatal("rewrite malformed diagnostic lost receipt or retried")
	}
	audio, err := briefing.Synthesize(context.Background(), p, "key", "model", "voice", "Script.", time.Second, 1024)
	if !errors.As(err, &failure) || failure == nil || audio.Outcome.State != openrouter.MayHaveActed || len(audio.Audio) != 0 || audio.RequestID != "request" || audio.GenerationID != "generation" || p.speech != 1 {
		t.Fatal("speech malformed diagnostic lost receipt or retried")
	}
	_ = errors.Is(err, context.Canceled)
}
