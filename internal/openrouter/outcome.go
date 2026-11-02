package openrouter

import "errors"

// AttemptState describes observations, never permission to retry or a billing
// verdict. ResponseReceived means a complete successful HTTP response was read;
// it does not certify usable content, final accounting or publication authority.
type AttemptState string

const (
	NotDispatched    AttemptState = "not_dispatched"
	ResponseReceived AttemptState = "response_received"
	MayHaveActed     AttemptState = "may_have_acted"
)

// AttemptOutcome retains available references even when content is rejected.
// Empty references or zero token counts are not proof of no provider effect.
type AttemptOutcome struct {
	State                          AttemptState
	Status                         int
	RequestID, GenerationID        string
	PromptTokens, CompletionTokens int
}

// AttemptError has a fixed safe rendering. The wrapped cause supports errors.Is
// and errors.As; callers must not log arbitrary provider-port causes or payloads.
type AttemptError struct {
	Outcome AttemptOutcome
	cause   error
}

func (e *AttemptError) Error() string {
	return "provider attempt failed (" + string(normalizeState(e.Outcome.State)) + ")"
}
func (e *AttemptError) Unwrap() error { return e.cause }

// FinishAttempt preserves a provider's typed observation. An untyped error after
// a port handoff is conservatively unknown. This helper does not schedule work.
func FinishAttempt(outcome AttemptOutcome, err error) (AttemptOutcome, error) {
	outcome.State = normalizeState(outcome.State)
	if err == nil {
		return outcome, nil
	}
	var prior *AttemptError
	if errors.As(err, &prior) {
		outcome.State = normalizeState(prior.Outcome.State)
		outcome.Status = prior.Outcome.Status
		if outcome.PromptTokens == 0 {
			outcome.PromptTokens = prior.Outcome.PromptTokens
		}
		if outcome.CompletionTokens == 0 {
			outcome.CompletionTokens = prior.Outcome.CompletionTokens
		}
		if outcome.RequestID == "" {
			outcome.RequestID = prior.Outcome.RequestID
		}
		if outcome.GenerationID == "" {
			outcome.GenerationID = prior.Outcome.GenerationID
		}
	}
	return outcome, &AttemptError{Outcome: outcome, cause: err}
}

func normalizeState(state AttemptState) AttemptState {
	switch state {
	case NotDispatched, ResponseReceived, MayHaveActed:
		return state
	default:
		return MayHaveActed
	}
}
