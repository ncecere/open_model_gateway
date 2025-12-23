package public

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type openAIModerationRequest struct {
	Model string          `json:"model"`
	Input json.RawMessage `json:"input"`
}

type openAIModerationResponse struct {
	ID      string                   `json:"id"`
	Model   string                   `json:"model"`
	Results []openAIModerationResult `json:"results"`
}

type openAIModerationResult struct {
	Categories                models.ModerationCategories                `json:"categories"`
	CategoryAppliedInputTypes models.ModerationCategoryAppliedInputTypes `json:"category_applied_input_types"`
	CategoryScores            models.ModerationCategoryScores            `json:"category_scores"`
	Flagged                   bool                                       `json:"flagged"`
}

func (h *openAIHandler) moderations(c *fiber.Ctx) error {
	var req openAIModerationRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	alias := strings.TrimSpace(req.Model)
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	inputs, err := parseEmbeddingInput(req.Input)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "input must be string or array of strings")
	}
	if len(inputs) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "input is required")
	}

	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	return h.moderationPipeline.Execute(c, rc, alias, inputs)
}
