package configuration

import (
	"encoding/json"
	"testing"

	"github.com/narrate-it/prosody/internal/openrouter"
)

// TestSelectionRequiresKnownTextTokenLimits rejects missing/unknown completion
// metadata while preserving bounded models with supported output-token limits.
func TestSelectionRequiresKnownTextTokenLimits(t *testing.T) {
	for _, limits := range []string{
		`"top_provider":{"context_length":8192,"max_completion_tokens":4096},"supported_parameters":["max_tokens"]`,
		`"top_provider":{"context_length":8192,"max_completion_tokens":null},"supported_parameters":["max_tokens"]`,
		`"top_provider":{"context_length":8192},"supported_parameters":["max_tokens"]`,
		`"top_provider":{"context_length":8192,"max_completion_tokens":4096},"supported_parameters":[]`,
	} {
		var catalog []openrouter.Model
		if err := json.Unmarshal([]byte(`[{"id":"text","context_length":8192,`+limits+`,"architecture":{"input_modalities":["text"],"output_modalities":["text"]}},{"id":"elevenlabs/eleven-v4-turbo","supported_voices":["voice"],"architecture":{"input_modalities":["text"],"output_modalities":["speech"]}}]`), &catalog); err != nil {
			t.Fatal(err)
		}
		err := (Selection{TextModel: "text", SpeechModel: "elevenlabs/eleven-v4-turbo", Voice: "voice"}).Validate(catalog)
		wantValid := catalog[0].TopProvider.MaxCompletionTokens != nil && len(catalog[0].SupportedParameters) > 0
		if (err == nil) != wantValid {
			t.Fatalf("selection validity=%t err=%v", wantValid, err)
		}
	}
}

// TestSelectionRequiresDedicatedSpeechModalityAndListedVoice requires dedicated
// text-to-speech capability and a voice listed in the current model metadata.
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
			speech := openrouter.Model{ID: "elevenlabs/eleven-v4-turbo", Voices: tc.voices}
			speech.Architecture.Input = []string{tc.input}
			speech.Architecture.Output = []string{tc.output}
			err := (Selection{TextModel: "text", SpeechModel: "elevenlabs/eleven-v4-turbo", Voice: tc.selectedVoice}).Validate([]openrouter.Model{text, speech})
			if (err == nil) != tc.wantValid {
				t.Fatalf("valid=%t err=%v", tc.wantValid, err)
			}
		})
	}
}

// TestSelectionRejectsPCMOnlyAndUnreviewedSpeechFormats preserves the reviewed
// MP3 compatibility requirement even when speech and voice metadata are present.
func TestSelectionRejectsPCMOnlyAndUnreviewedSpeechFormats(t *testing.T) {
	for _, tc := range []struct {
		name, modelID string
		wantValid     bool
	}{
		{"reviewed MP3 with current voice", "elevenlabs/eleven-v4-turbo", true},
		{"reviewed alternative MP3", "minimax/speech-2.8-hd", true},
		{"PCM only Gemini", "google/gemini-3.8-flash-tts", false},
		{"PCM only HeyGen", "heygen/heygen-voice-1", false},
		{"unknown provider", "new-provider/new-tts", false},
		{"unknown model from reviewed provider", "elevenlabs/future-tts", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var catalog []openrouter.Model
			if err := json.Unmarshal([]byte(`[{"id":"text","context_length":8192,"top_provider":{"context_length":8192,"max_completion_tokens":4096},"supported_parameters":["max_tokens"],"architecture":{"input_modalities":["text"],"output_modalities":["text"]}},{"supported_voices":["voice"],"architecture":{"input_modalities":["text"],"output_modalities":["speech"]}}]`), &catalog); err != nil {
				t.Fatal(err)
			}
			catalog[1].ID = tc.modelID
			err := (Selection{TextModel: "text", SpeechModel: tc.modelID, Voice: "voice"}).Validate(catalog)
			if (err == nil) != tc.wantValid {
				t.Fatalf("MP3 selection valid=%t, want %t (error %v)", err == nil, tc.wantValid, err)
			}
		})
	}
}

