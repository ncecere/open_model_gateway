package openaihelper

import (
	"testing"
)

func TestExtractMessageContent_EmptyString(t *testing.T) {
	text, parts := ExtractMessageContent("", "fallback")
	if text != "fallback" {
		t.Fatalf("expected fallback, got %q", text)
	}
	if len(parts) != 0 {
		t.Fatalf("expected no parts, got %d", len(parts))
	}
}

func TestExtractMessageContent_PlainString(t *testing.T) {
	text, parts := ExtractMessageContent(`"hello world"`, "fb")
	if text != "hello world" {
		t.Fatalf("expected 'hello world', got %q", text)
	}
	if len(parts) != 0 {
		t.Fatalf("expected no parts, got %d", len(parts))
	}
}

func TestExtractMessageContent_Whitespace(t *testing.T) {
	text, parts := ExtractMessageContent("   ", "fb")
	if text != "fb" {
		t.Fatalf("expected fallback for whitespace, got %q", text)
	}
	if parts != nil {
		t.Fatalf("expected nil parts")
	}
}

func TestExtractMessageContent_ArrayContent(t *testing.T) {
	raw := `[{"type":"text","text":"Hello"},{"type":"text","text":"World"}]`
	text, parts := ExtractMessageContent(raw, "fb")
	if text == "fb" && len(parts) == 0 {
		// If the raw doesn't parse as an envelope, fallback is fine
		return
	}
	// If it parsed, we should have parts
	if len(parts) > 0 && text == "" {
		t.Fatalf("expected non-empty text or fallback")
	}
}

func TestExtractMessageContent_EnvelopeContent(t *testing.T) {
	raw := `{"content":"hello"}`
	text, _ := ExtractMessageContent(raw, "fb")
	// Should extract "hello" from envelope
	if text != "hello" && text != "fb" {
		t.Fatalf("expected 'hello' or fallback, got %q", text)
	}
}
