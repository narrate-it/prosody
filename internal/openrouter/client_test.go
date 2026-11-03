package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRewriteSeparatesSourceAndRejectsTruncation(t *testing.T) {
	finish := "stop"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer synthetic" {
			t.Error("request binding")
		}
		var p struct {
			Messages  []map[string]string `json:"messages"`
			Model     string              `json:"model"`
			MaxTokens int                 `json:"max_tokens"`
		}
		if json.NewDecoder(r.Body).Decode(&p) != nil || len(p.Messages) != 2 || p.Messages[0]["role"] != "system" || p.Messages[1]["content"] != "untrusted source" || p.Model != "selected" || p.MaxTokens != 400 {
			t.Error("source/instruction boundary")
		}
		w.Header().Set("X-Request-Id", "request")
		if err := json.NewEncoder(w).Encode(map[string]any{"id": "generation", "usage": map[string]int{"prompt_tokens": 20, "completion_tokens": 10}, "choices": []any{map[string]any{"message": map[string]string{"content": "Briefing."}, "finish_reason": finish}}}); err != nil {
			t.Errorf("write rewrite fixture: %v", err)
		}
	}))
	defer s.Close()
	c := New()
	c.base = s.URL
	res, err := c.Rewrite(context.Background(), "synthetic", "selected", "craft", "untrusted source", 400)
	if err != nil || res.Text != "Briefing." {
		t.Fatalf("rewrite: %v", err)
	}
	finish = "length"
	res, err = c.Rewrite(context.Background(), "synthetic", "selected", "craft", "untrusted source", 400)
	if err == nil || res.Text != "" || res.GenerationID != "generation" || res.RequestID != "request" || res.PromptTokens != 20 || res.CompletionTokens != 10 {
		t.Fatal("truncated script accepted or receipt lost")
	}
}

type receiptTransport func(*http.Request) (*http.Response, error)

func (f receiptTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type interruptedBody struct{}

func (interruptedBody) Read([]byte) (int, error) { return 0, errors.New("private body failure") }
func (interruptedBody) Close() error             { return nil }

func TestSpeechRetainsHeadersOnFailure(t *testing.T) {
	for _, mode := range []string{"interrupted", "invalid type", "empty", "denied"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			c := New()
			c.http.Transport = receiptTransport(func(*http.Request) (*http.Response, error) {
				calls++
				body := io.ReadCloser(io.NopCloser(strings.NewReader("private invalid bytes")))
				contentType, status := "audio/mpeg", http.StatusOK
				switch mode {
				case "interrupted":
					body = interruptedBody{}
				case "invalid type":
					contentType = "application/json"
				case "empty":
					body = io.NopCloser(strings.NewReader(""))
				case "denied":
					status = http.StatusForbidden
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {contentType}, "X-Generation-Id": {"generation"}, "X-Request-Id": {"request"}}, Body: body}, nil
			})
			got, err := c.Speech(context.Background(), "synthetic", "model", "voice", "script")
			if err == nil || calls != 1 || len(got.Audio) != 0 || got.GenerationID != "generation" || got.RequestID != "request" || strings.Contains(err.Error(), "private") {
				t.Fatal("receipt lost, invalid content exposed, or automatic retry")
			}
		})
	}
}
func TestNoRedirectOrPaidRetry(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Location", "/other")
		w.WriteHeader(http.StatusTemporaryRedirect)
		if _, err := w.Write([]byte("sensitive-provider-body")); err != nil {
			t.Errorf("write denial fixture: %v", err)
		}
	}))
	defer s.Close()
	c := New()
	c.base = s.URL
	_, err := c.Speech(context.Background(), "synthetic", "model", "voice", "script")
	if err == nil || calls != 1 || strings.Contains(err.Error(), "sensitive-provider-body") {
		t.Fatal("redirect, retry or error disclosure")
	}
}

