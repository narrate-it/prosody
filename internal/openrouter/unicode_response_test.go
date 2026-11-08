package openrouter_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/narrate-it/prosody/internal/briefing"
	"github.com/narrate-it/prosody/internal/openrouter"
)

// TestRewriteUnicodeEnvelope refuses lossy provider JSON through the real
// owned TLS client and briefing port. Complete responses remain observations,
// not settlement, and rejection never authorizes another paid request.
func TestRewriteUnicodeEnvelope(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, fragment, want string
		badMetadata          bool
	}{
		{"plain", `"A short briefing."`, "A short briefing.", false},
		{"paired escape", `"A briefing \ud83d\ude00."`, "A briefing 😀.", false},
		{"literal replacement rune", `"A briefing �."`, "A briefing �.", false},
		{"escaped replacement rune", `"A briefing \ufffd."`, "A briefing �.", false},
		{"escaped lookalike", `"Literal \\ud800 text."`, `Literal \ud800 text.`, false},
		{"invalid UTF8", "\"PRIVATE-SCRIPT-" + string([]byte{0xff}) + "\"", "", false},
		{"UTF8 encoded surrogate", "\"PRIVATE-SCRIPT-" + string([]byte{0xed, 0xa0, 0x80}) + "\"", "", false},
		{"lone high escape", `"PRIVATE-SCRIPT-\ud800"`, "", false},
		{"lone low escape", `"PRIVATE-SCRIPT-\udc00"`, "", false},
		{"high then ordinary escape", `"PRIVATE-SCRIPT-\ud800\u0041"`, "", false},
		{"high then text", `"PRIVATE-SCRIPT-\ud800x"`, "", false},
		{"malformed metadata", `"A short briefing."`, "", true},
	}
	for _, level := range []string{"client", "briefing"} {
		t.Run(level, func(t *testing.T) {
			t.Parallel()
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					body := fmt.Sprintf(`{"id":"body-generation","choices":[{"message":{"content":%s},"finish_reason":"stop"}],"usage":{"prompt_tokens":17,"completion_tokens":9}}`, tc.fragment)
					if tc.badMetadata {
						body = strings.Replace(body, `"body-generation"`, `"body-\ud800"`, 1)
					}
					var calls atomic.Int32
					server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if r.Method != http.MethodPost || r.URL.Path != "/api/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer "+httpPrivateKey {
							t.Error("request binding changed")
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						w.Header().Set("X-Request-Id", "header-request")
						w.Header().Set("X-Generation-Id", "header-generation")
						if _, err := w.Write([]byte(body)); err != nil {
							t.Error("owned response write failed")
						}
					}))
					t.Cleanup(server.Close)
					client := openrouter.NewClientForTLSServer(t, server)
					var result openrouter.RewriteResult
					var script string
					var err error
					if level == "client" {
						result, err = client.Rewrite(context.Background(), httpPrivateKey, "synthetic-text", "policy", "source", 1024)
						script = result.Text
					} else {
						out, failure := briefing.Rewrite(context.Background(), client, httpPrivateKey, httpTextModel(), httpSource(), briefing.TokenBudget{MaxCompletionTokens: 1024, CountPromptTokens: func(_, _, _ string) (int, error) { return 100, nil }})
						result, script, err = out.Provider, out.Script, failure
					}
					if calls.Load() != 1 {
						t.Fatal("unexpected provider retry or missing owned HTTP call")
					}
					if result.RequestID != "header-request" || result.Outcome.RequestID != "header-request" || result.Outcome.State != openrouter.ResponseReceived {
						t.Fatal("complete response observation lost")
					}
					if tc.want != "" {
						if err != nil || script != tc.want || result.Text != tc.want || result.GenerationID != "body-generation" || result.PromptTokens != 17 || result.CompletionTokens != 9 || result.Outcome.PromptTokens != 17 || result.Outcome.CompletionTokens != 9 {
							t.Fatal("valid Unicode control changed or lost accounting observations")
						}
						return
					}
					var attempt *openrouter.AttemptError
					if err == nil || script != "" || result.Text != "" || !errors.As(err, &attempt) || attempt == nil || attempt.Outcome != result.Outcome {
						t.Fatal("malformed Unicode became usable output or lost typed rejection")
					}
					// Default zero fields are unavailable observations, never zero-cost proof.
					if result.PromptTokens != 0 || result.CompletionTokens != 0 || result.Outcome.PromptTokens != 0 || result.Outcome.CompletionTokens != 0 {
						t.Fatal("rejected body usage was admitted as an accounting observation")
					}
					if result.GenerationID != "header-generation" || result.Outcome.GenerationID != "header-generation" || strings.Contains(err.Error(), "PRIVATE-") || strings.Contains(err.Error(), "body-") {
						t.Fatal("malformed body metadata admitted or diagnostics leaked input")
					}
				})
			}
		})
	}
}
