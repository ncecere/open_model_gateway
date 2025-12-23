package public

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type openAIEmbeddingRequest struct {
	Model string          `json:"model"`
	Input json.RawMessage `json:"input"`
}

type openAIEmbedding struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
	Object    string    `json:"object"`
}

type openAIEmbeddingResponse struct {
	Object string            `json:"object"`
	Model  string            `json:"model"`
	Data   []openAIEmbedding `json:"data"`
	Usage  openAIUsage       `json:"usage"`
}

func (h *openAIHandler) embeddings(c *fiber.Ctx) error {
	var req openAIEmbeddingRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	inputs, err := parseEmbeddingInput(req.Input)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid input field")
	}

	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	modelReq := models.EmbeddingsRequest{
		Model: req.Model,
		Input: inputs,
	}

	return h.embeddingPipeline.Execute(c, rc, req.Model, modelReq)
}
