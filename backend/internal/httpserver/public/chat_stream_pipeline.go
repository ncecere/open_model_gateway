package public

import (
	"bufio"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

type chatStreamPipeline struct {
	container *app.Container
}

func newChatStreamPipeline(container *app.Container) *chatStreamPipeline {
	return &chatStreamPipeline{container: container}
}

func (p *chatStreamPipeline) Stream(
	c *fiber.Ctx,
	rc *requestctx.Context,
	alias string,
	traceID string,
	idempotencyKey string,
	req models.ChatRequest,
) error {
	ctx := c.UserContext()
	routes := p.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}

	initialBudget, err := p.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if initialBudget.Exceeded {
		httputil.ApplyBudgetHeaders(c, initialBudget)
		_, _ = p.container.UsageLogger.Record(ctx, usagepipeline.Record{
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
	httputil.ApplyBudgetHeaders(c, initialBudget)

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := p.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return httputil.WriteError(c, fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	var once sync.Once
	releaseOnce := func() { once.Do(release) }

	return p.streamChat(c, alias, rc, traceID, idempotencyKey, req, routes, keyKey, keyCfg, tenantKey, tenantCfg, releaseOnce)
}

func (p *chatStreamPipeline) streamChat(
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
			p.container.Engine.ReportFailure(alias, route)
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
					if err := p.container.RateLimiter.TokenAllowance(ctx, keyKey, tokensUsed, keyCfg); err != nil {
						if errors.Is(err, limits.ErrLimitExceeded) {
							recordStatus = fiber.StatusTooManyRequests
						} else {
							recordStatus = fiber.StatusInternalServerError
						}
						recordSuccess = false
					}
					if err := p.container.RateLimiter.TokenAllowance(ctx, tenantKey, tokensUsed, tenantCfg); err != nil {
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

				if _, err := p.container.UsageLogger.Record(ctx, record); err != nil {
					slog.Error("record stream usage", slog.String("alias", alias), slog.String("error", err.Error()))
				}
			}

			defer recordUsage()
			defer func() {
				if !reported {
					p.container.Engine.ReportFailure(alias, route)
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

			p.container.Engine.ReportSuccess(alias, route)
			reported = true
		})

		return nil
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	if lastRoute.Provider != "" && rc != nil {
		_, _ = p.container.UsageLogger.Record(ctx, usagepipeline.Record{
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
