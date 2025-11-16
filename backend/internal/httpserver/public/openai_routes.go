package public

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

type openAIHandler struct {
	container *app.Container
	executor  *executor.Executor
}

type imageOperationType string

const (
	imageOperationGeneration imageOperationType = "generation"
	imageOperationEdit       imageOperationType = "edit"
	imageOperationVariation  imageOperationType = "variation"
)

const (
	maxImageUploadBytes = 4 * 1024 * 1024
	maxImageInputs      = 16
)

func (h *openAIHandler) listModels(c *fiber.Ctx) error {
	ctx := c.UserContext()
	var rc *requestctx.Context
	if rctx, ok := requestctx.FromContext(ctx); ok {
		rc = rctx
		if status, err := h.container.UsageLogger.CheckBudget(ctx, rctx, time.Now().UTC()); err == nil {
			setBudgetHeaders(c, status)
		}
	}

	aliases := h.container.Engine.ListAliases()
	models := make([]openAIModel, 0, len(aliases))
	now := time.Now().Unix()

	for alias, routes := range aliases {
		if len(routes) == 0 {
			continue
		}
		if rc != nil && !h.container.IsModelAllowed(rc.TenantID, alias) {
			continue
		}
		route := routes[0]
		deployment := route.Metadata["deployment"]
		models = append(models, openAIModel{
			ID:         alias,
			Object:     "model",
			OwnedBy:    route.Provider,
			Created:    now,
			Deployment: deployment,
		})
	}

	return c.JSON(openAIModelList{
		Object: "list",
		Data:   models,
	})
}

type imageOperationConfig struct {
	Alias          string
	IdempotencyKey string
	Operation      imageOperationType
	Builder        func(route providers.Route) (models.ImageResponse, error)
}

func (h *openAIHandler) runImageOperation(c *fiber.Ctx, cfg imageOperationConfig) error {
	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	alias := strings.TrimSpace(cfg.Alias)
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if !h.container.IsModelAllowed(rc.TenantID, alias) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	operation := cfg.Operation
	if operation == "" {
		operation = imageOperationGeneration
	}

	routes := h.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}

	traceID := traceIDFromContext(c)
	initialBudget, err := h.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if initialBudget.Exceeded {
		setBudgetHeaders(c, initialBudget)
		_, _ = h.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
		return httputil.WriteError(c, fiber.StatusForbidden, "tenant budget exceeded")
	}
	setBudgetHeaders(c, initialBudget)

	idempotencyKey := strings.TrimSpace(cfg.IdempotencyKey)
	if idempotencyKey != "" {
		if data, ok := h.container.Idempotency.Get(ctx, idempotencyKey); ok {
			c.Set("Content-Type", "application/json")
			return c.Send(data)
		}
	}

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := h.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return httputil.WriteError(c, fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer release()

	var lastErr error
	var lastRoute providers.Route
	for _, route := range routes {
		if route.Image == nil {
			continue
		}
		lastRoute = route
		start := time.Now()
		resp, err := cfg.Builder(route)
		if err != nil {
			if errors.Is(err, models.ErrImageOperationUnsupported) {
				continue
			}
			h.container.Engine.ReportFailure(alias, route)
			lastErr = err
			continue
		}
		h.container.Engine.ReportSuccess(alias, route)

		tokensUsed := int(resp.Usage.TotalTokens)
		if tokensUsed > 0 {
			if err := h.container.RateLimiter.TokenAllowance(context.Background(), keyKey, tokensUsed, keyCfg); err != nil {
				release()
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
			}
			if err := h.container.RateLimiter.TokenAllowance(context.Background(), tenantKey, tokensUsed, tenantCfg); err != nil {
				release()
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
			}
		}

		record := usagepipeline.Record{
			Context:           rc,
			Alias:             alias,
			Provider:          route.Provider,
			Usage:             resp.Usage,
			Latency:           time.Since(start),
			Status:            fiber.StatusOK,
			TraceID:           traceID,
			Timestamp:         time.Now().UTC(),
			Success:           true,
			IdempotencyKey:    idempotencyKey,
			OverrideCostCents: parseImageOverrideCost(route.Metadata, operation),
		}
		budgetStatus, err := h.container.UsageLogger.Record(ctx, record)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to persist usage")
		}
		setBudgetHeaders(c, budgetStatus)

		payload, err := json.Marshal(convertImageResponse(resp))
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to encode response")
		}
		if idempotencyKey != "" {
			h.container.Idempotency.Set(ctx, idempotencyKey, payload)
		}

		c.Set("Content-Type", "application/json")
		return c.Send(payload)
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	h.container.Engine.ReportFailure(alias, lastRoute)
	_, _ = h.container.UsageLogger.Record(ctx, usagepipeline.Record{
		Context:   rc,
		Alias:     alias,
		Provider:  lastRoute.Provider,
		Status:    fiber.StatusBadGateway,
		ErrorCode: errMessage(lastErr),
		TraceID:   traceID,
		Timestamp: time.Now().UTC(),
		Success:   false,
	})
	return httputil.WriteError(c, fiber.StatusBadGateway, errMessage(lastErr))
}

