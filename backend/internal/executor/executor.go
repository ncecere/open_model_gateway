package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
	providermetrics "github.com/ncecere/open_model_gateway/backend/internal/telemetry/provider"
)

var execTracer = otel.Tracer("open-model-gateway/executor")

// Executor encapsulates provider execution logic so both HTTP handlers and
// background workers can invoke the same code path.
type Executor struct {
	container *app.Container
}

func New(container *app.Container) *Executor {
	return &Executor{container: container}
}

// ChatResult captures the outcome of a chat execution.
type ChatResult struct {
	Response     models.ChatResponse
	BudgetStatus usagepipeline.BudgetStatus
}

// ImageOperationConfig wraps shared parameters for image executions.
type ImageOperationConfig struct {
	Alias          string
	IdempotencyKey string
	Builder        func(context.Context, providers.Route) (models.ImageResponse, error)
	OverrideCost   func(metadata map[string]string) *int64
}

// ImageResult captures the outcome of an image execution.
type ImageResult struct {
	Response     models.ImageResponse
	BudgetStatus usagepipeline.BudgetStatus
}

// EmbeddingsResult captures the outcome of an embedding execution.
type EmbeddingsResult struct {
	Response     models.EmbeddingsResponse
	BudgetStatus usagepipeline.BudgetStatus
}

// ModerationResult captures moderation execution output.
type ModerationResult struct {
	Response     models.ModerationResponse
	BudgetStatus usagepipeline.BudgetStatus
}

// apiError wraps an error with an HTTP status code so callers can map it
// directly to OpenAI-compatible responses.
type apiError struct {
	status int
	msg    string
}

func (e apiError) Error() string { return e.msg }

// NewAPIError creates an error tied to an HTTP status code.
func NewAPIError(status int, msg string) error {
	return apiError{status: status, msg: msg}
}

// AsAPIError extracts the HTTP status information when available.
func AsAPIError(err error) (int, string, bool) {
	var apiErr apiError
	if errors.As(err, &apiErr) {
		return apiErr.status, apiErr.msg, true
	}
	return 0, "", false
}

