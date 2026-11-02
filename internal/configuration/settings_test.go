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
		if err := json.Unmarshal([]byte(`[{"id":"text","context_length":8192,`+limits+`,"architecture":{"input_modalities":["text"],"output_modalities":["text"]}},{"id":"speech","supported_voices":["voice"],"architecture":{"input_modalities":["text"],"output_modalities":["audio"]}}]`), &catalog); err != nil {
			t.Fatal(err)
		}
		err := (Selection{TextModel: "text", SpeechModel: "speech", Voice: "voice"}).Validate(catalog)
		wantValid := catalog[0].TopProvider.MaxCompletionTokens != nil && len(catalog[0].SupportedParameters) > 0
		if (err == nil) != wantValid {
			t.Fatalf("selection validity=%t err=%v", wantValid, err)
		}
	}
}
