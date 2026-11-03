package openrouter

import (
	"errors"
	"fmt"
	"testing"
)

func TestTypedNilAttemptDiagnostic(t *testing.T) {
	var malformed *AttemptError
	sentinel := errors.New("local sentinel")
	for _, diagnostic := range []error{malformed, fmt.Errorf("wrapped: %w", malformed), errors.Join(malformed, sentinel)} {
		t.Run(fmt.Sprintf("%T", diagnostic), func(t *testing.T) {
			observation, failure := FinishAttempt(AttemptOutcome{State: ResponseReceived, RequestID: "request", GenerationID: "generation", PromptTokens: 20}, diagnostic)
			if observation.State != MayHaveActed || observation.RequestID != "request" || observation.GenerationID != "generation" || observation.PromptTokens != 20 {
				t.Fatalf("lost conservative receipt: %+v", observation)
			}
			var got *AttemptError
			if !errors.As(failure, &got) || got == nil || got.Outcome != observation {
				t.Fatal("missing typed receipt")
			}
			if failure.Error() != "provider attempt failed (may_have_acted)" {
				t.Fatal("unsafe rendering")
			}
			if errors.Is(diagnostic, sentinel) != errors.Is(failure, sentinel) {
				t.Fatal("sentinel changed")
			}
			_ = errors.Is(failure, errors.New("absent"))
			var other *ProviderError
			if errors.As(failure, &other) {
				t.Fatal("unexpected error type")
			}
		})
	}
}

func TestNilAttemptMethodsAreSafe(t *testing.T) {
	var malformed *AttemptError
	if malformed.Error() != "provider attempt failed (may_have_acted)" || malformed.Unwrap() != nil {
		t.Fatal("nil receiver unsafe")
	}
	if errors.Is(malformed, errors.New("absent")) {
		t.Fatal("unexpected match")
	}
	var other *ProviderError
	if errors.As(malformed, &other) {
		t.Fatal("unexpected type")
	}
}
