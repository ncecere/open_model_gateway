package openaihelper

import (
	"encoding/json"
	"strings"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

// ExtractMessageContent decodes OpenAI-style message payloads or raw content
// fragments into legacy text plus structured content parts.
func ExtractMessageContent(raw string, fallback string) (string, []models.MessageContentPart) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	var envelope struct {
		Content json.RawMessage `json:"content"`
	}
	// If raw is an entire message envelope, peel out the content first.
	if err := json.Unmarshal([]byte(raw), &envelope); err == nil && len(envelope.Content) > 0 {
		raw = strings.TrimSpace(string(envelope.Content))
	}
	if raw == "" {
		return fallback, nil
	}
	content := json.RawMessage(raw)
	text, parts, err := models.ParseMessageContent(content)
	if err != nil {
		return fallback, nil
	}
	if strings.TrimSpace(text) == "" {
		text = fallback
	}
	return text, parts
}
