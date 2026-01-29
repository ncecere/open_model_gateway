package executor

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
	providermetrics "github.com/ncecere/open_model_gateway/backend/internal/telemetry/provider"
)

// routeCallResult captures the outcome of a single provider call attempt.
type routeCallResult[T any] struct {
	response T
	usage    models.Usage
	// breakRoute signals the retry loop should skip to the next route
	// (e.g., ErrImageOperationUnsupported).
	breakRoute bool
}

// routeCallFn is the provider-specific call made on each attempt.
type routeCallFn[T any] func(ctx context.Context, route providers.Route) (routeCallResult[T], error)

// successRecordFn builds the usage record for a successful execution.
type successRecordFn func(rc *requestctx.Context, alias string, route providers.Route, usage models.Usage, elapsed time.Duration, traceID string) usagepipeline.Record

// routeFilter decides whether a route is eligible (i.e., has the right capability).
type routeFilter func(route providers.Route) bool

// retryLoopConfig configures the generic retry loop.
type retryLoopConfig[T any] struct {
	spanName       string
	alias          string
	traceID        string
	call           routeCallFn[T]
	routeFilter    routeFilter
	buildRecord    successRecordFn
	idempotencyKey string
}

// retryLoopResult wraps the outcome of the retry loop.
type retryLoopResult[T any] struct {
	response     T
	budgetStatus usagepipeline.BudgetStatus
	metrics      ExecutionMetrics
}

// executeWithRetry handles the shared budget-check → rate-limit → route/retry loop
// pattern used by Chat, Image, Embed, and Moderate.
func executeWithRetry[T any](
	ctx context.Context,
	e *Executor,
	span trace.Span,
	rc *requestctx.Context,
	routes []providers.Route,
	cfg retryLoopConfig[T],
) (retryLoopResult[T], error) {
	var zero T
	alias := cfg.alias
	traceID := cfg.traceID

	// --- Budget check ---
	budgetStatus, err := e.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		e.spanError(span, err)
		return retryLoopResult[T]{}, err
	}
	if budgetStatus.Exceeded {
		if _, recErr := e.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		}); recErr != nil {
			e.container.Log().WarnContext(ctx, "failed to record budget-exceeded denial", "error", recErr)
		}
		return retryLoopResult[T]{response: zero, budgetStatus: budgetStatus},
			NewAPIError(fiber.StatusForbidden, "tenant budget exceeded")
	}

	// --- Rate limit acquisition ---
	keyKey, keyCfg, tenantKey, tenantCfg, release, err := e.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			err = NewAPIError(fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		e.spanError(span, err)
		return retryLoopResult[T]{}, err
	}
	defer func() {
		if release != nil {
			release()
		}
	}()

	// --- Route / retry loop ---
	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration

	for _, route := range routes {
		if cfg.routeFilter != nil && !cfg.routeFilter(route) {
			continue
		}
		lastRoute = route
		retryCfg := normalizedRetry(route)

		for attempt := 1; attempt <= retryCfg.MaxAttempts; attempt++ {
			attemptCtx, attemptSpan := execTracer.Start(ctx, cfg.spanName+"Attempt", trace.WithAttributes(
				attribute.String("provider", route.Provider),
				attribute.String("model", route.Model),
				attribute.Int("attempt", attempt),
			))
			start := time.Now()
			result, callErr := cfg.call(attemptCtx, route)
			elapsed := time.Since(start)

			if callErr != nil {
				// Special: break to next route without retrying (e.g., unsupported op).
				if result.breakRoute {
					attemptSpan.End()
					break
				}
				e.container.Engine.ReportFailure(alias, route)
				lastLatency = elapsed
				lastErr = callErr
				retryable := shouldRetryProvider(callErr)
				attemptSpan.RecordError(callErr)
				attemptSpan.SetStatus(codes.Error, callErr.Error())
				attemptSpan.End()

				if retryable && attempt < retryCfg.MaxAttempts {
					e.recordRetry(rc, alias, route.Provider, retryReason(callErr))
					delay := retryDelay(callErr, retryCfg.InitialBackoff, retryCfg.BackoffMultiplier, attempt)
					if delay > 0 {
						if wErr := waitForRetry(ctx, delay); wErr != nil {
							return retryLoopResult[T]{}, wErr
						}
					}
					continue
				}
				break
			}

			// --- Success ---
			e.container.Engine.ReportSuccess(alias, route)
			lastLatency = elapsed
			attemptSpan.SetAttributes(attribute.Float64("duration_ms", float64(elapsed.Milliseconds())))
			attemptSpan.SetStatus(codes.Ok, "")
			attemptSpan.End()

			if tokens := int(result.usage.TotalTokens); tokens > 0 {
				if tErr := e.consumeTokens(ctx, keyKey, tenantKey, tokens, keyCfg, tenantCfg); tErr != nil {
					return retryLoopResult[T]{}, tErr
				}
			}

			record := cfg.buildRecord(rc, alias, route, result.usage, elapsed, traceID)
			budgetStatus, recErr := e.container.UsageLogger.Record(ctx, record)
			if recErr != nil {
				return retryLoopResult[T]{}, recErr
			}

			e.recordProviderSample(ctx, rc, route, alias, elapsed, providermetrics.ResultSuccess, nil, result.usage)
			span.SetStatus(codes.Ok, "")
			return retryLoopResult[T]{
				response:     result.response,
				budgetStatus: budgetStatus,
				metrics: ExecutionMetrics{
					Provider:      route.Provider,
					ProviderModel: route.Model,
					Deployment:    route.ResolveDeployment(),
					LatencyMs:     elapsed.Milliseconds(),
					RetryCount:    attempt - 1,
					RouteCount:    len(routes),
					CostMicroUSD:  budgetStatus.LastRecordCostMicros,
				},
			}, nil
		}
	}

	// --- All routes exhausted ---
	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	e.spanError(span, lastErr)
	if lastRoute.Provider != "" {
		if _, recErr := e.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  lastRoute.Provider,
			Latency:   lastLatency,
			Status:    fiber.StatusBadGateway,
			ErrorCode: lastErr.Error(),
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		}); recErr != nil {
			e.container.Log().WarnContext(ctx, "failed to record failure denial", "error", recErr)
		}
	}
	e.recordProviderSample(ctx, rc, lastRoute, alias, lastLatency, providermetrics.ResultError, lastErr, models.Usage{})

	return retryLoopResult[T]{}, NewAPIError(fiber.StatusBadGateway, lastErr.Error())
}