// Chat executes a chat completion against the routed providers.
func (e *Executor) Chat(ctx context.Context, rc *requestctx.Context, alias string, req models.ChatRequest, traceID string, idempotencyKey string) (ChatResult, error) {
	ctx, span := execTracer.Start(ctx, "Executor.Chat", trace.WithAttributes(attribute.String("alias", alias)))
	defer span.End()
	routes := e.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		err := NewAPIError(fiber.StatusServiceUnavailable, "no backend available for model")
		e.spanError(span, err)
		return ChatResult{}, err
	}

	reqCaps := req.CapabilityRequirements()
	if reqCaps.HasRequirements() {
		eligible := make([]providers.Route, 0, len(routes))
		for _, route := range routes {
			if route.SupportsCapabilities(reqCaps) {
				eligible = append(eligible, route)
			}
		}
		if len(eligible) == 0 {
			err := NewAPIError(fiber.StatusBadRequest, fmt.Sprintf("model does not support %s input", reqCaps.Describe()))
			e.spanError(span, err)
			return ChatResult{}, err
		}
		routes = eligible
	}

	budgetStatus, err := e.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		e.spanError(span, err)
		return ChatResult{}, err
	}
	if budgetStatus.Exceeded {
		_, _ = e.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
		return ChatResult{BudgetStatus: budgetStatus}, NewAPIError(fiber.StatusForbidden, "tenant budget exceeded")
	}

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := e.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			err = NewAPIError(fiber.StatusTooManyRequests, "rate limit exceeded")
			e.spanError(span, err)
			return ChatResult{}, err
		}
		e.spanError(span, err)
		return ChatResult{}, err
	}
	defer func() {
		if release != nil {
			release()
		}
	}()

	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration

	for _, route := range routes {
		if route.Chat == nil {
			continue
		}
		lastRoute = route
		req.Model = route.ResolveDeployment()
		retryCfg := normalizedRetry(route)
		delay := retryCfg.InitialBackoff

		for attempt := 1; attempt <= retryCfg.MaxAttempts; attempt++ {
			attemptCtx, attemptSpan := execTracer.Start(ctx, "Executor.ChatAttempt", trace.WithAttributes(
				attribute.String("provider", route.Provider),
				attribute.String("model", route.Model),
				attribute.Int("attempt", attempt),
			))
			start := time.Now()
			resp, err := route.Chat.Chat(attemptCtx, req)
			elapsed := time.Since(start)
			if err != nil {
				e.container.Engine.ReportFailure(alias, route)
				lastLatency = elapsed
				lastErr = err
				retryable := shouldRetryProvider(err)
				attemptSpan.RecordError(err)
				attemptSpan.SetStatus(codes.Error, err.Error())
				attemptSpan.End()

				if retryable && attempt < retryCfg.MaxAttempts {
					e.recordRetry(rc, alias, route.Provider, retryReason(err))
					if delay > 0 {
						if err := waitForRetry(ctx, delay); err != nil {
							return ChatResult{}, err
						}
						if retryCfg.BackoffMultiplier > 0 {
							delay = time.Duration(float64(delay) * retryCfg.BackoffMultiplier)
						}
					}
					continue
				}
				break
			}
			e.container.Engine.ReportSuccess(alias, route)
			lastLatency = elapsed
			attemptSpan.SetAttributes(attribute.Float64("duration_ms", float64(elapsed.Milliseconds())))
			attemptSpan.SetStatus(codes.Ok, "")
			attemptSpan.End()

			if tokens := int(resp.Usage.TotalTokens); tokens > 0 {
				if err := e.consumeTokens(ctx, keyKey, tenantKey, tokens, keyCfg, tenantCfg); err != nil {
					return ChatResult{}, err
				}
			}

			record := usagepipeline.Record{
				Context:        rc,
				Alias:          alias,
				Provider:       route.Provider,
				Usage:          resp.Usage,
				Latency:        elapsed,
				Status:         fiber.StatusOK,
				IdempotencyKey: idempotencyKey,
				TraceID:        traceID,
				Timestamp:      time.Now().UTC(),
				Success:        true,
			}
			budgetStatus, err := e.container.UsageLogger.Record(ctx, record)
			if err != nil {
				return ChatResult{}, err
			}

			span.SetStatus(codes.Ok, "")
			e.recordProviderSample(ctx, rc, route, alias, elapsed, providermetrics.ResultSuccess, nil, resp.Usage)
			return ChatResult{
				Response:     resp,
				BudgetStatus: budgetStatus,
			}, nil
		}
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	e.spanError(span, lastErr)
	if lastRoute.Provider != "" {
		_, _ = e.container.UsageLogger.Record(ctx, usagepipeline.Record{
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
	e.recordProviderSample(ctx, rc, lastRoute, alias, lastLatency, providermetrics.ResultError, lastErr, models.Usage{})

	return ChatResult{}, NewAPIError(fiber.StatusBadGateway, lastErr.Error())
}

// Image executes an image generation/edit/variation operation across routed providers.
func (e *Executor) Image(ctx context.Context, rc *requestctx.Context, traceID string, cfg ImageOperationConfig) (ImageResult, error) {
	alias := strings.TrimSpace(cfg.Alias)
	if alias == "" {
		return ImageResult{}, NewAPIError(fiber.StatusBadRequest, "model is required")
	}
	ctx, span := execTracer.Start(ctx, "Executor.Image", trace.WithAttributes(attribute.String("alias", alias)))
	defer span.End()
	routes := e.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		err := NewAPIError(fiber.StatusServiceUnavailable, "no backend available for model")
		e.spanError(span, err)
		return ImageResult{}, err
	}

	budgetStatus, err := e.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		e.spanError(span, err)
		return ImageResult{}, err
	}
	if budgetStatus.Exceeded {
		_, _ = e.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
		return ImageResult{BudgetStatus: budgetStatus}, NewAPIError(fiber.StatusForbidden, "tenant budget exceeded")
	}

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := e.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			err = NewAPIError(fiber.StatusTooManyRequests, "rate limit exceeded")
			e.spanError(span, err)
			return ImageResult{}, err
		}
		e.spanError(span, err)
		return ImageResult{}, err
	}
	defer release()

	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration

	for _, route := range routes {
		if route.Image == nil {
			continue
		}
		lastRoute = route
		retryCfg := normalizedRetry(route)
		delay := retryCfg.InitialBackoff

		for attempt := 1; attempt <= retryCfg.MaxAttempts; attempt++ {
			attemptCtx, attemptSpan := execTracer.Start(ctx, "Executor.ImageAttempt", trace.WithAttributes(
				attribute.String("provider", route.Provider),
				attribute.String("model", route.Model),
				attribute.Int("attempt", attempt),
			))
			start := time.Now()
			resp, err := cfg.Builder(attemptCtx, route)
			elapsed := time.Since(start)
			if err != nil {
				if errors.Is(err, models.ErrImageOperationUnsupported) {
					attemptSpan.End()
					break
				}
				e.container.Engine.ReportFailure(alias, route)
				lastLatency = elapsed
				lastErr = err
				retryable := shouldRetryProvider(err)
				attemptSpan.RecordError(err)
				attemptSpan.SetStatus(codes.Error, err.Error())
				attemptSpan.End()

				if retryable && attempt < retryCfg.MaxAttempts {
					e.recordRetry(rc, alias, route.Provider, retryReason(err))
					if delay > 0 {
						if err := waitForRetry(ctx, delay); err != nil {
							return ImageResult{}, err
						}
						if retryCfg.BackoffMultiplier > 0 {
							delay = time.Duration(float64(delay) * retryCfg.BackoffMultiplier)
						}
					}
					continue
				}
				break
			}
			e.container.Engine.ReportSuccess(alias, route)
			lastLatency = elapsed
			attemptSpan.SetAttributes(attribute.Float64("duration_ms", float64(elapsed.Milliseconds())))
			attemptSpan.SetStatus(codes.Ok, "")
			attemptSpan.End()

			if tokens := int(resp.Usage.TotalTokens); tokens > 0 {
				if err := e.consumeTokens(ctx, keyKey, tenantKey, tokens, keyCfg, tenantCfg); err != nil {
					return ImageResult{}, err
				}
			}

			record := usagepipeline.Record{
				Context:        rc,
				Alias:          alias,
				Provider:       route.Provider,
				Usage:          resp.Usage,
				Latency:        elapsed,
				Status:         fiber.StatusOK,
				TraceID:        traceID,
				Timestamp:      time.Now().UTC(),
				Success:        true,
				IdempotencyKey: cfg.IdempotencyKey,
			}
			if cfg.OverrideCost != nil {
				record.OverrideCostCents = cfg.OverrideCost(route.Metadata)
			}
			budgetStatus, err := e.container.UsageLogger.Record(ctx, record)
			if err != nil {
				return ImageResult{}, err
			}

			e.recordProviderSample(ctx, rc, route, alias, elapsed, providermetrics.ResultSuccess, nil, resp.Usage)
			span.SetStatus(codes.Ok, "")
			return ImageResult{
				Response:     resp,
				BudgetStatus: budgetStatus,
			}, nil
		}
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	e.spanError(span, lastErr)
	if lastRoute.Provider != "" {
		_, _ = e.container.UsageLogger.Record(ctx, usagepipeline.Record{
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
	e.recordProviderSample(ctx, rc, lastRoute, alias, lastLatency, providermetrics.ResultError, lastErr, models.Usage{})

	return ImageResult{}, NewAPIError(fiber.StatusBadGateway, lastErr.Error())
}

// Embed executes an embeddings request against routed providers.
func (e *Executor) Embed(ctx context.Context, rc *requestctx.Context, alias string, req models.EmbeddingsRequest, traceID string) (EmbeddingsResult, error) {
	ctx, span := execTracer.Start(ctx, "Executor.Embed", trace.WithAttributes(attribute.String("alias", alias)))
	defer span.End()
	routes := e.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		err := NewAPIError(fiber.StatusServiceUnavailable, "no backend available for model")
		e.spanError(span, err)
		return EmbeddingsResult{}, err
	}

	budgetStatus, err := e.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		e.spanError(span, err)
		return EmbeddingsResult{}, err
	}
	if budgetStatus.Exceeded {
		_, _ = e.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
		return EmbeddingsResult{BudgetStatus: budgetStatus}, NewAPIError(fiber.StatusForbidden, "tenant budget exceeded")
	}

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := e.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			err = NewAPIError(fiber.StatusTooManyRequests, "rate limit exceeded")
			e.spanError(span, err)
			return EmbeddingsResult{}, err
		}
		e.spanError(span, err)
		return EmbeddingsResult{}, err
	}
	defer release()

	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration

	for _, route := range routes {
		if route.Embedding == nil {
			continue
		}
		lastRoute = route
		req.Model = route.ResolveDeployment()
		retryCfg := normalizedRetry(route)
		delay := retryCfg.InitialBackoff

		for attempt := 1; attempt <= retryCfg.MaxAttempts; attempt++ {
			attemptCtx, attemptSpan := execTracer.Start(ctx, "Executor.EmbedAttempt", trace.WithAttributes(
				attribute.String("provider", route.Provider),
				attribute.String("model", route.Model),
				attribute.Int("attempt", attempt),
			))
			start := time.Now()
			resp, err := route.Embedding.Embed(attemptCtx, req)
			elapsed := time.Since(start)
			if err != nil {
				e.container.Engine.ReportFailure(alias, route)
				lastLatency = elapsed
				lastErr = err
				retryable := shouldRetryProvider(err)
				attemptSpan.RecordError(err)
				attemptSpan.SetStatus(codes.Error, err.Error())
				attemptSpan.End()

				if retryable && attempt < retryCfg.MaxAttempts {
					e.recordRetry(rc, alias, route.Provider, retryReason(err))
					if delay > 0 {
						if err := waitForRetry(ctx, delay); err != nil {
							return EmbeddingsResult{}, err
						}
						if retryCfg.BackoffMultiplier > 0 {
							delay = time.Duration(float64(delay) * retryCfg.BackoffMultiplier)
						}
					}
					continue
				}
				break
			}
			e.container.Engine.ReportSuccess(alias, route)
			lastLatency = elapsed
			attemptSpan.SetAttributes(attribute.Float64("duration_ms", float64(elapsed.Milliseconds())))
			attemptSpan.SetStatus(codes.Ok, "")
			attemptSpan.End()

			if tokens := int(resp.Usage.TotalTokens); tokens > 0 {
				if err := e.consumeTokens(ctx, keyKey, tenantKey, tokens, keyCfg, tenantCfg); err != nil {
					return EmbeddingsResult{}, err
				}
			}

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
			budgetStatus, err := e.container.UsageLogger.Record(ctx, record)
			if err != nil {
				return EmbeddingsResult{}, err
			}

			e.recordProviderSample(ctx, rc, route, alias, elapsed, providermetrics.ResultSuccess, nil, resp.Usage)
			span.SetStatus(codes.Ok, "")
			return EmbeddingsResult{
				Response:     resp,
				BudgetStatus: budgetStatus,
			}, nil
		}
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	e.spanError(span, lastErr)
	if lastRoute.Provider != "" {
		_, _ = e.container.UsageLogger.Record(ctx, usagepipeline.Record{
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
	e.recordProviderSample(ctx, rc, lastRoute, alias, lastLatency, providermetrics.ResultError, lastErr, models.Usage{})

	return EmbeddingsResult{}, NewAPIError(fiber.StatusBadGateway, lastErr.Error())
}

// Moderate executes moderation requests across routed providers.
func (e *Executor) Moderate(ctx context.Context, rc *requestctx.Context, alias string, inputs []string, traceID string) (ModerationResult, error) {
	ctx, span := execTracer.Start(ctx, "Executor.Moderate", trace.WithAttributes(attribute.String("alias", alias)))
	defer span.End()
	routes := e.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		err := NewAPIError(fiber.StatusServiceUnavailable, "no backend available for model")
		e.spanError(span, err)
		return ModerationResult{}, err
	}

	budgetStatus, err := e.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		e.spanError(span, err)
		return ModerationResult{}, err
	}
	if budgetStatus.Exceeded {
		_, _ = e.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
		return ModerationResult{BudgetStatus: budgetStatus}, NewAPIError(fiber.StatusForbidden, "tenant budget exceeded")
	}

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := e.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			err = NewAPIError(fiber.StatusTooManyRequests, "rate limit exceeded")
			e.spanError(span, err)
			return ModerationResult{}, err
		}
		e.spanError(span, err)
		return ModerationResult{}, err
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
		req := models.ModerationRequest{
			Model: route.ResolveDeployment(),
			Input: inputs,
		}
		retryCfg := normalizedRetry(route)
		delay := retryCfg.InitialBackoff

		for attempt := 1; attempt <= retryCfg.MaxAttempts; attempt++ {
			attemptCtx, attemptSpan := execTracer.Start(ctx, "Executor.ModerateAttempt", trace.WithAttributes(
				attribute.String("provider", route.Provider),
				attribute.String("model", route.Model),
				attribute.Int("attempt", attempt),
			))
			start := time.Now()
			resp, err := route.Moderations.Moderate(attemptCtx, req)
			elapsed := time.Since(start)
			if err != nil {
				e.container.Engine.ReportFailure(alias, route)
				lastLatency = elapsed
				lastErr = err
				retryable := shouldRetryProvider(err)
				attemptSpan.RecordError(err)
				attemptSpan.SetStatus(codes.Error, err.Error())
				attemptSpan.End()

				if retryable && attempt < retryCfg.MaxAttempts {
					e.recordRetry(rc, alias, route.Provider, retryReason(err))
					if delay > 0 {
						if err := waitForRetry(ctx, delay); err != nil {
							return ModerationResult{}, err
						}
						if retryCfg.BackoffMultiplier > 0 {
							delay = time.Duration(float64(delay) * retryCfg.BackoffMultiplier)
						}
					}
					continue
				}
				break
			}
			e.container.Engine.ReportSuccess(alias, route)
			lastLatency = elapsed
			attemptSpan.SetAttributes(attribute.Float64("duration_ms", float64(elapsed.Milliseconds())))
			attemptSpan.SetStatus(codes.Ok, "")
			attemptSpan.End()

			if tokens := int(resp.Usage.TotalTokens); tokens > 0 {
				if err := e.consumeTokens(ctx, keyKey, tenantKey, tokens, keyCfg, tenantCfg); err != nil {
					return ModerationResult{}, err
				}
			}

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
			budgetStatus, err := e.container.UsageLogger.Record(ctx, record)
			if err != nil {
				return ModerationResult{}, err
			}

			e.recordProviderSample(ctx, rc, route, alias, elapsed, providermetrics.ResultSuccess, nil, resp.Usage)
			span.SetStatus(codes.Ok, "")
			return ModerationResult{
				Response:     resp,
				BudgetStatus: budgetStatus,
			}, nil
		}
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	e.spanError(span, lastErr)
	if lastRoute.Provider != "" {
		_, _ = e.container.UsageLogger.Record(ctx, usagepipeline.Record{
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
	e.recordProviderSample(ctx, rc, lastRoute, alias, lastLatency, providermetrics.ResultError, lastErr, models.Usage{})

	return ModerationResult{}, NewAPIError(fiber.StatusBadGateway, lastErr.Error())
}

func (e *Executor) consumeTokens(ctx context.Context, keyKey, tenantKey string, tokens int, keyCfg, tenantCfg limits.LimitConfig) error {
	if err := e.container.RateLimiter.TokenAllowance(ctx, keyKey, tokens, keyCfg); err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return NewAPIError(fiber.StatusTooManyRequests, "token limit exceeded")
		}
		return err
	}
	if err := e.container.RateLimiter.TokenAllowance(ctx, tenantKey, tokens, tenantCfg); err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return NewAPIError(fiber.StatusTooManyRequests, "token limit exceeded")
		}
		return err
	}
	return nil
}

