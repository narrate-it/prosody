package openrouter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaidAttemptObservations(t *testing.T) {
	for _, operation := range []string{"rewrite", "speech"} {
		for _, mode := range []string{"preflight", "drop", "truncated", "http failure", "decode", "success"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if mode == "drop" {
						conn, _, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						if err := conn.Close(); err != nil {
							t.Error(err)
						}
						return
					}
					w.Header().Set("X-Request-Id", "request")
					w.Header().Set("X-Generation-Id", "generation")
					if mode == "truncated" {
						w.Header().Set("Content-Length", "500")
					}
					if mode == "http failure" {
						w.WriteHeader(http.StatusBadGateway)
					}
					body := "private-source-key-body"
					if mode == "success" {
						if operation == "rewrite" {
							body = `{"id":"generation","choices":[{"message":{"content":"Briefing"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":10}}`
						} else {
							w.Header().Set("Content-Type", "audio/mpeg")
							body = "audio"
						}
					}
					if _, err := fmt.Fprint(w, body); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				client := New()
				client.base = server.URL
				key := "private-source-key-body"
				if mode == "preflight" {
					key = ""
				}
				var observation AttemptOutcome
				var err error
				usable := false
				if operation == "rewrite" {
					var result RewriteResult
					result, err = client.Rewrite(context.Background(), key, "model", "policy", "private-source-key-body", 20)
					observation, usable = result.Outcome, result.Text != ""
				} else {
					var result SpeechResult
					result, err = client.Speech(context.Background(), key, "model", "voice", "private-source-key-body")
					observation, usable = result.Outcome, len(result.Audio) > 0
				}
				want := MayHaveActed
				if mode == "preflight" {
					want = NotDispatched
				}
				if mode == "decode" || mode == "success" {
					want = ResponseReceived
				}
				if observation.State != want {
					t.Fatalf("outcome %+v, want %s", observation, want)
				}
				wantCalls := 1
				if mode == "preflight" {
					wantCalls = 0
				}
				if calls != wantCalls {
					t.Fatalf("calls %d, want %d", calls, wantCalls)
				}
				if mode == "success" {
					if err != nil || !usable {
						t.Fatalf("success: %v", err)
					}
					return
				}
				var failure *AttemptError
				if !errors.As(err, &failure) || failure.Outcome != observation || usable {
					t.Fatalf("failed outcome %+v, %v usable=%v", observation, err, usable)
				}
				if strings.Contains(err.Error(), "private") {
					t.Fatal("payload leaked")
				}
				if mode != "preflight" && mode != "drop" && (observation.RequestID != "request" || observation.GenerationID != "generation") {
					t.Fatal("lost references")
				}
			})
		}
	}
}
