package public

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

func convertOpenAIMessages(msgs []openAIChatMessage, label string) ([]models.ChatMessage, error) {
	if label == "" {
		label = "message"
	}
	sanitized := make([]models.ChatMessage, 0, len(msgs))
	for idx, m := range msgs {
		role := strings.ToLower(m.Role)
		if role == "" {
			role = "user"
		}
		textContent, parts, err := models.ParseMessageContent(m.Content)
		if err != nil {
			return nil, fmt.Errorf("invalid content for %s %d: %v", label, idx, err)
		}
		sanitized = append(sanitized, models.ChatMessage{
			Role:         role,
			Content:      textContent,
			ContentParts: parts,
			Name:         m.Name,
		})
	}
	return sanitized, nil
}

func parseStop(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return []string{str}, nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("invalid stop value")
}

func parseEmbeddingInput(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, errors.New("input is required")
	}

	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return []string{str}, nil
	}

	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	return nil, errors.New("input must be string or array of strings")
}

func traceIDFromContext(c *fiber.Ctx) string {
	if v := c.Locals("requestid"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func errMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func convertChatResponse(resp models.ChatResponse, alias string) openAIChatResponse {
	choices := make([]openAIChatChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		msg := openAIChatMessage{
			Role:      choice.Message.Role,
			Content:   models.MarshalMessageContent(choice.Message),
			Reasoning: choice.Message.Reasoning,
		}
		choices = append(choices, openAIChatChoice{
			Index:        choice.Index,
			Message:      msg,
			FinishReason: choice.FinishReason,
		})
	}

	return openAIChatResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: resp.Created.Unix(),
		Model:   alias,
		Choices: choices,
		Usage: openAIUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			ReasoningTokens:  resp.Usage.ReasoningTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}

func convertEmbeddingResponse(resp models.EmbeddingsResponse, alias string) openAIEmbeddingResponse {
	data := make([]openAIEmbedding, 0, len(resp.Embeddings))
	for _, emb := range resp.Embeddings {
		data = append(data, openAIEmbedding{
			Index:     emb.Index,
			Embedding: emb.Vector,
			Object:    "embedding",
		})
	}

	return openAIEmbeddingResponse{
		Object: "list",
		Model:  alias,
		Data:   data,
		Usage: openAIUsage{
			PromptTokens: resp.Usage.PromptTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}
}

func convertStreamChunk(chunk models.ChatChunk, alias string) openAIStreamChunk {
	choices := make([]openAIStreamChoice, 0, len(chunk.Choices))
	for _, choice := range chunk.Choices {
		delta := openAIStreamDelta{
			Role:      choice.Delta.Role,
			Content:   models.MarshalMessageContent(choice.Delta),
			Reasoning: choice.Delta.Reasoning,
		}
		choices = append(choices, openAIStreamChoice{
			Index:        choice.Index,
			Delta:        delta,
			FinishReason: choice.FinishReason,
		})
	}

	return openAIStreamChunk{
		ID:      chunk.ID,
		Object:  "chat.completion.chunk",
		Created: chunk.Created.Unix(),
		Model:   alias,
		Choices: choices,
	}
}

func convertModerationHTTPResponse(resp models.ModerationResponse, alias string) openAIModerationResponse {
	results := make([]openAIModerationResult, 0, len(resp.Results))
	for _, item := range resp.Results {
		results = append(results, openAIModerationResult{
			Categories:                item.Categories,
			CategoryAppliedInputTypes: item.CategoryAppliedInputTypes,
			CategoryScores:            item.CategoryScores,
			Flagged:                   item.Flagged,
		})
	}
	return openAIModerationResponse{
		ID:      resp.ID,
		Model:   alias,
		Results: results,
	}
}

func convertImageResponse(resp models.ImageResponse) openAIImageResponse {
	data := make([]openAIImageData, 0, len(resp.Data))
	for _, item := range resp.Data {
		data = append(data, openAIImageData{
			B64JSON:       item.B64JSON,
			URL:           item.URL,
			RevisedPrompt: item.RevisedPrompt,
		})
	}

	created := resp.Created.Unix()
	if created < 0 {
		created = 0
	}

	return openAIImageResponse{
		Created: created,
		Data:    data,
	}
}
