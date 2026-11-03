package briefing_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/narrate-it/prosody/internal/briefing"
	"github.com/narrate-it/prosody/internal/content"
	"github.com/narrate-it/prosody/internal/openrouter"
)

type failureOnlyProvider struct{ err error }

func (p failureOnlyProvider) Rewrite(context.Context, string, string, string, string, int) (openrouter.RewriteResult, error) {
	return openrouter.RewriteResult{}, p.err
}

func TestRewritePreservesErrorOnlyAccounting(t *testing.T) {
	for _, state := range []openrouter.AttemptState{openrouter.MayHaveActed, "", "private-source-key-body"} {
		t.Run(string(state), func(t *testing.T) {
			failure := &openrouter.AttemptError{Outcome: openrouter.AttemptOutcome{State: state, RequestID: "request", GenerationID: "generation", PromptTokens: 50, CompletionTokens: 12}}
			p := failureOnlyProvider{err: failure}
			result, err := briefing.Rewrite(context.Background(), p, "key", selectedModel(), content.TextSubmission{WorkspaceID: "w", FeedID: "f", IdempotencyKey: "k", Text: "source"}, admittedBudget())
			var attempt *openrouter.AttemptError
			if !errors.As(err, &attempt) || attempt.Outcome.PromptTokens != 50 || attempt.Outcome.CompletionTokens != 12 || result.Provider.Outcome != attempt.Outcome || result.Provider.RequestID != "request" || result.Provider.GenerationID != "generation" || result.Provider.PromptTokens != 50 || result.Provider.CompletionTokens != 12 || attempt.Outcome.State != openrouter.MayHaveActed || attempt.Outcome.RequestID != "request" || attempt.Outcome.GenerationID != "generation" {
				t.Fatalf("lost error-only receipt: %+v", result.Provider.Outcome)
			}
			if strings.Contains(err.Error(), "private") || strings.Contains(failure.Error(), "private") || result.Script != "" || result.Provider.Text != "" {
				t.Fatal("unsafe error or output")
			}
		})
	}
}

func (p failureOnlyProvider) Speech(context.Context, string, string, string, string) (openrouter.SpeechResult, error) {
	return openrouter.SpeechResult{}, p.err
}
func TestSynthesisBackfillsErrorOnlyReferences(t *testing.T) {
	p := failureOnlyProvider{err: &openrouter.AttemptError{Outcome: openrouter.AttemptOutcome{State: openrouter.MayHaveActed, RequestID: "request", GenerationID: "generation"}}}
	result, err := briefing.Synthesize(context.Background(), p, "key", "model", "voice", "Script.", time.Second, 1024)
	if err == nil || result.RequestID != "request" || result.GenerationID != "generation" || len(result.Audio) != 0 {
		t.Fatalf("lost flat receipt: %+v, %v", result, err)
	}
}
