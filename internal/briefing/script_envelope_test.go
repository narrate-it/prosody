package briefing_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/narrate-it/prosody/internal/briefing"
	"github.com/narrate-it/prosody/internal/content"
	"github.com/narrate-it/prosody/internal/openrouter"
)

// TestRewriteScriptEnvelope checks that a successful rewrite cannot return an
// encoding or byte length that speech preflight already refuses. Receipt
// observations remain available after rejection; they are not settlement proof.
func TestRewriteScriptEnvelope(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		accepted   bool
	}{
		{"ascii byte boundary", strings.Repeat("a", 64000), true},
		{"ascii byte overflow", strings.Repeat("a", 64001), false},
		{"multibyte byte boundary", strings.Repeat("界", 21333) + "a", true},
		{"multibyte byte overflow", strings.Repeat("界", 21333) + "ab", false},
		{"invalid UTF8", "PRIVATE-SCRIPT-" + string([]byte{0xff}), false},
		{"trim before byte boundary", " \n" + strings.Repeat("a", 64000) + "\t ", true},
		{"short script", "A short briefing.", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &provider{output: tc.text}
			source := content.TextSubmission{WorkspaceID: "w", FeedID: "f", IdempotencyKey: "k", Text: "Source"}
			got, err := briefing.Rewrite(context.Background(), p, "PRIVATE-KEY", selectedModel(), source, admittedBudget())
			if p.calls != 1 || got.Provider.RequestID != "request" || got.Provider.GenerationID != "generation" || got.Provider.PromptTokens != 20 || got.Provider.CompletionTokens != 10 || len(got.InstructionsDigest) != 64 {
				t.Fatal("rewrite retried or lost receipt/accounting references")
			}
			if got.Provider.Outcome.State != openrouter.ResponseReceived {
				t.Fatal("complete provider response lost its observation")
			}
			if tc.accepted {
				if err != nil || got.Script != strings.TrimSpace(tc.text) || got.Provider.Text != tc.text {
					t.Fatal("valid boundary script rejected or changed")
				}
				return
			}
			var attempt *openrouter.AttemptError
			if err == nil || got.Script != "" || got.Provider.Text != "" || !errors.As(err, &attempt) || attempt == nil || attempt.Outcome != got.Provider.Outcome {
				t.Fatal("invalid script exposed as usable output or rejection lost receipt")
			}
			if strings.Contains(err.Error(), "PRIVATE-") || strings.Contains(err.Error(), tc.text) {
				t.Fatal("diagnostic exposed private key or provider text")
			}
		})
	}
}
