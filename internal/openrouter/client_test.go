package openrouter

import (
	"context"
	"encoding/json"
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
			Messages []map[string]string `json:"messages"`
			Model    string              `json:"model"`
		}
		if json.NewDecoder(r.Body).Decode(&p) != nil || len(p.Messages) != 2 || p.Messages[0]["role"] != "system" || p.Messages[1]["content"] != "untrusted source" || p.Model != "selected" {
			t.Error("source/instruction boundary")
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "generation", "choices": []any{map[string]any{"message": map[string]string{"content": "Briefing."}, "finish_reason": finish}}})
	}))
	defer s.Close()
	c := New()
	c.base = s.URL
	res, err := c.Rewrite(context.Background(), "synthetic", "selected", "craft", "untrusted source", 400)
	if err != nil || res.Text != "Briefing." {
		t.Fatalf("rewrite: %v", err)
	}
	finish = "length"
	if _, err = c.Rewrite(context.Background(), "synthetic", "selected", "craft", "untrusted source", 400); err == nil {
		t.Fatal("truncated script accepted")
	}
}
func TestNoRedirectOrPaidRetry(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Location", "/other")
		w.WriteHeader(http.StatusTemporaryRedirect)
		w.Write([]byte("sensitive-provider-body"))
	}))
	defer s.Close()
	c := New()
	c.base = s.URL
	_, err := c.Speech(context.Background(), "synthetic", "model", "voice", "script")
	if err == nil || calls != 1 || strings.Contains(err.Error(), "sensitive-provider-body") {
		t.Fatal("redirect, retry or error disclosure")
	}
}
