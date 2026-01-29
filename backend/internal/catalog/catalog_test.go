package catalog

import "testing"

// --- NormalizeModelType tests ---

func TestNormalizeModelType(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"LLM", "llm"},
		{"  Embedding  ", "embedding"},
		{"IMAGE", "image"},
		{"", ""},
		{"audio_speech", "audio_speech"},
	}
	for _, tc := range cases {
		got := NormalizeModelType(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeModelType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- NormalizeProviderSlug tests ---

func TestNormalizeProviderSlug(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"OpenAI", "openai"},
		{"openai_compatible", "openai-compatible"},
		{"open-router", "openrouter"},
		{"open_router", "openrouter"},
		{"Groq", "groq"},
		{"", ""},
		{"  Azure  ", "azure"},
		{"custom-provider", "custom-provider"},
	}
	for _, tc := range cases {
		got := NormalizeProviderSlug(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeProviderSlug(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
