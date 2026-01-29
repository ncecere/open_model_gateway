package executor

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/apperror"
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

// ExecutionMetrics captures provider execution details for logging.
type ExecutionMetrics struct {
	Provider      string
	ProviderModel string
	Deployment    string
	LatencyMs     int64
	RetryCount    int
	RouteCount    int
	CostMicroUSD  int64
}

// ChatResult captures the outcome of a chat execution.
type ChatResult struct {
	Response     models.ChatResponse
	BudgetStatus usagepipeline.BudgetStatus
	Metrics      ExecutionMetrics
}

// ImageOperationConfig wraps shared parameters for image executions.
type ImageOperationConfig struct {
	Alias           string
	IdempotencyKey  string
	Builder         func(context.Context, providers.Route) (models.ImageResponse, error)
	OverrideCost    func(metadata map[string]string) *int64
	ImagePixels     int64
	PricingMetadata map[string]string
}

// ImageResult captures the outcome of an image execution.
type ImageResult struct {
	Response     models.ImageResponse
	BudgetStatus usagepipeline.BudgetStatus
	Metrics      ExecutionMetrics
}

// EmbeddingsResult captures the outcome of an embedding execution.
type EmbeddingsResult struct {
	Response     models.EmbeddingsResponse
	BudgetStatus usagepipeline.BudgetStatus
	Metrics      ExecutionMetrics
}

// ModerationResult captures moderation execution output.
type ModerationResult struct {
	Response     models.ModerationResponse
	BudgetStatus usagepipeline.BudgetStatus
	Metrics      ExecutionMetrics
}

// apiError wraps an error with an HTTP status code so callers can map it
// directly to OpenAI-compatible responses.
// Deprecated: Use apperror package types directly for new code.
type apiError struct {
	status int
	msg    string
}

func (e apiError) Error() string { return e.msg }

// NewAPIError creates an error tied to an HTTP status code.
// Deprecated: Use apperror package functions (BadRequest, RateLimited, etc.) instead.
func NewAPIError(status int, msg string) error {
	// Map to apperror types for consistency
	switch status {
	case fiber.StatusBadRequest:
		return apperror.BadRequest("executor", msg)
	case fiber.StatusUnauthorized:
		return apperror.Unauthorized("executor", msg)
	case fiber.StatusForbidden:
		return apperror.Forbidden("executor", msg)
	case fiber.StatusNotFound:
		return apperror.NotFound("executor", msg)
	case fiber.StatusTooManyRequests:
		return apperror.RateLimited("executor", msg)
	case fiber.StatusPaymentRequired:
		return apperror.BudgetExceeded("executor", msg)
	case fiber.StatusServiceUnavailable:
		return apperror.ServiceUnavailable("executor", msg)
	case fiber.StatusBadGateway, fiber.StatusGatewayTimeout:
		return apperror.ServiceUnavailable("executor", msg)
	default:
		return apperror.Internal("executor", msg)
	}
}

// AsAPIError extracts the HTTP status information when available.
// Works with both legacy apiError and apperror.Error types.
func AsAPIError(err error) (int, string, bool) {
	if err == nil {
		return 0, "", false
	}
	// Check for apperror.Error first
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return apperror.StatusCode(err), apperror.GetMessage(err), true
	}
	// Fall back to legacy apiError for backward compatibility
	var apiErr apiError
	if errors.As(err, &apiErr) {
		return apiErr.status, apiErr.msg, true
	}
	return 0, "", false
}

func (e *Executor) selectRoutes(alias string) ([]providers.Route, error) {
	routes := e.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return nil, NewAPIError(fiber.StatusServiceUnavailable, "no backend available for model")
	}
	return routes, nil
}

func (e *Executor) filterByCapabilities(routes []providers.Route, reqCaps models.CapabilityRequirements) ([]providers.Route, error) {
	if !reqCaps.HasRequirements() {
		return routes, nil
	}
	eligible := make([]providers.Route, 0, len(routes))
	for _, route := range routes {
		if route.SupportsCapabilities(reqCaps) {
			eligible = append(eligible, route)
		}
	}
	if len(eligible) == 0 {
		return nil, NewAPIError(fiber.StatusBadRequest, fmt.Sprintf("model does not support %s input", reqCaps.Describe()))
	}
	return eligible, nil
}

