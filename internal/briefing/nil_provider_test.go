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

type missingPort interface {
	briefing.Provider
	briefing.SpeechProvider
}
type nilCallCounter struct{}
type nilReceiverPort struct{}
type nilFunctionPort func()

// These nil receivers could execute code; missing-port validation must reject
// them BEFORE invoking either method, not recover after a handoff/panic.
func (*nilReceiverPort) Rewrite(ctx context.Context, _, _, _, _ string, _ int) (openrouter.RewriteResult, error) {
	countNilCall(ctx)
	return openrouter.RewriteResult{}, nil
}
func (*nilReceiverPort) Speech(ctx context.Context, _, _, _, _ string) (openrouter.SpeechResult, error) {
	countNilCall(ctx)
	return openrouter.SpeechResult{}, nil
}
func (nilFunctionPort) Rewrite(ctx context.Context, _, _, _, _ string, _ int) (openrouter.RewriteResult, error) {
	countNilCall(ctx)
	return openrouter.RewriteResult{}, nil
}
func (nilFunctionPort) Speech(ctx context.Context, _, _, _, _ string) (openrouter.SpeechResult, error) {
	countNilCall(ctx)
	return openrouter.SpeechResult{}, nil
}
func countNilCall(ctx context.Context) { calls := ctx.Value(nilCallCounter{}).(*int); *calls++ }

func missingPorts() []struct {
	name string
	port missingPort
} {
	var client *openrouter.Client
	var receiver *nilReceiverPort
	var function nilFunctionPort
	return []struct {
		name string
		port missingPort
	}{{"plain nil", nil}, {"actual nil client", client}, {"nil receiver", receiver}, {"nil function", function}}
}

func TestMissingRewriteProviderRejectedBeforeAccountingOrDispatch(t *testing.T) {
	t.Parallel()
	for _, tc := range missingPorts() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			calls, tokens := 0, 0
			ctx := context.WithValue(context.Background(), nilCallCounter{}, &calls)
			budget := admittedBudget()
			budget.CountPromptTokens = func(_, _, _ string) (int, error) { tokens++; return 100, nil }
			source := content.TextSubmission{WorkspaceID: "w", FeedID: "f", IdempotencyKey: "synthetic", Text: "PRIVATE-SOURCE-SENTINEL"}
			// Recover exists only in this regression test so old nil-client panics are
			// reported as a RED assertion while the rest of the matrix still executes.
			func() {
				defer func() {
					if recover() != nil {
						t.Error("missing provider panicked")
					}
				}()
				out, err := briefing.Rewrite(ctx, tc.port, "PRIVATE-KEY-SENTINEL", selectedModel(), source, budget)
				assertMissingObservation(t, err, out.Provider.Outcome)
				if out.Script != "" || out.Provider.Text != "" || out.Provider.RequestID != "" || out.Provider.GenerationID != "" || out.InstructionsDigest != "" {
					t.Error("missing provider exposed usable output/identifiers")
				}
			}()
			if tokens != 0 || calls != 0 {
				t.Errorf("missing provider reached accounting=%d dispatch=%d", tokens, calls)
			}
		})
	}
}

func TestMissingSpeechProviderRejectedBeforeDispatch(t *testing.T) {
	t.Parallel()
	for _, tc := range missingPorts() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			ctx := context.WithValue(context.Background(), nilCallCounter{}, &calls)
			func() {
				defer func() {
					if recover() != nil {
						t.Error("missing provider panicked")
					}
				}()
				out, err := briefing.Synthesize(ctx, tc.port, "PRIVATE-KEY-SENTINEL", "synthetic", "synthetic", "PRIVATE-SOURCE-SENTINEL", time.Second, 1<<20)
				assertMissingObservation(t, err, out.Outcome)
				if len(out.Audio) != 0 || out.Duration != 0 || out.SHA256 != [32]byte{} || out.RequestID != "" || out.GenerationID != "" {
					t.Error("missing provider exposed usable output/identifiers")
				}
			}()
			if calls != 0 {
				t.Errorf("missing provider reached dispatch=%d", calls)
			}
		})
	}
}

func assertMissingObservation(t *testing.T, err error, outcome openrouter.AttemptOutcome) {
	t.Helper()
	var attempt *openrouter.AttemptError
	if err == nil || !errors.As(err, &attempt) || attempt == nil || attempt.Outcome != outcome || outcome.State != openrouter.NotDispatched || outcome.RequestID != "" || outcome.GenerationID != "" || outcome.PromptTokens != 0 || outcome.CompletionTokens != 0 {
		t.Error("missing provider lacked typed pre-handoff observation")
	}
	if err != nil && (strings.Contains(err.Error(), "PRIVATE-KEY-SENTINEL") || strings.Contains(err.Error(), "PRIVATE-SOURCE-SENTINEL")) {
		t.Error("private input exposed in error")
	}
}

func TestNilProviderRetainsContextCause(t *testing.T) {
	t.Parallel()
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "deadline"}[deadline], func(t *testing.T) {
			t.Parallel()
			var ctx context.Context
			var cancel context.CancelFunc
			expected := context.Canceled
			if deadline {
				ctx, cancel = context.WithDeadline(context.Background(), time.Unix(1, 0))
				expected = context.DeadlineExceeded
			} else {
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			defer cancel()
			var client *openrouter.Client
			rewrite, err := briefing.Rewrite(ctx, client, "private", selectedModel(), content.TextSubmission{}, admittedBudget())
			if !errors.Is(err, expected) || rewrite.Provider.Outcome.State != openrouter.NotDispatched {
				t.Fatal("rewrite context cause changed")
			}
			speech, err := briefing.Synthesize(ctx, client, "private", "model", "voice", "script", time.Second, 1<<20)
			if !errors.Is(err, expected) || speech.Outcome.State != openrouter.NotDispatched {
				t.Fatal("speech context cause changed")
			}
		})
	}
}
