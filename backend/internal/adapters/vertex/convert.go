package vertex

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

func vertexContentToMessage(content vertexContent) models.ChatMessage {
	role := strings.ToLower(strings.TrimSpace(content.Role))
	switch role {
	case "model", "assistant", "":
		role = "assistant"
	case "user":
		role = "user"
	default:
		role = "assistant"
	}

	// Extract tool calls from functionCall parts
	toolCalls := extractToolCallsFromParts(content.Parts)

	// Convert remaining parts to message content
	parts := vertexPartsToMessageParts(content.Parts)
	text := models.TextFromContentParts(parts)

	return models.ChatMessage{
		Role:         role,
		Content:      text,
		ContentParts: parts,
		ToolCalls:    toolCalls,
	}
}

// extractToolCallsFromParts extracts function calls from Vertex parts and converts to OpenAI tool calls.
func extractToolCallsFromParts(parts []vertexPart) []models.ToolCall {
	var toolCalls []models.ToolCall
	for i, part := range parts {
		if part.FunctionCall != nil {
			// Serialize args to JSON string
			argsJSON := "{}"
			if part.FunctionCall.Args != nil {
				if b, err := json.Marshal(part.FunctionCall.Args); err == nil {
					argsJSON = string(b)
				}
			}

			// Generate a unique ID for the tool call
			callID := fmt.Sprintf("call_%s", uuid.NewString()[:8])

			toolCalls = append(toolCalls, models.ToolCall{
				ID:    callID,
				Type:  "function",
				Index: intPtr(i),
				Function: models.ToolCallFunction{
					Name:      part.FunctionCall.Name,
					Arguments: argsJSON,
				},
			})
		}
	}
	return toolCalls
}

func intPtr(i int) *int {
	return &i
}

func vertexPartsToMessageParts(parts []vertexPart) []models.MessageContentPart {
	converted := make([]models.MessageContentPart, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part.Text) != "" {
			converted = append(converted, models.MessageContentPart{
				Type: models.MessageContentPartTypeText,
				Text: part.Text,
			})
			continue
		}
		if mp := inlineDataToContentPart(part.InlineData); mp != nil {
			converted = append(converted, *mp)
		}
	}
	return converted
}

func inlineDataToContentPart(inline *vertexInlineData) *models.MessageContentPart {
	if inline == nil {
		return nil
	}
	data := strings.TrimSpace(inline.Data)
	if data == "" {
		return nil
	}
	mime := strings.TrimSpace(inline.MimeType)
	if mime == "" {
		mime = "application/octet-stream"
	}
	if strings.HasPrefix(mime, "image/") {
		return &models.MessageContentPart{
			Type: models.MessageContentPartTypeImageURL,
			ImageURL: &models.MessageContentImageURL{
				URL: fmt.Sprintf("data:%s;base64,%s", mime, data),
			},
		}
	}
	return &models.MessageContentPart{
		Type: models.MessageContentPartTypeText,
		Text: fmt.Sprintf("[inline %s data omitted]", mime),
	}
}
