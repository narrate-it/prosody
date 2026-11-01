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
	Architecture  struct {
		Input  []string `json:"input_modalities"`
		Output []string `json:"output_modalities"`
	} `json:"architecture"`
	Voices []string `json:"supported_voices"`
}

func (c *Client) request(ctx context.Context, key, method, path string, payload any, limit int64) ([]byte, http.Header, error) {
	if key == "" || strings.ContainsAny(key, "\r\n") {
		return nil, nil, errors.New("provider credential required")
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, nil, errors.New("provider transport failed; outcome may be unknown")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, res.Header, &ProviderError{Status: res.StatusCode, RequestID: res.Header.Get("X-Request-Id")}
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, res.Header, errors.New("provider response interrupted; outcome may be unknown")
	}
	if int64(len(b)) > limit {
		return nil, res.Header, errors.New("provider response exceeds limit")
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
	Text, GenerationID             string
	PromptTokens, CompletionTokens int
}

func (c *Client) Rewrite(ctx context.Context, key, model, instructions, source string, maxTokens int) (RewriteResult, error) {
	if model == "" || source == "" || maxTokens < 1 || maxTokens > 8192 {
		return RewriteResult{}, errors.New("invalid rewrite request")
	}
	payload := struct {
		Model     string              `json:"model"`
		Messages  []map[string]string `json:"messages"`
		MaxTokens int                 `json:"max_tokens"`
		Stream    bool                `json:"stream"`
	}{model, []map[string]string{{"role": "system", "content": instructions}, {"role": "user", "content": source}}, maxTokens, false}
	b, _, err := c.request(ctx, key, http.MethodPost, "/chat/completions", payload, 2<<20)
	if err != nil {
		return RewriteResult{}, err
	}
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
	if json.Unmarshal(b, &result) != nil || len(result.Choices) != 1 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return RewriteResult{}, errors.New("invalid rewrite response")
	}
	if result.Choices[0].FinishReason != "stop" {
		return RewriteResult{}, errors.New("incomplete rewrite response")
	}
	return RewriteResult{result.Choices[0].Message.Content, result.ID, result.Usage.Prompt, result.Usage.Completion}, nil
}

type SpeechResult struct {
	Audio                     []byte
	ContentType, GenerationID string
}

// Speech returns the original provider bytes. Publication must separately
// validate the codec and measured aggregate duration, never trust headers alone.
func (c *Client) Speech(ctx context.Context, key, model, voice, text string) (SpeechResult, error) {
	if model == "" || voice == "" || strings.TrimSpace(text) == "" || len(text) > 64000 {
		return SpeechResult{}, errors.New("invalid speech request")
	}
	b, h, err := c.request(ctx, key, http.MethodPost, "/audio/speech", map[string]string{"model": model, "voice": voice, "input": text, "response_format": "mp3"}, 64<<20)
	if err != nil {
		return SpeechResult{}, err
	}
	if len(b) == 0 || !strings.HasPrefix(h.Get("Content-Type"), "audio/") {
		return SpeechResult{}, errors.New("invalid speech response")
	}
	return SpeechResult{b, h.Get("Content-Type"), h.Get("X-Generation-Id")}, nil
}
