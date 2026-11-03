// Package openrouter implements bounded customer-key provider calls. Callers own
// current authorization, cost admission and durable request reconciliation.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const endpoint = "https://openrouter.ai/api/v1"

type Client struct {
	http *http.Client
	base string
}

// New does not discover credentials or retry paid calls. Redirects are refused
// so customer credentials cannot leave the configured provider origin.
func New() *Client {
	return &Client{http: &http.Client{Timeout: 120 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, base: endpoint}
}

type ProviderError struct {
	Status    int
	RequestID string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("OpenRouter request failed (HTTP %d)", e.Status)
}

type Model struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextLength int    `json:"context_length"`
	TopProvider   struct {
		ContextLength       *int `json:"context_length"`
		MaxCompletionTokens *int `json:"max_completion_tokens"`
	} `json:"top_provider"`
	SupportedParameters []string `json:"supported_parameters"`
	Architecture        struct {
		Input  []string `json:"input_modalities"`
		Output []string `json:"output_modalities"`
	} `json:"architecture"`
	Voices []string `json:"supported_voices"`
}

func (c *Client) request(ctx context.Context, key, method, path string, payload any, limit int64) ([]byte, http.Header, error) {
	if key == "" || strings.ContainsAny(key, "\r\n") {
		_, failure := FinishAttempt(AttemptOutcome{State: NotDispatched}, errors.New("provider credential required"))
		return nil, nil, failure
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			_, failure := FinishAttempt(AttemptOutcome{State: NotDispatched}, err)
			return nil, nil, failure
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		_, failure := FinishAttempt(AttemptOutcome{State: NotDispatched}, err)
		return nil, nil, failure
	}
	req.Header.Set("Authorization", "Bearer "+key)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if err := ctx.Err(); err != nil {
		_, failure := FinishAttempt(AttemptOutcome{State: NotDispatched}, err)
		return nil, nil, failure
	}
	res, err := c.http.Do(req)
	if err != nil {
		_, failure := FinishAttempt(AttemptOutcome{State: MayHaveActed}, fmt.Errorf("provider transport failed: %w", err))
		return nil, nil, failure
	}
	// Body read/status determines the outcome; cleanup cannot undo a paid call.
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, failure := FinishAttempt(AttemptOutcome{State: MayHaveActed, Status: res.StatusCode, RequestID: res.Header.Get("X-Request-Id"), GenerationID: res.Header.Get("X-Generation-Id")}, &ProviderError{Status: res.StatusCode, RequestID: res.Header.Get("X-Request-Id")})
		return nil, res.Header, failure
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		_, failure := FinishAttempt(AttemptOutcome{State: MayHaveActed}, fmt.Errorf("provider response interrupted: %w", err))
		return nil, res.Header, failure
	}
	if int64(len(b)) > limit {
		_, failure := FinishAttempt(AttemptOutcome{State: MayHaveActed}, errors.New("provider response exceeds limit"))
		return nil, res.Header, failure
	}
	return b, res.Header, nil
}

func (c *Client) Models(ctx context.Context, key string) ([]Model, error) {
	b, _, err := c.request(ctx, key, http.MethodGet, "/models", nil, 8<<20)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []Model `json:"data"`
	}
	if json.Unmarshal(b, &result) != nil {
		return nil, errors.New("invalid model catalog")
	}
	return result.Data, nil
}

type RewriteResult struct {
	Outcome                        AttemptOutcome
	Text, GenerationID, RequestID  string
	PromptTokens, CompletionTokens int
}

func (c *Client) Rewrite(ctx context.Context, key, model, instructions, source string, maxTokens int) (out RewriteResult, failure error) {
	state := NotDispatched
	defer func() {
		out.Outcome, failure = FinishAttempt(AttemptOutcome{State: state, RequestID: out.RequestID, GenerationID: out.GenerationID, PromptTokens: out.PromptTokens, CompletionTokens: out.CompletionTokens}, failure)
	}()
	if model == "" || source == "" || maxTokens < 1 || maxTokens > MaxRewriteCompletionTokens {
		return RewriteResult{}, errors.New("invalid rewrite request")
	}
	payload := struct {
		Model     string              `json:"model"`
		Messages  []map[string]string `json:"messages"`
		MaxTokens int                 `json:"max_tokens"`
		Stream    bool                `json:"stream"`
	}{model, []map[string]string{{"role": "system", "content": instructions}, {"role": "user", "content": source}}, maxTokens, false}
	state = MayHaveActed
	b, h, err := c.request(ctx, key, http.MethodPost, "/chat/completions", payload, 2<<20)
	receipt := RewriteResult{GenerationID: h.Get("X-Generation-Id"), RequestID: h.Get("X-Request-Id")}
	if err != nil {
		return receipt, err
	}
	state = ResponseReceived
	var result struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			Prompt     int `json:"prompt_tokens"`
			Completion int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(b, &result) != nil {
		return receipt, errors.New("invalid rewrite response")
	}
	if result.ID != "" {
		receipt.GenerationID = result.ID
	}
	receipt.PromptTokens, receipt.CompletionTokens = result.Usage.Prompt, result.Usage.Completion
	if len(result.Choices) != 1 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return receipt, errors.New("invalid rewrite response")
	}
	if result.Choices[0].FinishReason != "stop" {
		return receipt, errors.New("incomplete rewrite response")
	}
	receipt.Text = result.Choices[0].Message.Content
	return receipt, nil
}

type SpeechResult struct {
	Outcome                              AttemptOutcome
	Audio                                []byte
	ContentType, GenerationID, RequestID string
}

// Speech returns the original provider bytes. Publication must separately
// validate the codec and measured aggregate duration, never trust headers alone.
func (c *Client) Speech(ctx context.Context, key, model, voice, text string) (out SpeechResult, failure error) {
	state := NotDispatched
	defer func() {
		out.Outcome, failure = FinishAttempt(AttemptOutcome{State: state, RequestID: out.RequestID, GenerationID: out.GenerationID}, failure)
	}()
	if model == "" || voice == "" || strings.TrimSpace(text) == "" || len(text) > 64000 {
		return SpeechResult{}, errors.New("invalid speech request")
	}
	state = MayHaveActed
	b, h, err := c.request(ctx, key, http.MethodPost, "/audio/speech", map[string]string{"model": model, "voice": voice, "input": text, "response_format": "mp3"}, 64<<20)
	receipt := SpeechResult{ContentType: h.Get("Content-Type"), GenerationID: h.Get("X-Generation-Id"), RequestID: h.Get("X-Request-Id")}
	if err != nil {
		return receipt, err
	}
	state = ResponseReceived
	if len(b) == 0 || !strings.HasPrefix(h.Get("Content-Type"), "audio/") {
		return receipt, errors.New("invalid speech response")
	}
	receipt.Audio = b
	return receipt, nil
}
