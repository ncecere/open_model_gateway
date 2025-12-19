package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseMessageContent extracts both the legacy string content and the structured
// content parts from a JSON payload where OpenAI allows either a plain string or
// an array of content part objects.
func ParseMessageContent(raw json.RawMessage) (string, []MessageContentPart, error) {
	if IsNullJSON(raw) {
		return "", nil, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil, nil
	}
	var parts []MessageContentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		return TextFromContentParts(parts), parts, nil
	}
	return "", nil, fmt.Errorf("content must be a string or array of content parts")
}

// MarshalMessageContent encodes a ChatMessage into the OpenAI wire format by
// preferring structured content parts when present, falling back to the legacy
// content string or reasoning text when necessary.
func MarshalMessageContent(msg ChatMessage) json.RawMessage {
	if len(msg.ContentParts) > 0 && msg.HasNonTextContent() {
		if data, err := json.Marshal(msg.ContentParts); err == nil {
			return data
		}
	}
	value := msg.Text()
	if data, err := json.Marshal(value); err == nil {
		return data
	}
	return json.RawMessage(`""`)
}

// IsNullJSON reports whether the raw message is empty/null so callers can avoid
// trying to unmarshal missing payloads.
func IsNullJSON(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return len(raw) == 0 || s == "" || strings.EqualFold(s, "null")
}