// Chat executes a chat completion against the routed providers.
func (e *Executor) Chat(ctx context.Context, rc *requestctx.Context, alias string, req models.ChatRequest, traceID string, idempotencyKey string) (ChatResult, error) {
	ctx, span := execTracer.Start(ctx, "Executor.Chat", trace.WithAttributes(attribute.String("alias", alias)))
	defer span.End()
	routes, err := e.selectRoutes(alias)
	if err != nil {
		e.spanError(span, err)
		return ChatResult{}, err
	}

	reqCaps := req.CapabilityRequirements()
	routes, err = e.filterByCapabilities(routes, reqCaps)
	if err != nil {
		e.spanError(span, err)
		return ChatResult{}, err
	}

	result, err := executeWithRetry(ctx, e, span, rc, routes, retryLoopConfig[models.ChatResponse]{
		spanName: "Executor.Chat",
		alias:    alias,
		traceID:  traceID,
		routeFilter: func(route providers.Route) bool {
			return route.Chat != nil
		},
		call: func(attemptCtx context.Context, route providers.Route) (routeCallResult[models.ChatResponse], error) {
			req.Model = route.ResolveDeployment()
			resp, err := route.Chat.Chat(attemptCtx, req)
			if err != nil {
				return routeCallResult[models.ChatResponse]{}, err
			}
			return routeCallResult[models.ChatResponse]{response: resp, usage: resp.Usage}, nil
		},
		buildRecord: func(rc *requestctx.Context, alias string, route providers.Route, usage models.Usage, elapsed time.Duration, traceID string) usagepipeline.Record {
			return usagepipeline.Record{
				Context:        rc,
				Alias:          alias,
				Provider:       route.Provider,
				Usage:          usage,
				Latency:        elapsed,
				Status:         fiber.StatusOK,
				IdempotencyKey: idempotencyKey,
				TraceID:        traceID,
				Timestamp:      time.Now().UTC(),
				Success:        true,
			}
		},
	})
	if err != nil {
		return ChatResult{BudgetStatus: result.budgetStatus}, err
	}
	return ChatResult{
		Response:     result.response,
		BudgetStatus: result.budgetStatus,
		Metrics:      result.metrics,
	}, nil
}

// Image executes an image generation/edit/variation operation across routed providers.
func (e *Executor) Image(ctx context.Context, rc *requestctx.Context, traceID string, imgCfg ImageOperationConfig) (ImageResult, error) {
	alias := strings.TrimSpace(imgCfg.Alias)
	if alias == "" {
		return ImageResult{}, NewAPIError(fiber.StatusBadRequest, "model is required")
	}
	ctx, span := execTracer.Start(ctx, "Executor.Image", trace.WithAttributes(attribute.String("alias", alias)))
	defer span.End()
	routes, err := e.selectRoutes(alias)
	if err != nil {
		e.spanError(span, err)
		return ImageResult{}, err
	}

	result, err := executeWithRetry(ctx, e, span, rc, routes, retryLoopConfig[models.ImageResponse]{
		spanName: "Executor.Image",
		alias:    alias,
		traceID:  traceID,
		routeFilter: func(route providers.Route) bool {
			return route.Image != nil
		},
		call: func(attemptCtx context.Context, route providers.Route) (routeCallResult[models.ImageResponse], error) {
			resp, err := imgCfg.Builder(attemptCtx, route)
			if err != nil {
				if errors.Is(err, models.ErrImageOperationUnsupported) {
					return routeCallResult[models.ImageResponse]{breakRoute: true}, err
				}
				return routeCallResult[models.ImageResponse]{}, err
			}
			return routeCallResult[models.ImageResponse]{response: resp, usage: resp.Usage}, nil
		},
		buildRecord: func(rc *requestctx.Context, alias string, route providers.Route, usage models.Usage, elapsed time.Duration, traceID string) usagepipeline.Record {
			record := usagepipeline.Record{
				Context:         rc,
				Alias:           alias,
				Provider:        route.Provider,
				Usage:           usage,
				Latency:         elapsed,
				Status:          fiber.StatusOK,
				TraceID:         traceID,
				Timestamp:       time.Now().UTC(),
				Success:         true,
				IdempotencyKey:  imgCfg.IdempotencyKey,
				PricingMetadata: imgCfg.PricingMetadata,
			}
			if imgCfg.OverrideCost != nil {
				record.OverrideCostCents = imgCfg.OverrideCost(route.Metadata)
			}
			if imgCfg.ImagePixels > 0 && record.Usage.ImagePixels == 0 {
				record.Usage.ImagePixels = imgCfg.ImagePixels
			}
			return record
		},
	})
	if err != nil {
		return ImageResult{BudgetStatus: result.budgetStatus}, err
	}
	// Patch ImageCount if not set by provider.
	if result.metrics.CostMicroUSD == 0 {
		// The metrics are already populated by the loop.
	}
	return ImageResult{
		Response:     result.response,
		BudgetStatus: result.budgetStatus,
		Metrics:      result.metrics,
	}, nil
}

