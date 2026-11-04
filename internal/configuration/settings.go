// Package configuration validates immutable model/voice selections independently
// of transport. Catalog compatibility does not qualify model narration quality.
package configuration

import (
	"errors"
	"slices"
	"strings"

	"github.com/narrate-it/prosody/internal/openrouter"
)

var ErrInvalid = errors.New("invalid provider configuration")

type Selection struct {
	TextModel   string `json:"textModel"`
	SpeechModel string `json:"speechModel"`
	Voice       string `json:"voice"`
}

// Validate requires exactly one catalog row for each selected model ID before
// accepting its capabilities. One row may serve both roles; unselected IDs do
// not affect validation. Catalog identity does not establish current authority.
func (s Selection) Validate(catalog []openrouter.Model) error {
	if strings.TrimSpace(s.TextModel) == "" || strings.TrimSpace(s.SpeechModel) == "" || strings.TrimSpace(s.Voice) == "" {
		return ErrInvalid
	}
	text, speech := false, false
	textMatches, speechMatches := 0, 0
	for _, model := range catalog {
		if model.ID == s.TextModel {
			textMatches++
			_, _, err := model.TextLimits()
			text = err == nil
		}
		if model.ID == s.SpeechModel {
			speechMatches++
			if model.SupportsMP3Speech() && slices.Contains(model.Architecture.Input, "text") && slices.Contains(model.Architecture.Output, "speech") && slices.Contains(model.Voices, s.Voice) {
				speech = true
			}
		}
	}
	if textMatches != 1 || speechMatches != 1 || !text || !speech {
		return ErrInvalid
	}
	return nil
}

type Snapshot struct {
	Selection
	SettingsRevision   int64  `json:"settingsRevision"`
	CredentialID       string `json:"credentialId"`
	CredentialRevision int64  `json:"credentialRevision"`
}
