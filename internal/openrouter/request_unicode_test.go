package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// requestUnicodeClient exercises the real client against an owned TLS double.
// Speech bytes are synthetic transport data, not qualified MP3 content.
func requestUnicodeClient(t *testing.T, want []string) (*Client, *atomic.Int32) {
	t.Helper()
	calls := new(atomic.Int32)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body struct {
			Model    string `json:"model"`
			Voice    string `json:"voice"`
			Input    string `json:"input"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode fixture request: %v", err)
		}
		w.Header().Set("X-Request-Id", "synthetic-request")
		w.Header().Set("X-Generation-Id", "synthetic-generation")
		switch r.URL.Path {
		case "/chat/completions":
			if want != nil && (body.Model != want[0] || len(body.Messages) != 2 || body.Messages[0].Content != want[1] || body.Messages[1].Content != want[2]) {
				t.Error("rewrite values changed in transport")
			}
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"id":"synthetic-generation","choices":[{"message":{"content":"Synthetic rewrite."},"finish_reason":"stop"}]}`)); err != nil {
				t.Errorf("write fixture response: %v", err)
			}
		case "/audio/speech":
			if want != nil && (body.Model != want[0] || body.Voice != want[1] || body.Input != want[2]) {
				t.Error("speech values changed in transport")
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			if _, err := w.Write([]byte{1, 2, 3}); err != nil {
				t.Errorf("write fixture response: %v", err)
			}
		default:
			t.Error("unexpected provider path")
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	client := New()
	client.base = server.URL
	client.http = server.Client()
	client.http.Timeout = 5 * time.Second
	t.Cleanup(client.http.CloseIdleConnections)
	return client, calls
}

// TestProviderRequestUnicodeAdmission rejects lossy JSON encoding before transport.
func TestProviderRequestUnicodeAdmission(t *testing.T) {
	malformed := []struct {
		name  string
		bytes []byte
	}{
		{"invalid-leading", []byte{0xff}},
		{"truncated", []byte{0xe2, 0x82}},
		{"surrogate", []byte{0xed, 0xa0, 0x80}},
		{"overlong", []byte{0xc0, 0xaf}},
		{"continuation", []byte{0x80}},
	}
	for _, mode := range []string{"rewrite", "speech"} {
		for field := 0; field < 3; field++ {
			for _, bad := range malformed {
				fieldName := []string{"model", "instructions", "source"}[field]
				if mode == "speech" {
					fieldName = []string{"model", "voice", "text"}[field]
				}
				t.Run(mode+"/"+fieldName+"/"+bad.name, func(t *testing.T) {
					client, calls := requestUnicodeClient(t, nil)
					values := []string{"selected", "instruction-or-voice", "original source"}
					values[field] = "private-invalid-prefix" + string(bad.bytes)
					if mode == "rewrite" {
						out, err := client.Rewrite(context.Background(), "synthetic", values[0], values[1], values[2], 100)
						if err == nil || out.Outcome.State != NotDispatched || out.Text != "" {
							t.Errorf("malformed rewrite admitted: state=%s error=%v", out.Outcome.State, err)
						}
						if err != nil && strings.Contains(err.Error(), "private-invalid-prefix") {
							t.Error("private input rendered in error")
						}
					} else {
						out, err := client.Speech(context.Background(), "synthetic", values[0], values[1], values[2])
						if err == nil || out.Outcome.State != NotDispatched || len(out.Audio) != 0 {
							t.Errorf("malformed speech admitted: state=%s error=%v", out.Outcome.State, err)
						}
						if err != nil && strings.Contains(err.Error(), "private-invalid-prefix") {
							t.Error("private input rendered in error")
						}
					}
					if got := calls.Load(); got != 0 {
						t.Errorf("pretransport refusal made %d calls", got)
					}
				})
			}
		}
	}
}

// TestProviderValidRequestUnicodePreserved retains legitimate text without normalization.
func TestProviderValidRequestUnicodePreserved(t *testing.T) {
	valid := []struct{ name, value string }{
		{"replacement", "literal \ufffd"},
		{"non-bmp", "literal \U0001f680"},
		{"nul", "literal\x00text"},
		{"backslash-lookalike", `literal \ud800`},
	}
	for _, mode := range []string{"rewrite", "speech"} {
		for _, tc := range valid {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				want := []string{"selected-" + tc.value, "instruction-or-voice-" + tc.value, "source-" + tc.value}
				client, calls := requestUnicodeClient(t, want)
				if mode == "rewrite" {
					out, err := client.Rewrite(context.Background(), "synthetic", want[0], want[1], want[2], 100)
					if err != nil || out.Text != "Synthetic rewrite." || out.Outcome.State != ResponseReceived {
						t.Errorf("valid rewrite rejected: %v", err)
					}
				} else {
					out, err := client.Speech(context.Background(), "synthetic", want[0], want[1], want[2])
					if err != nil || len(out.Audio) != 3 || out.Outcome.State != ResponseReceived {
						t.Errorf("valid speech rejected: %v", err)
					}
				}
				if got := calls.Load(); got != 1 {
					t.Errorf("valid input made %d calls", got)
				}
			})
		}
	}
}
