package openrouter

// SupportsMP3Speech is Prosody's reviewed compatibility policy, not a provider
// catalog capability field. The public speech catalog has no output-format
// metadata (checked 2026-11-02); speech modality alone cannot qualify MP3.
//
// https://openrouter.ai/docs/guides/overview/multimodal/tts documents MP3 for
// the dedicated endpoint, MiniMax as MP3-only, and Gemini/HeyGen as PCM-only.
// These exact IDs were present in that day's public speech catalog. Unknown IDs
// fail closed until reviewed; do not widen this to provider-prefix matching.
// This policy qualifies request format only, not availability, voice, billing,
// audio quality, or successful synthesis. Callers must also validate current
// catalog modalities/voices and retain response format/decoder checks.
func (m Model) SupportsMP3Speech() bool {
	switch m.ID {
	case "elevenlabs/eleven-v4-turbo",
		"minimax/speech-2.8-hd", "minimax/speech-2.8-turbo",
		"microsoft/mai-voice-2", "microsoft/mai-voice-2.1", "microsoft/mai-voice-2.1-flash",
		"mistralai/voxtral-mini-tts-2603", "mistralai/voxtral-mini-tts-3-0",
		"fish-audio/s1", "fish-audio/s2-pro", "fish-audio/s2.1-pro":
		return true
	default:
		return false
	}
}