// Embed executes an embeddings request against routed providers.
func (e *Executor) Embed(ctx context.Context, rc *requestctx.Context, alias string, req models.EmbeddingsRequest, traceID string) (EmbeddingsResult, error) {
	ctx, span := execTracer.Start(ctx, "Executor.Embed", trace.WithAttributes(attribute.String("alias", alias)))
	defer span.End()
	routes, err := e.selectRoutes(alias)
	if err != nil {
		e.spanError(span, err)
		return EmbeddingsResult{}, err
	}

	result, err := executeWithRetry(ctx, e, span, rc, routes, retryLoopConfig[models.EmbeddingsResponse]{
		spanName: "Executor.Embed",
		alias:    alias,
		traceID:  traceID,
		routeFilter: func(route providers.Route) bool {
			return route.Embedding != nil
		},
		call: func(attemptCtx context.Context, route providers.Route) (routeCallResult[models.EmbeddingsResponse], error) {
			req.Model = route.ResolveDeployment()
			resp, err := route.Embedding.Embed(attemptCtx, req)
			if err != nil {
				return routeCallResult[models.EmbeddingsResponse]{}, err
			}
			return routeCallResult[models.EmbeddingsResponse]{response: resp, usage: resp.Usage}, nil
		},
		buildRecord: func(rc *requestctx.Context, alias string, route providers.Route, usage models.Usage, elapsed time.Duration, traceID string) usagepipeline.Record {
			return usagepipeline.Record{
				Context:   rc,
				Alias:     alias,
				Provider:  route.Provider,
				Usage:     usage,
				Latency:   elapsed,
				Status:    fiber.StatusOK,
				TraceID:   traceID,
				Timestamp: time.Now().UTC(),
				Success:   true,
			}
		},
	})
	if err != nil {
		return EmbeddingsResult{BudgetStatus: result.budgetStatus}, err
	}
	return EmbeddingsResult{
		Response:     result.response,
		BudgetStatus: result.budgetStatus,
		Metrics:      result.metrics,
	}, nil
}

// Moderate executes moderation requests across routed providers.
func (e *Executor) Moderate(ctx context.Context, rc *requestctx.Context, alias string, inputs []string, traceID string) (ModerationResult, error) {
	ctx, span := execTracer.Start(ctx, "Executor.Moderate", trace.WithAttributes(attribute.String("alias", alias)))
	defer span.End()
	routes, err := e.selectRoutes(alias)
	if err != nil {
		e.spanError(span, err)
		return ModerationResult{}, err
	}

	result, err := executeWithRetry(ctx, e, span, rc, routes, retryLoopConfig[models.ModerationResponse]{
		spanName: "Executor.Moderate",
		alias:    alias,
		traceID:  traceID,
		routeFilter: func(route providers.Route) bool {
			return route.Moderations != nil
		},
		call: func(attemptCtx context.Context, route providers.Route) (routeCallResult[models.ModerationResponse], error) {
			req := models.ModerationRequest{
				Model: route.ResolveDeployment(),
				Input: inputs,
			}
			resp, err := route.Moderations.Moderate(attemptCtx, req)
			if err != nil {
				return routeCallResult[models.ModerationResponse]{}, err
			}
			return routeCallResult[models.ModerationResponse]{response: resp, usage: resp.Usage}, nil
		},
		buildRecord: func(rc *requestctx.Context, alias string, route providers.Route, usage models.Usage, elapsed time.Duration, traceID string) usagepipeline.Record {
			return usagepipeline.Record{
				Context:   rc,
				Alias:     alias,
				Provider:  route.Provider,
				Usage:     usage,
				Latency:   elapsed,
				Status:    fiber.StatusOK,
				TraceID:   traceID,
				Timestamp: time.Now().UTC(),
				Success:   true,
			}
		},
	})
	if err != nil {
		return ModerationResult{BudgetStatus: result.budgetStatus}, err
	}
	return ModerationResult{
		Response:     result.response,
		BudgetStatus: result.budgetStatus,
		Metrics:      result.metrics,
	}, nil
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
	// Check apperror types
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return fmt.Sprintf("http_%d", apperror.StatusCode(err))
	}
	// Legacy apiError support
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
	// Check apperror types - retry on rate limit or server errors
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		status := apperror.StatusCode(err)
		if status == fiber.StatusTooManyRequests || status >= 500 {
			return true
		}
		return false
	}
	// Legacy apiError support
	var apiErr apiError
	if errors.As(err, &apiErr) {
		if apiErr.status == fiber.StatusTooManyRequests || apiErr.status >= 500 {
			return true
		}
		return false
	}
	return true
}

// waitForRetry sleeps for the given delay plus random jitter (up to 25% of delay).
func waitForRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	// Add jitter: 0-25% of the base delay to avoid thundering herd
	jitter := time.Duration(rand.Int63n(int64(delay/4) + 1))
	total := delay + jitter
	timer := time.NewTimer(total)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retryDelay computes the delay for the next retry attempt, honoring
// Retry-After from the error if present, otherwise using exponential backoff.
func retryDelay(err error, baseDelay time.Duration, multiplier float64, attempt int) time.Duration {
	if ra := apperror.GetRetryAfter(err); ra > 0 {
		return ra
	}
	delay := baseDelay
	for i := 1; i < attempt; i++ {
		if multiplier > 0 {
			delay = time.Duration(float64(delay) * multiplier)
		}
	}
	return delay
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
