package vertex

import (
	"errors"
	"strings"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

func buildGenerateContentRequest(req models.ChatRequest) (vertexGenerateRequest, error) {
	if len(req.Messages) == 0 {
		return vertexGenerateRequest{}, errors.New("vertex: at least one message is required")
	}

	var systemParts []string
	contents := make([]vertexContent, 0, len(req.Messages))

	for _, msg := range req.Messages {
		text := strings.TrimSpace(msg.Content)
		if text == "" {
			continue
		}
		switch strings.ToLower(msg.Role) {
		case "system":
			systemParts = append(systemParts, text)
		case "assistant":
			contents = append(contents, vertexContent{
				Role:  "model",
				Parts: []vertexPart{{Text: text}},
			})
		case "user", "function", "tool", "developer":
			contents = append(contents, vertexContent{
				Role:  "user",
				Parts: []vertexPart{{Text: text}},
			})
		default:
			contents = append(contents, vertexContent{
				Role:  "user",
				Parts: []vertexPart{{Text: text}},
			})
		}
	}

	if len(contents) == 0 {
		return vertexGenerateRequest{}, errors.New("vertex: no user/assistant messages provided")
	}

	var systemInstruction *vertexContent
	if len(systemParts) > 0 {
		systemInstruction = &vertexContent{
			Role:  "system",
			Parts: []vertexPart{{Text: strings.Join(systemParts, "\n")}},
		}
	}

	cfg := &vertexGenerationConfig{}
	if req.MaxTokens != nil {
		cfg.MaxOutputTokens = req.MaxTokens
	}
	if req.Temperature != nil {
		cfg.Temperature = req.Temperature
	}
	if req.TopP != nil {
		cfg.TopP = req.TopP
	}
	if len(req.Stop) > 0 {
		cfg.StopSequences = append(cfg.StopSequences, req.Stop...)
	}

	if cfg.MaxOutputTokens == nil && cfg.Temperature == nil && cfg.TopP == nil && len(cfg.StopSequences) == 0 {
		cfg = nil
	}

	return vertexGenerateRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
		GenerationConfig:  cfg,
	}, nil
}