type openAIModel struct {
	ID         string `json:"id"`
	Object     string `json:"object"`
	OwnedBy    string `json:"owned_by"`
	Created    int64  `json:"created"`
	Deployment string `json:"deployment"`
}

type openAIModelList struct {
	Object string        `json:"object"`
	Data   []openAIModel `json:"data"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature *float32            `json:"temperature,omitempty"`
	TopP        *float32            `json:"top_p,omitempty"`
	MaxTokens   *int32              `json:"max_tokens,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	StopRaw     json.RawMessage     `json:"stop,omitempty"`
}

type openAIChatChoice struct {
	Index        int               `json:"index"`
	Message      openAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
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

	messages := make([]models.ChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := strings.ToLower(m.Role)
		if role == "" {
			role = "user"
		}
		messages = append(messages, models.ChatMessage{
			Role:    role,
			Content: m.Content,
			Name:    m.Name,
		})
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
		return h.handleStreamChat(c, rc, alias, traceID, idempotencyKey, modelReq)
	}

	if idempotencyKey != "" {
		if data, ok := h.container.Idempotency.Get(ctx, idempotencyKey); ok {
			c.Set("Content-Type", "application/json")
			return c.Send(data)
		}
	}

	chatResult, err := h.executor.Chat(ctx, rc, alias, modelReq, traceID, idempotencyKey)
	if err != nil {
		if status, msg, ok := executor.AsAPIError(err); ok {
			return httputil.WriteError(c, status, msg)
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	setBudgetHeaders(c, chatResult.BudgetStatus)

	resp := convertChatResponse(chatResult.Response, alias)
	if idempotencyKey != "" {
		if payload, err := json.Marshal(resp); err == nil {
			h.container.Idempotency.Set(ctx, idempotencyKey, payload)
		}
	}

	return c.JSON(resp)
}

func (h *openAIHandler) handleStreamChat(
	c *fiber.Ctx,
	rc *requestctx.Context,
	alias string,
	traceID string,
	idempotencyKey string,
	req models.ChatRequest,
) error {
	ctx := c.UserContext()
	routes := h.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}

	initialBudget, err := h.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if initialBudget.Exceeded {
		setBudgetHeaders(c, initialBudget)
		_, _ = h.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
		return httputil.WriteError(c, fiber.StatusForbidden, "tenant budget exceeded")
	}
	setBudgetHeaders(c, initialBudget)

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := h.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return httputil.WriteError(c, fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	var once sync.Once
	releaseOnce := func() { once.Do(release) }

	return h.streamChat(c, alias, rc, traceID, idempotencyKey, req, routes, keyKey, keyCfg, tenantKey, tenantCfg, releaseOnce)
}

func (h *openAIHandler) streamChat(
	c *fiber.Ctx,
	alias string,
	rc *requestctx.Context,
	traceID, idempotencyKey string,
	req models.ChatRequest,
	routes []providers.Route,
	keyKey string,
	keyCfg limits.LimitConfig,
	tenantKey string,
	tenantCfg limits.LimitConfig,
	release func(),
) error {
	ctx := c.UserContext()

	var lastErr error
	var lastRoute providers.Route
	for _, route := range routes {
		if route.ChatStream == nil {
			continue
		}
		lastRoute = route
		req.Model = route.ResolveDeployment()
		chunks, cancel, err := route.ChatStream.ChatStream(ctx, req)
		if err != nil {
			h.container.Engine.ReportFailure(alias, route)
			lastErr = err
			continue
		}
		lastRoute = route

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")

		streamStart := time.Now()
		var firstTokenLatency time.Duration
		var firstTokenMeasured bool

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			defer cancel()
			defer release()

			recordStatus := fiber.StatusOK
			recordSuccess := false
			reported := false
			var streamUsage models.Usage
			usageCaptured := false

			recordUsage := func() {
				tokensUsed := int(streamUsage.TotalTokens)
				if usageCaptured && tokensUsed > 0 {
					if err := h.container.RateLimiter.TokenAllowance(ctx, keyKey, tokensUsed, keyCfg); err != nil {
						if errors.Is(err, limits.ErrLimitExceeded) {
							recordStatus = fiber.StatusTooManyRequests
						} else {
							recordStatus = fiber.StatusInternalServerError
						}
						recordSuccess = false
					}
					if err := h.container.RateLimiter.TokenAllowance(ctx, tenantKey, tokensUsed, tenantCfg); err != nil {
						if errors.Is(err, limits.ErrLimitExceeded) {
							recordStatus = fiber.StatusTooManyRequests
						} else {
							recordStatus = fiber.StatusInternalServerError
						}
						recordSuccess = false
					}
				}

				latency := time.Since(streamStart)
				if firstTokenMeasured && firstTokenLatency > 0 {
					latency = firstTokenLatency
				}
				record := usagepipeline.Record{
					Context:        rc,
					Alias:          alias,
					Provider:       route.Provider,
					Usage:          streamUsage,
					Latency:        latency,
					Status:         recordStatus,
					TraceID:        traceID,
					IdempotencyKey: idempotencyKey,
					Timestamp:      time.Now().UTC(),
					Success:        recordSuccess && recordStatus == fiber.StatusOK,
				}

				if _, err := h.container.UsageLogger.Record(ctx, record); err != nil {
					slog.Error("record stream usage", slog.String("alias", alias), slog.String("error", err.Error()))
				}
			}

			defer recordUsage()
			defer func() {
				if !reported {
					h.container.Engine.ReportFailure(alias, route)
				}
			}()

			for chunk := range chunks {
				if chunk.IsUsageOnly() {
					if chunk.Usage != nil {
						streamUsage = *chunk.Usage
						usageCaptured = true
					}
					continue
				}

				if !firstTokenMeasured {
					firstTokenLatency = time.Since(streamStart)
					firstTokenMeasured = true
				}

				payload := convertStreamChunk(chunk, alias)
				data, err := json.Marshal(payload)
				if err != nil {
					recordStatus = fiber.StatusInternalServerError
					return
				}
				if _, err = w.WriteString("data: "); err != nil {
					recordStatus = fiber.StatusInternalServerError
					return
				}
				if _, err = w.Write(data); err != nil {
					recordStatus = fiber.StatusInternalServerError
					return
				}
				if _, err = w.WriteString("\n\n"); err != nil {
					recordStatus = fiber.StatusInternalServerError
					return
				}
				if err = w.Flush(); err != nil {
					recordStatus = fiber.StatusInternalServerError
					return
				}

				if chunk.Usage != nil {
					streamUsage = *chunk.Usage
					usageCaptured = true
				}

				recordSuccess = true
			}

			if _, err := w.WriteString("data: [DONE]\n\n"); err != nil {
				recordStatus = fiber.StatusInternalServerError
				return
			}
			if err := w.Flush(); err != nil {
				recordStatus = fiber.StatusInternalServerError
				return
			}

			h.container.Engine.ReportSuccess(alias, route)
			reported = true
		})

		return nil
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	if lastRoute.Provider != "" && rc != nil {
		_, _ = h.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  lastRoute.Provider,
			Status:    fiber.StatusBadGateway,
			ErrorCode: lastErr.Error(),
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
	}
	release()
	return httputil.WriteError(c, fiber.StatusBadGateway, lastErr.Error())
}

type openAIEmbeddingRequest struct {
	Model string          `json:"model"`
	Input json.RawMessage `json:"input"`
}

type openAIModerationRequest struct {
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

type openAIImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Quality        string `json:"quality,omitempty"`
	N              int    `json:"n,omitempty"`
	User           string `json:"user,omitempty"`
	Background     string `json:"background,omitempty"`
	Style          string `json:"style,omitempty"`
}

type openAIImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type openAIImageResponse struct {
	Created int64             `json:"created"`
	Data    []openAIImageData `json:"data"`
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
	if !h.container.IsModelAllowed(rc.TenantID, alias) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	routes := h.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}

	traceID := traceIDFromContext(c)
	initialBudget, err := h.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if initialBudget.Exceeded {
		setBudgetHeaders(c, initialBudget)
		_, _ = h.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
		return httputil.WriteError(c, fiber.StatusForbidden, "tenant budget exceeded")
	}
	setBudgetHeaders(c, initialBudget)

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := h.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return httputil.WriteError(c, fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer release()

	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration
	for _, route := range routes {
		if route.Moderations == nil {
			continue
		}
		lastRoute = route
		modelReq := models.ModerationRequest{
			Model: route.ResolveDeployment(),
			Input: inputs,
		}
		start := time.Now()
		resp, err := route.Moderations.Moderate(ctx, modelReq)
		if err != nil {
			h.container.Engine.ReportFailure(alias, route)
			lastErr = err
			lastLatency = time.Since(start)
			continue
		}
		h.container.Engine.ReportSuccess(alias, route)

		tokens := int(resp.Usage.TotalTokens)
		if tokens > 0 {
			if err := h.container.RateLimiter.TokenAllowance(context.Background(), keyKey, tokens, keyCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
			}
			if err := h.container.RateLimiter.TokenAllowance(context.Background(), tenantKey, tokens, tenantCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
			}
		}

		record := usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  route.Provider,
			Usage:     resp.Usage,
			Latency:   time.Since(start),
			Status:    fiber.StatusOK,
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   true,
		}
		if status, err := h.container.UsageLogger.Record(ctx, record); err == nil {
			setBudgetHeaders(c, status)
		} else {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to persist usage")
		}

		response := convertModerationHTTPResponse(resp, alias)
		return c.JSON(response)
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	if lastRoute.Provider != "" {
		_, _ = h.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  lastRoute.Provider,
			Latency:   lastLatency,
			Status:    fiber.StatusBadGateway,
			ErrorCode: errMessage(lastErr),
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
	}

	return httputil.WriteError(c, fiber.StatusBadGateway, errMessage(lastErr))
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
	if !h.container.IsModelAllowed(rc.TenantID, req.Model) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	routes := h.container.Engine.SelectRoutes(req.Model)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}

	alias := req.Model

	traceID := traceIDFromContext(c)

	initialBudget, err := h.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if initialBudget.Exceeded {
		setBudgetHeaders(c, initialBudget)
		_, _ = h.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     req.Model,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
		return httputil.WriteError(c, fiber.StatusForbidden, "tenant budget exceeded")
	}
	setBudgetHeaders(c, initialBudget)

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := h.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return httputil.WriteError(c, fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer release()

	modelReq := models.EmbeddingsRequest{
		Input: inputs,
	}

	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration
	for _, route := range routes {
		lastRoute = route
		modelReq.Model = route.ResolveDeployment()
		start := time.Now()
		resp, err := route.Embedding.Embed(ctx, modelReq)
		if err != nil {
			h.container.Engine.ReportFailure(req.Model, route)
			lastLatency = time.Since(start)
			lastErr = err
			continue
		}
		h.container.Engine.ReportSuccess(req.Model, route)
		elapsed := time.Since(start)
		lastLatency = elapsed

		tokensUsed := int(resp.Usage.TotalTokens)
		if tokensUsed > 0 {
			if err := h.container.RateLimiter.TokenAllowance(context.Background(), keyKey, tokensUsed, keyCfg); err != nil {
				release()
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
			}
			if err := h.container.RateLimiter.TokenAllowance(context.Background(), tenantKey, tokensUsed, tenantCfg); err != nil {
				release()
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
			}
		}
		openaiResp := convertEmbeddingResponse(resp, alias)
		record := usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  route.Provider,
			Usage:     resp.Usage,
			Latency:   elapsed,
			Status:    fiber.StatusOK,
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   true,
		}
		if status, err := h.container.UsageLogger.Record(ctx, record); err == nil {
			setBudgetHeaders(c, status)
		} else {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to persist usage")
		}
		return c.JSON(openaiResp)
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	if lastRoute.Provider != "" {
		_, _ = h.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  lastRoute.Provider,
			Latency:   lastLatency,
			Status:    fiber.StatusBadGateway,
			ErrorCode: lastErr.Error(),
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
	}
	return httputil.WriteError(c, fiber.StatusBadGateway, lastErr.Error())
}

func (h *openAIHandler) imageGenerations(c *fiber.Ctx) error {
	var req openAIImageRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Model = strings.TrimSpace(req.Model)
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if req.Prompt == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "prompt is required")
	}

	n := req.N
	if n <= 0 {
		n = 1
	}
	if n > 10 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "n must be between 1 and 10")
	}

	ctx := c.UserContext()
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	baseReq := req
	return h.runImageOperation(c, imageOperationConfig{
		Alias:          req.Model,
		IdempotencyKey: idempotencyKey,
		Operation:      imageOperationGeneration,
		Builder: func(route providers.Route) (models.ImageResponse, error) {
			modelReq := models.ImageRequest{
				Model:          route.ResolveDeployment(),
				Prompt:         baseReq.Prompt,
				Size:           baseReq.Size,
				ResponseFormat: baseReq.ResponseFormat,
				Quality:        baseReq.Quality,
				N:              n,
				User:           baseReq.User,
				Background:     baseReq.Background,
				Style:          baseReq.Style,
			}
			return route.Image.Generate(ctx, modelReq)
		},
	})
}

