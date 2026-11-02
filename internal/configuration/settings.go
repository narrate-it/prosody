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

func (s Selection) Validate(catalog []openrouter.Model) error {
	if strings.TrimSpace(s.TextModel) == "" || strings.TrimSpace(s.SpeechModel) == "" || strings.TrimSpace(s.Voice) == "" {
		return ErrInvalid
	}
	text, speech := false, false
	for _, model := range catalog {
		if model.ID == s.TextModel {
			_, _, err := model.TextLimits()
			text = err == nil
		}
		if model.ID == s.SpeechModel && slices.Contains(model.Architecture.Input, "text") && slices.Contains(model.Architecture.Output, "audio") && slices.Contains(model.Voices, s.Voice) {
			speech = true
		}
	}
	if !text || !speech {
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
