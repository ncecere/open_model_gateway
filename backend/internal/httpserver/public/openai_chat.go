package public

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature *float32            `json:"temperature,omitempty"`
	TopP        *float32            `json:"top_p,omitempty"`
	MaxTokens   *int32              `json:"max_tokens,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	StopRaw     json.RawMessage     `json:"stop,omitempty"`
}

type openAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openAIChatChoice `json:"choices"`
	Usage   openAIUsage        `json:"usage"`
}

func (h *openAIHandler) chatCompletions(c *fiber.Ctx) error {
	var req openAIChatRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if len(req.Messages) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "messages are required")
	}
	stop, err := parseStop(req.StopRaw)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid stop field")
	}

	messages, err := convertOpenAIMessages(req.Messages, "message")
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	if !h.container.IsModelAllowed(rc.TenantID, req.Model) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	traceID := traceIDFromContext(c)
	alias := req.Model
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))

	modelReq := models.ChatRequest{
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stop:        stop,
	}

	if req.Stream {
		return h.chatStreamPipeline.Stream(c, rc, alias, traceID, idempotencyKey, modelReq)
	}

	return h.chatPipeline.Execute(c, rc, alias, traceID, idempotencyKey, modelReq)
}