func TestModelCatalogRetainsNullableTokenLimits(t *testing.T) {
	calls := 0
	c := New()
	c.http.Transport = receiptTransport(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method != http.MethodGet || request.URL.Path != "/api/v1/models" {
			t.Fatal("unexpected catalog request")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"bounded","context_length":8192,"top_provider":{"context_length":8192,"max_completion_tokens":4096},"supported_parameters":["max_tokens"],"architecture":{"input_modalities":["text"],"output_modalities":["text"]}},{"id":"unknown","context_length":8192,"top_provider":{"context_length":8192,"max_completion_tokens":null}}]}`))}, nil
	})
	models, err := c.Models(context.Background(), "synthetic")
	if err != nil || calls != 1 || len(models) != 2 {
		t.Fatalf("catalog result: %v", err)
	}
	if models[0].TopProvider.MaxCompletionTokens == nil || *models[0].TopProvider.MaxCompletionTokens != 4096 || models[1].TopProvider.MaxCompletionTokens != nil {
		t.Fatal("completion limits lost or unknown limit defaulted")
	}
	cap, err := models[0].RewriteTokenLimit(7000, 8192)
	if err != nil || cap != 1192 {
		t.Fatalf("remaining context cap=%d err=%v", cap, err)
	}
	if _, err := models[1].RewriteTokenLimit(100, 8192); err == nil {
		t.Fatal("unknown completion bound admitted")
	}
}

func TestSpeechCatalogQueryAndDefaultCatalogIsolation(t *testing.T) {
	for _, speech := range []bool{false, true} {
		name := "default catalog"
		if speech {
			name = "speech catalog"
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			client := New()
			client.http.Transport = receiptTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				if request.Method != http.MethodGet || request.URL.Path != "/api/v1/models" || request.Header.Get("Authorization") != "Bearer synthetic" {
					t.Fatal("unexpected catalog route or credential binding")
				}
				wantQuery := ""
				if speech {
					wantQuery = "output_modalities=speech"
				}
				if request.URL.RawQuery != wantQuery {
					t.Fatalf("query=%q want=%q", request.URL.RawQuery, wantQuery)
				}
				body := `{"data":[{"id":"catalog-text","architecture":{"input_modalities":["text"],"output_modalities":["text"]}}]}`
				if speech {
					body = `{"data":[{"id":"catalog-tts","supported_voices":["voice-from-provider"],"architecture":{"input_modalities":["text"],"output_modalities":["speech"]}}]}`
				}
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			var models []Model
			var err error
			if speech {
				models, err = client.SpeechModels(context.Background(), "synthetic")
			} else {
				models, err = client.Models(context.Background(), "synthetic")
			}
			if err != nil || calls != 1 || len(models) != 1 {
				t.Fatalf("catalog calls=%d models=%d err=%v", calls, len(models), err)
			}
			if speech {
				if models[0].ID != "catalog-tts" || len(models[0].Voices) != 1 || models[0].Voices[0] != "voice-from-provider" || len(models[0].Architecture.Output) != 1 || models[0].Architecture.Output[0] != "speech" {
					t.Fatal("speech model metadata was lost or replaced")
				}
			} else if models[0].ID != "catalog-text" || len(models[0].Voices) != 0 {
				t.Fatal("default catalog behavior changed")
			}
		})
	}
}

func TestSpeechCatalogFailureRetainsAttemptOutcome(t *testing.T) {
	calls := 0
	client := New()
	client.http.Transport = receiptTransport(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.RawQuery != "output_modalities=speech" {
			t.Fatal("speech catalog query lost")
		}
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"X-Request-Id": {"catalog-request"}, "X-Generation-Id": {"catalog-generation"}},
			Body:       io.NopCloser(strings.NewReader("private catalog failure")),
		}, nil
	})
	models, err := client.SpeechModels(context.Background(), "synthetic")
	var attempt *AttemptError
	if !errors.As(err, &attempt) || attempt == nil {
		t.Fatalf("catalog lost typed outcome: %v", err)
	}
	if calls != 1 || len(models) != 0 || attempt.Outcome.State != MayHaveActed || attempt.Outcome.Status != http.StatusForbidden || attempt.Outcome.RequestID != "catalog-request" || attempt.Outcome.GenerationID != "catalog-generation" || strings.Contains(err.Error(), "private") {
		t.Fatal("catalog receipt lost, private diagnostic exposed, or request retried")
	}
}
