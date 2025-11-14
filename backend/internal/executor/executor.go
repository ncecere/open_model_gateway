package executor

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

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
	routes := e.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return ChatResult{}, NewAPIError(fiber.StatusServiceUnavailable, "no backend available for model")
	}

	budgetStatus, err := e.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
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
			return ChatResult{}, NewAPIError(fiber.StatusTooManyRequests, "rate limit exceeded")
		}
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
		start := time.Now()
		resp, err := route.Chat.Chat(ctx, req)
		if err != nil {
			e.container.Engine.ReportFailure(alias, route)
			lastLatency = time.Since(start)
			lastErr = err
			continue
		}
		e.container.Engine.ReportSuccess(alias, route)
		elapsed := time.Since(start)
		lastLatency = elapsed

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

		return ChatResult{
			Response:     resp,
			BudgetStatus: budgetStatus,
		}, nil
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
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

	return ChatResult{}, NewAPIError(fiber.StatusBadGateway, lastErr.Error())
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
