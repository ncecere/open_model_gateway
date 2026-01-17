package public

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/logging"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
	providermetrics "github.com/ncecere/open_model_gateway/backend/internal/telemetry/provider"
)

type chatStreamPipeline struct {
	container *app.Container
}

type streamRenderer interface {
	ContentType() string
	Init(w *bufio.Writer) error
	HandleChunk(chunk models.ChatChunk, w *bufio.Writer) error
	Done(w *bufio.Writer, usage *models.Usage) error
}

type chatCompletionStreamRenderer struct {
	alias string
}

func (r *chatCompletionStreamRenderer) ContentType() string { return "text/event-stream" }

func (r *chatCompletionStreamRenderer) Init(*bufio.Writer) error { return nil }

func (r *chatCompletionStreamRenderer) HandleChunk(chunk models.ChatChunk, w *bufio.Writer) error {
	payload := convertStreamChunk(chunk, r.alias)
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err = w.WriteString("data: "); err != nil {
		return err
	}
	if _, err = w.Write(data); err != nil {
		return err
	}
	if _, err = w.WriteString("\n\n"); err != nil {
		return err
	}
	return nil
}

func (r *chatCompletionStreamRenderer) Done(w *bufio.Writer, _ *models.Usage) error {
	if _, err := w.WriteString("data: [DONE]\n\n"); err != nil {
		return err
	}
	return w.Flush()
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
	renderer := &chatCompletionStreamRenderer{alias: alias}
	return p.streamWithRenderer(c, rc, alias, traceID, idempotencyKey, req, renderer)
}

func (p *chatStreamPipeline) StreamResponses(
	c *fiber.Ctx,
	rc *requestctx.Context,
	alias string,
	traceID string,
	idempotencyKey string,
	req models.ChatRequest,
	opts openAIResponseOptions,
) error {
	renderer := newResponsesStreamRenderer(alias, opts)
	return p.streamWithRenderer(c, rc, alias, traceID, idempotencyKey, req, renderer)
}

