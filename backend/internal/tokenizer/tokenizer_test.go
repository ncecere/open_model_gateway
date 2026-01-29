package tokenizer

import (
	"testing"
)

func TestCountTokens_OpenAI(t *testing.T) {
	text := "Hello, world! This is a test of the tokenizer."
	n, err := CountTokens("openai", text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n <= 0 {
		t.Fatalf("expected positive token count, got %d", n)
	}
	// cl100k_base should tokenize this to roughly 11 tokens
	if n < 8 || n > 20 {
		t.Errorf("token count %d outside expected range [8,20] for %q", n, text)
	}
}

func TestCountTokens_Anthropic(t *testing.T) {
	text := "Hello, world!"
	n, err := CountTokens("anthropic", text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n <= 0 {
		t.Fatalf("expected positive token count, got %d", n)
	}
}

func TestCountTokens_UnknownProvider(t *testing.T) {
	// Unknown providers should fall back to cl100k_base, not error
	n, err := CountTokens("some-unknown-provider", "Hello")
	if err != nil {
		t.Fatalf("unexpected error for unknown provider: %v", err)
	}
	if n <= 0 {
		t.Fatalf("expected positive token count, got %d", n)
	}
}

func TestMustCountTokens_Fallback(t *testing.T) {
	// MustCountTokens should never panic
	n := MustCountTokens("openai", "Hello, world!")
	if n <= 0 {
		t.Fatalf("expected positive token count, got %d", n)
	}
}

func TestForProvider_Caching(t *testing.T) {
	enc1, err := ForProvider("openai")
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := ForProvider("openai")
	if err != nil {
		t.Fatal(err)
	}
	// Should return the same cached instance
	if enc1 != enc2 {
		t.Error("expected same encoder instance from cache")
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	n := EstimateMessageTokens("openai", "user", "Hello, world!")
	// Should be content tokens + role tokens + 4 overhead
	if n < 5 {
		t.Errorf("expected at least 5 tokens for message, got %d", n)
	}
}

func TestForEncoding_O200k(t *testing.T) {
	enc, err := ForEncoding(O200kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n := enc.CountTokens("Hello, world!")
	if n <= 0 {
		t.Fatalf("expected positive token count, got %d", n)
	}
}

func TestForEncoding_CL100k(t *testing.T) {
	enc, err := ForEncoding(CL100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n := enc.CountTokens("Hello, world!")
	if n <= 0 {
		t.Fatalf("expected positive token count, got %d", n)
	}
}
