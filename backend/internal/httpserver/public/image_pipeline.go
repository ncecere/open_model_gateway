package public

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/pipeline"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type imagePipeline struct {
	*pipeline.Base
}

func newImagePipeline(container *app.Container, exec *executor.Executor) *imagePipeline {
	return &imagePipeline{Base: pipeline.NewBase(container, exec)}
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
	ctx := c.UserContext()

	// Validate alias and tenant access
	alias, err := p.ValidateAlias(c, rc.TenantID, cfg.Alias)
	if err != nil {
		return err
	}

	operation := cfg.Operation
	if operation == "" {
		operation = imageOperationGeneration
	}

	idempotencyKey := strings.TrimSpace(cfg.IdempotencyKey)

	// Check idempotency cache
	if found, err := p.CheckIdempotency(c, ctx, idempotencyKey); found || err != nil {
		return err
	}

	traceID := traceIDFromContext(c)
	result, err := p.Executor.Image(ctx, rc, traceID, executor.ImageOperationConfig{
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
		return p.HandleExecutorError(c, err)
	}

	return p.SendJSONWithIdempotency(c, ctx, result.BudgetStatus, convertImageResponse(result.Response), idempotencyKey)
}