func (p *chatStreamPipeline) streamWithRenderer(
	c *fiber.Ctx,
	rc *requestctx.Context,
	alias string,
	traceID string,
	idempotencyKey string,
	req models.ChatRequest,
	renderer streamRenderer,
) error {
	ctx := c.UserContext()
	routes := p.container.Routing.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
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
			return httputil.WriteError(c, fiber.StatusBadRequest, fmt.Sprintf("model does not support %s input", reqCaps.Describe()))
		}
		routes = eligible
	}

	initialBudget, err := p.container.Telemetry.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if initialBudget.Exceeded {
		httputil.ApplyBudgetHeaders(c, initialBudget)
		_, _ = p.container.Telemetry.UsageLogger.Record(ctx, usagepipeline.Record{
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

	return p.streamChat(c, alias, rc, traceID, idempotencyKey, req, routes, keyKey, keyCfg, tenantKey, tenantCfg, releaseOnce, renderer)
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
	renderer streamRenderer,
) error {
	ctx := c.UserContext()
	if renderer == nil {
		renderer = &chatCompletionStreamRenderer{alias: alias}
	}

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
			p.container.Routing.Engine.ReportFailure(alias, route)
			lastErr = err
			p.container.RecordProviderTelemetry(ctx, providermetrics.Sample{
				Provider:   route.Provider,
				ModelAlias: alias,
				Route:      route.ResolveDeployment(),
				Result:     providermetrics.ResultError,
				ErrorClass: providermetrics.ClassifyError(err),
				Timestamp:  time.Now().UTC(),
			})
			continue
		}
		lastRoute = route

		contentType := renderer.ContentType()
		if strings.TrimSpace(contentType) == "" {
			contentType = "text/event-stream"
		}
		c.Set("Content-Type", contentType)
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")

		streamStart := time.Now()
		var firstTokenLatency time.Duration
		var firstTokenMeasured bool
		if p.container.Telemetry.Observability != nil {
			p.container.Telemetry.Observability.IncProviderStream(route.Provider, alias, route.ResolveDeployment())
		}

		// Capture wide event before entering stream writer and mark as streaming
		wideEvent, hasWideEvent := logging.WideEventFromContext(ctx)
		if hasWideEvent {
			wideEvent.MarkStreaming()
			wideEvent.ModelAlias = alias
			wideEvent.SetModelContext(alias, route.Provider, route.Model, route.ResolveDeployment())
			wideEvent.RouteCount = len(routes)
		}

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			defer cancel()
			defer release()
			defer func() {
				if p.container.Telemetry.Observability != nil {
					p.container.Telemetry.Observability.DecProviderStream(route.Provider, alias, route.ResolveDeployment())
				}
			}()

			recordStatus := fiber.StatusOK
			recordSuccess := false
			reported := false
			var streamUsage models.Usage
			usageCaptured := false
			var streamErr error

			if err := renderer.Init(w); err != nil {
				recordStatus = fiber.StatusInternalServerError
				streamErr = err
				return
			}

			recordUsage := func() {
				tokensUsed := int(streamUsage.TotalTokens)
				if usageCaptured && tokensUsed > 0 {
					if err := p.container.RateLimits.RateLimiter.TokenAllowance(ctx, keyKey, tokensUsed, keyCfg); err != nil {
						if errors.Is(err, limits.ErrLimitExceeded) {
							recordStatus = fiber.StatusTooManyRequests
						} else {
							recordStatus = fiber.StatusInternalServerError
						}
						recordSuccess = false
					}
					if err := p.container.RateLimits.RateLimiter.TokenAllowance(ctx, tenantKey, tokensUsed, tenantCfg); err != nil {
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
				result := providermetrics.ResultSuccess
				if recordStatus != fiber.StatusOK || !recordSuccess {
					result = providermetrics.ResultError
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

				budgetStatus, recordErr := p.container.Telemetry.UsageLogger.Record(ctx, record)
				if recordErr != nil {
					slog.Error("record stream usage", slog.String("alias", alias), slog.String("error", recordErr.Error()))
				}
				p.container.RecordProviderTelemetry(ctx, providermetrics.Sample{
					Provider:     route.Provider,
					ModelAlias:   alias,
					Route:        route.ResolveDeployment(),
					Result:       result,
					ErrorClass:   providermetrics.ClassifyError(recordErr),
					Latency:      latency,
					InputTokens:  int64(streamUsage.PromptTokens),
					OutputTokens: int64(streamUsage.CompletionTokens),
					Timestamp:    time.Now().UTC(),
				})

				// Emit wide event at stream close
				if hasWideEvent {
					totalDuration := time.Since(streamStart)
					wideEvent.SetUsageMetrics(
						int64(streamUsage.PromptTokens),
						int64(streamUsage.CompletionTokens),
						int64(streamUsage.TotalTokens),
						budgetStatus.LastRecordCostMicros,
					)
					wideEvent.SetExecutionMetrics(latency.Milliseconds(), 0, len(routes))
					remainingCents := budgetStatus.LimitCents - budgetStatus.TotalCostCents
					if remainingCents < 0 {
						remainingCents = 0
					}
					wideEvent.SetBudgetStatus(
						budgetStatus.TotalCostCents,
						remainingCents,
						budgetStatus.Exceeded,
					)
					if streamErr != nil {
						wideEvent.SetError("stream_error", "", streamErr.Error(), false)
					}
					wideEvent.Finalize(totalDuration.Milliseconds(), recordStatus, 0)
					wideEvent.Emit(slog.Default())
				}
			}

			defer recordUsage()
			defer func() {
				if !reported {
					p.container.Routing.Engine.ReportFailure(alias, route)
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

				if err := renderer.HandleChunk(chunk, w); err != nil {
					recordStatus = fiber.StatusInternalServerError
					streamErr = err
					return
				}
				if err := w.Flush(); err != nil {
					recordStatus = fiber.StatusInternalServerError
					streamErr = err
					return
				}

				if chunk.Usage != nil {
					streamUsage = *chunk.Usage
					usageCaptured = true
				}

				recordSuccess = true
			}

			var usagePtr *models.Usage
			if usageCaptured {
				usageCopy := streamUsage
				usagePtr = &usageCopy
			}
			if err := renderer.Done(w, usagePtr); err != nil {
				recordStatus = fiber.StatusInternalServerError
				streamErr = err
				return
			}

			p.container.Routing.Engine.ReportSuccess(alias, route)
			reported = true
		})

		return nil
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	if lastRoute.Provider != "" && rc != nil {
		_, _ = p.container.Telemetry.UsageLogger.Record(ctx, usagepipeline.Record{
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