// selectionIdentityFixture supplies synthetic current capabilities for the
// existing text-token, dedicated-speech, MP3 and voice validation requirements.
func selectionIdentityFixture(t *testing.T) (Selection, []openrouter.Model) {
	t.Helper()
	var catalog []openrouter.Model
	if err := json.Unmarshal([]byte(`[{"id":"text","context_length":8192,"top_provider":{"context_length":8192,"max_completion_tokens":4096},"supported_parameters":["max_tokens"],"architecture":{"input_modalities":["text"],"output_modalities":["text"]}},{"id":"elevenlabs/eleven-v4-turbo","supported_voices":["voice"],"architecture":{"input_modalities":["text"],"output_modalities":["speech"]}}]`), &catalog); err != nil {
		t.Fatal(err)
	}
	return Selection{TextModel: "text", SpeechModel: "elevenlabs/eleven-v4-turbo", Voice: "voice"}, catalog
}

// TestSelectionRejectsAmbiguousSelectedModelIDs rejects identical or conflicting
// rows for either selected ID regardless of row order or capability validity.
func TestSelectionRejectsAmbiguousSelectedModelIDs(t *testing.T) {
	t.Parallel()
	for _, role := range []string{"text", "speech"} {
		t.Run(role, func(t *testing.T) {
			t.Parallel()
			for _, conflicting := range []bool{false, true} {
				name := "identical"
				if conflicting {
					name = "conflicting"
				}
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					for _, reverse := range []bool{false, true} {
						name := "forward"
						if reverse {
							name = "reverse"
						}
						t.Run(name, func(t *testing.T) {
							t.Parallel()
							selected, catalog := selectionIdentityFixture(t)
							index := 0
							if role == "speech" {
								index = 1
							}
							duplicate := catalog[index]
							if conflicting {
								if role == "text" {
									duplicate.ContextLength = 0
								} else {
									duplicate.Voices = []string{"other-voice"}
								}
							}
							catalog = append(catalog, duplicate)
							if reverse {
								for i, j := 0, len(catalog)-1; i < j; i, j = i+1, j-1 {
									catalog[i], catalog[j] = catalog[j], catalog[i]
								}
							}
							if err := selected.Validate(catalog); err != ErrInvalid {
								t.Fatalf("ambiguous selected ID accepted or error changed: %v", err)
							}
						})
					}
				})
			}
		})
	}
}

// TestSelectionUniqueIdentityControls preserves unique catalogs, a single
// capable row serving both roles and the existing policy for unselected IDs.
func TestSelectionUniqueIdentityControls(t *testing.T) {
	t.Parallel()
	for _, tc := range []string{"unique distinct roles", "one row both roles", "unselected duplicates", "duplicated row both roles"} {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			selected, catalog := selectionIdentityFixture(t)
			wantInvalid := false
			switch tc {
			case "one row both roles", "duplicated row both roles":
				combined := catalog[0]
				combined.ID = selected.SpeechModel
				combined.Architecture.Output = []string{"text", "speech"}
				combined.Voices = []string{selected.Voice}
				selected.TextModel = selected.SpeechModel
				catalog = []openrouter.Model{combined}
				if tc == "duplicated row both roles" {
					catalog = append(catalog, combined)
					wantInvalid = true
				}
			case "unselected duplicates":
				first := catalog[0]
				first.ID = "unselected"
				second := first
				second.ContextLength = 0
				catalog = append(catalog, first, second)
			}
			err := selected.Validate(catalog)
			if wantInvalid {
				if err != ErrInvalid {
					t.Fatalf("duplicated shared-role row accepted: %v", err)
				}
			} else if err != nil {
				t.Fatalf("unique selection changed: %v", err)
			}
		})
	}
}
