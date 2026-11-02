package openrouter

import (
	"errors"
	"slices"
	"strings"
)

const MaxRewriteCompletionTokens = 8192

var ErrTokenBudget = errors.New("model token limits or admitted rewrite budget unavailable")

// TextLimits uses explicit current catalog limits; missing/null values do not
// imply an unlimited context or output allowance. Top-provider metadata alone
// does not qualify every routed provider; the host still owns route admission.
func (m Model) TextLimits() (contextTokens, completionTokens int, err error) {
	if strings.TrimSpace(m.ID) == "" || m.ContextLength <= 0 ||
		m.TopProvider.ContextLength == nil || *m.TopProvider.ContextLength <= 0 ||
		m.TopProvider.MaxCompletionTokens == nil || *m.TopProvider.MaxCompletionTokens <= 0 ||
		!slices.Contains(m.Architecture.Input, "text") ||
		!slices.Contains(m.Architecture.Output, "text") ||
		!slices.Contains(m.SupportedParameters, "max_tokens") {
		return 0, 0, ErrTokenBudget
	}
	return min(m.ContextLength, *m.TopProvider.ContextLength), *m.TopProvider.MaxCompletionTokens, nil
}

// RewriteTokenLimit bounds visible output plus any provider reasoning tokens.
// promptTokens must include the full messages and model/provider framing from a
// qualified counter; admittedTokens is a caller-owned financial ceiling, not a
// grant of authority. No model switch, retry or guessed default occurs here.
func (m Model) RewriteTokenLimit(promptTokens, admittedTokens int) (int, error) {
	contextTokens, completionTokens, err := m.TextLimits()
	if err != nil || promptTokens <= 0 || promptTokens >= contextTokens || admittedTokens <= 0 || admittedTokens > MaxRewriteCompletionTokens {
		return 0, ErrTokenBudget
	}
	return min(admittedTokens, completionTokens, contextTokens-promptTokens), nil
}
