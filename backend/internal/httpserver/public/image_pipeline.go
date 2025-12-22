package public

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type imagePipeline struct {
	container *app.Container
	executor  *executor.Executor
}

func newImagePipeline(container *app.Container, exec *executor.Executor) *imagePipeline {
	return &imagePipeline{container: container, executor: exec}
}

type imageOperationType string

const (
	imageOperationGeneration imageOperationType = "generation"
	imageOperationEdit       imageOperationType = "edit"
	imageOperationVariation  imageOperationType = "variation"
)

type imageOperationConfig struct {
	Alias           string
	IdempotencyKey  string
	Operation       imageOperationType
	Builder         func(context.Context, providers.Route) (models.ImageResponse, error)
	ImagePixels     int64
	PricingMetadata map[string]string
}

func (p *imagePipeline) Execute(c *fiber.Ctx, rc *requestctx.Context, cfg imageOperationConfig) error {
	alias := strings.TrimSpace(cfg.Alias)
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if !p.container.IsModelAllowed(rc.TenantID, alias) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	operation := cfg.Operation
	if operation == "" {
		operation = imageOperationGeneration
	}

	traceID := traceIDFromContext(c)
	idempotencyKey := strings.TrimSpace(cfg.IdempotencyKey)
	if idempotencyKey != "" {
		if data, ok := p.container.Idempotency.Get(c.UserContext(), idempotencyKey); ok {
			c.Set("Content-Type", "application/json")
			return c.Send(data)
		}
	}
	result, err := p.executor.Image(c.UserContext(), rc, traceID, executor.ImageOperationConfig{
		Alias:           alias,
		IdempotencyKey:  idempotencyKey,
		Builder:         cfg.Builder,
		ImagePixels:     cfg.ImagePixels,
		PricingMetadata: cfg.PricingMetadata,
		OverrideCost: func(metadata map[string]string) *int64 {
			return parseImageOverrideCost(metadata, operation)
		},
	})
	if err != nil {
		if status, msg, ok := executor.AsAPIError(err); ok {
			return httputil.WriteError(c, status, msg)
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	httputil.ApplyBudgetHeaders(c, result.BudgetStatus)

	payload, err := json.Marshal(convertImageResponse(result.Response))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to encode response")
	}
	if idempotencyKey != "" {
		p.container.Idempotency.Set(c.UserContext(), idempotencyKey, payload)
	}

	c.Set("Content-Type", "application/json")
	return c.Send(payload)
}