func (h *openAIHandler) imageEdits(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "multipart form required")
	}
	model := strings.TrimSpace(c.FormValue("model"))
	prompt := strings.TrimSpace(c.FormValue("prompt"))
	if model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if prompt == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "prompt is required")
	}
	imageHeaders := gatherFormFiles(form, "image", "image[]")
	if len(imageHeaders) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "at least one image is required")
	}
	if len(imageHeaders) > maxImageInputs {
		return httputil.WriteError(c, fiber.StatusBadRequest, "a maximum of 16 images are supported")
	}
	images := make([]models.ImageInput, 0, len(imageHeaders))
	for _, fh := range imageHeaders {
		input, err := loadImageInput(fh)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "failed to read image upload")
		}
		images = append(images, input)
	}
	var maskInput *models.ImageInput
	maskHeaders := gatherFormFiles(form, "mask", "mask[]")
	if len(maskHeaders) > 0 {
		if len(maskHeaders) > 1 {
			return httputil.WriteError(c, fiber.StatusBadRequest, "only one mask file is supported")
		}
		mask, err := loadImageInput(maskHeaders[0])
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "failed to read mask upload")
		}
		maskInput = &mask
	}
	n, err := parseImageCount(c.FormValue("n"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	baseReq := models.ImageEditRequest{
		Model:          model,
		Prompt:         prompt,
		Images:         images,
		Mask:           maskInput,
		Size:           strings.TrimSpace(c.FormValue("size")),
		ResponseFormat: strings.TrimSpace(c.FormValue("response_format")),
		Quality:        strings.TrimSpace(c.FormValue("quality")),
		Background:     strings.TrimSpace(c.FormValue("background")),
		Style:          strings.TrimSpace(c.FormValue("style")),
		N:              n,
		User:           strings.TrimSpace(c.FormValue("user")),
	}
	ctx := c.UserContext()
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	return h.runImageOperation(c, imageOperationConfig{
		Alias:          model,
		IdempotencyKey: idempotencyKey,
		Operation:      imageOperationEdit,
		Builder: func(route providers.Route) (models.ImageResponse, error) {
			req := baseReq
			req.Model = route.ResolveDeployment()
			req.Images = cloneImageInputs(baseReq.Images)
			if baseReq.Mask != nil {
				maskCopy := *baseReq.Mask
				req.Mask = &maskCopy
			}
			return route.Image.Edit(ctx, req)
		},
	})
}

func (h *openAIHandler) imageVariations(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "multipart form required")
	}
	model := strings.TrimSpace(c.FormValue("model"))
	if model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	imageHeaders := gatherFormFiles(form, "image", "image[]")
	if len(imageHeaders) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "image file is required")
	}
	if len(imageHeaders) > 1 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "only one image file is supported for variations")
	}
	baseImage, err := loadImageInput(imageHeaders[0])
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "failed to read image upload")
	}
	n, err := parseImageCount(c.FormValue("n"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	baseReq := models.ImageVariationRequest{
		Model:          model,
		Image:          baseImage,
		Size:           strings.TrimSpace(c.FormValue("size")),
		ResponseFormat: strings.TrimSpace(c.FormValue("response_format")),
		Quality:        strings.TrimSpace(c.FormValue("quality")),
		Background:     strings.TrimSpace(c.FormValue("background")),
		Style:          strings.TrimSpace(c.FormValue("style")),
		N:              n,
		User:           strings.TrimSpace(c.FormValue("user")),
	}
	ctx := c.UserContext()
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	return h.runImageOperation(c, imageOperationConfig{
		Alias:          model,
		IdempotencyKey: idempotencyKey,
		Operation:      imageOperationVariation,
		Builder: func(route providers.Route) (models.ImageResponse, error) {
			req := baseReq
			req.Model = route.ResolveDeployment()
			req.Image = baseReq.Image
			return route.Image.Variation(ctx, req)
		},
	})
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

func parseImageCount(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 1, nil
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("n must be between 1 and 10")
	}
	if val < 1 || val > 10 {
		return 0, errors.New("n must be between 1 and 10")
	}
	return val, nil
}

func loadImageInput(fh *multipart.FileHeader) (models.ImageInput, error) {
	file, err := fh.Open()
	if err != nil {
		return models.ImageInput{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return models.ImageInput{}, err
	}
	if int64(len(data)) > maxImageUploadBytes {
		return models.ImageInput{}, fmt.Errorf("image uploads must be <= %d MB", maxImageUploadBytes/1024/1024)
	}
	contentType := strings.TrimSpace(fh.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" || !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return models.ImageInput{}, errors.New("image uploads must use an image/* content type")
	}
	return models.ImageInput{
		Data:        data,
		Filename:    fh.Filename,
		ContentType: contentType,
	}, nil
}

func cloneImageInputs(inputs []models.ImageInput) []models.ImageInput {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]models.ImageInput, len(inputs))
	copy(out, inputs)
	return out
}

func parseImageOverrideCost(metadata map[string]string, op imageOperationType) *int64 {
	return parseImageCost(metadata, op)
}

func parseImageCost(metadata map[string]string, op imageOperationType) *int64 {
	if metadata == nil {
		return nil
	}
	var keys []string
	switch op {
	case imageOperationEdit:
		keys = []string{"price_image_edit_cents"}
	case imageOperationVariation:
		keys = []string{"price_image_variation_cents"}
	default:
		keys = []string{"price_image_cents"}
	}
	keys = append(keys, "price_image_cents")
	for _, key := range keys {
		if price := strings.TrimSpace(metadata[key]); price != "" {
			if cents, err := strconv.ParseInt(price, 10, 64); err == nil {
				return &cents
			}
		}
	}
	return nil
}

func errMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func gatherFormFiles(form *multipart.Form, names ...string) []*multipart.FileHeader {
	if form == nil {
		return nil
	}
	var headers []*multipart.FileHeader
	for _, name := range names {
		if files := form.File[name]; len(files) > 0 {
			headers = append(headers, files...)
		}
	}
	return headers
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

func setBudgetHeaders(c *fiber.Ctx, status usagepipeline.BudgetStatus) {
	c.Set("X-Budget-Limit-Cents", strconv.FormatInt(status.LimitCents, 10))
	c.Set("X-Budget-Total-Cents", strconv.FormatInt(status.TotalCostCents, 10))
	remaining := status.LimitCents - status.TotalCostCents
	if remaining < 0 {
		remaining = 0
	}
	c.Set("X-Budget-Remaining-Cents", strconv.FormatInt(remaining, 10))
	if status.Warning {
		c.Set("X-Budget-Warning", "true")
	}
	if status.Exceeded {
		c.Set("X-Budget-Exceeded", "true")
	}
}

func convertChatResponse(resp models.ChatResponse, alias string) openAIChatResponse {
	choices := make([]openAIChatChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		choices = append(choices, openAIChatChoice{
			Index: choice.Index,
			Message: openAIChatMessage{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
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

type openAIStreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type openAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

type openAIStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openAIStreamChoice `json:"choices"`
}

func convertStreamChunk(chunk models.ChatChunk, alias string) openAIStreamChunk {
	choices := make([]openAIStreamChoice, 0, len(chunk.Choices))
	for _, choice := range chunk.Choices {
		delta := openAIStreamDelta{
			Role:    choice.Delta.Role,
			Content: choice.Delta.Content,
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
