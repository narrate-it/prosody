package openrouter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPaidAttemptObservations(t *testing.T) {
	for _, operation := range []string{"rewrite", "speech"} {
		for _, mode := range []string{"preflight", "drop", "canceled after acceptance", "truncated", "http failure", "decode", "success"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var calls atomic.Int32
				release := make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if mode == "canceled after acceptance" {
						cancel()
						select {
						case <-r.Context().Done():
						case <-release:
						}
						return
					}
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
				defer close(release)
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
					result, err = client.Rewrite(ctx, key, "model", "policy", "private-source-key-body", 20)
					observation, usable = result.Outcome, result.Text != ""
				} else {
					var result SpeechResult
					result, err = client.Speech(ctx, key, "model", "voice", "private-source-key-body")
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
				if int(calls.Load()) != wantCalls {
					t.Fatalf("calls %d, want %d", calls.Load(), wantCalls)
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
				if mode != "preflight" && mode != "drop" && mode != "canceled after acceptance" && (observation.RequestID != "request" || observation.GenerationID != "generation") {
					t.Fatal("lost references")
				}
			})
		}
	}
}

type deadlineTransport struct{ calls int }

func (d *deadlineTransport) RoundTrip(*http.Request) (*http.Response, error) {
	d.calls++
	return nil, context.DeadlineExceeded
}

func TestDeadlineAfterTransportHandoffIsUnknown(t *testing.T) {
	transport := &deadlineTransport{}
	client := New()
	client.http.Transport = transport
	result, err := client.Rewrite(context.Background(), "synthetic", "model", "policy", "source", 20)
	var failure *AttemptError
	if transport.calls != 1 || !errors.As(err, &failure) || result.Outcome.State != MayHaveActed || failure.Outcome != result.Outcome || result.Text != "" {
		t.Fatalf("deadline outcome lost: %+v, %v", result.Outcome, err)
	}
}
