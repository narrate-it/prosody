package configuration

import (
	"encoding/json"
	"testing"

	"github.com/narrate-it/prosody/internal/openrouter"
)

func TestSelectionRequiresKnownTextTokenLimits(t *testing.T) {
	for _, limits := range []string{
		`"top_provider":{"context_length":8192,"max_completion_tokens":4096},"supported_parameters":["max_tokens"]`,
		`"top_provider":{"context_length":8192,"max_completion_tokens":null},"supported_parameters":["max_tokens"]`,
		`"top_provider":{"context_length":8192},"supported_parameters":["max_tokens"]`,
		`"top_provider":{"context_length":8192,"max_completion_tokens":4096},"supported_parameters":[]`,
	} {
		var catalog []openrouter.Model
		if err := json.Unmarshal([]byte(`[{"id":"text","context_length":8192,`+limits+`,"architecture":{"input_modalities":["text"],"output_modalities":["text"]}},{"id":"speech","supported_voices":["voice"],"architecture":{"input_modalities":["text"],"output_modalities":["speech"]}}]`), &catalog); err != nil {
			t.Fatal(err)
		}
		err := (Selection{TextModel: "text", SpeechModel: "speech", Voice: "voice"}).Validate(catalog)
		wantValid := catalog[0].TopProvider.MaxCompletionTokens != nil && len(catalog[0].SupportedParameters) > 0
		if (err == nil) != wantValid {
			t.Fatalf("selection validity=%t err=%v", wantValid, err)
		}
	}
}

func TestSelectionRequiresDedicatedSpeechModalityAndListedVoice(t *testing.T) {
	for _, tc := range []struct {
		name, input, output, selectedVoice string
		voices                             []string
		wantValid                          bool
	}{
		{"text to speech", "text", "speech", "listed-voice", []string{"listed-voice"}, true},
		{"chat text to audio", "text", "audio", "listed-voice", []string{"listed-voice"}, false},
		{"chat audio to audio", "audio", "audio", "listed-voice", []string{"listed-voice"}, false},
		{"no text input", "audio", "speech", "listed-voice", []string{"listed-voice"}, false},
		{"unknown voice", "text", "speech", "other-voice", []string{"listed-voice"}, false},
		{"no voice metadata", "text", "speech", "listed-voice", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var text openrouter.Model
			if err := json.Unmarshal([]byte(`{"id":"text","context_length":8192,"top_provider":{"context_length":8192,"max_completion_tokens":4096},"supported_parameters":["max_tokens"],"architecture":{"input_modalities":["text"],"output_modalities":["text"]}}`), &text); err != nil {
				t.Fatal(err)
			}
			speech := openrouter.Model{ID: "speech", Voices: tc.voices}
			speech.Architecture.Input = []string{tc.input}
			speech.Architecture.Output = []string{tc.output}
			err := (Selection{TextModel: "text", SpeechModel: "speech", Voice: tc.selectedVoice}).Validate([]openrouter.Model{text, speech})
			if (err == nil) != tc.wantValid {
				t.Fatalf("valid=%t err=%v", tc.wantValid, err)
			}
		})
	}
}