func (e *Executor) spanError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func (e *Executor) recordRetry(rc *requestctx.Context, alias, provider, reason string) {
	if e == nil || e.container == nil || e.container.Observability == nil {
		return
	}
	tenant := ""
	if rc != nil {
		tenant = rc.TenantID.String()
	}
	if reason == "" {
		reason = "provider_error"
	}
	e.container.Observability.RecordRetry(tenant, alias, provider, reason)
}

func (e *Executor) recordProviderSample(ctx context.Context, rc *requestctx.Context, route providers.Route, alias string, latency time.Duration, result providermetrics.Result, err error, usage models.Usage) {
	if e == nil || e.container == nil {
		return
	}
	if route.Provider == "" {
		return
	}
	sample := providermetrics.Sample{
		Provider:     route.Provider,
		ModelAlias:   alias,
		Route:        route.ResolveDeployment(),
		Result:       result,
		ErrorClass:   providermetrics.ClassifyError(err),
		Latency:      latency,
		InputTokens:  int64(usage.PromptTokens),
		OutputTokens: int64(usage.CompletionTokens),
		Timestamp:    time.Now().UTC(),
	}
	e.container.RecordProviderTelemetry(ctx, sample)
}

func retryReason(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, models.ErrImageOperationUnsupported) {
		return "unsupported"
	}
	if errors.Is(err, limits.ErrLimitExceeded) {
		return "limit_exceeded"
	}
	var apiErr apiError
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("http_%d", apiErr.status)
	}
	return "provider_error"
}

func shouldRetryProvider(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, models.ErrImageOperationUnsupported) {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var apiErr apiError
	if errors.As(err, &apiErr) {
		if apiErr.status == fiber.StatusTooManyRequests || apiErr.status >= 500 {
			return true
		}
		return false
	}
	return true
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func normalizedRetry(route providers.Route) providers.RetryConfig {
	cfg := route.Retry
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.InitialBackoff < 0 {
		cfg.InitialBackoff = 0
	}
	if cfg.BackoffMultiplier <= 0 {
		cfg.BackoffMultiplier = 1
	}
	return cfg
}
