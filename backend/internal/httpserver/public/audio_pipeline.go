package public

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"log/slog"
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

type audioPipeline struct {
	container *app.Container
}

func newAudioPipeline(container *app.Container) *audioPipeline {
	return &audioPipeline{container: container}
}

func (p *audioPipeline) recordProviderSample(ctx context.Context, rc *requestctx.Context, alias string, route providers.Route, latency time.Duration, result providermetrics.Result, err error, usage models.Usage) {
	if p == nil || p.container == nil {
		return
	}
	p.container.RecordProviderTelemetry(ctx, providermetrics.Sample{
		Provider:     route.Provider,
		ModelAlias:   alias,
		Route:        route.ResolveDeployment(),
		Result:       result,
		ErrorClass:   providermetrics.ClassifyError(err),
		Latency:      latency,
		InputTokens:  int64(usage.PromptTokens),
		OutputTokens: int64(usage.CompletionTokens),
		Timestamp:    time.Now().UTC(),
	})
}

func (p *audioPipeline) Transcribe(c *fiber.Ctx, inv audioInvocation) error {
	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	if !p.container.IsModelAllowed(rc.TenantID, inv.Model) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}
	if p.container.Routing == nil || p.container.Routing.Engine == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "routing not configured")
	}
	routes := p.container.Routing.Engine.SelectRoutes(inv.Model)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}

	traceID := traceIDFromContext(c)

	budget, err := p.container.Telemetry.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if budget.Exceeded {
		httputil.ApplyBudgetHeaders(c, budget)
		return httputil.WriteError(c, fiber.StatusForbidden, "tenant budget exceeded")
	}
	httputil.ApplyBudgetHeaders(c, budget)

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := p.container.AcquireRateLimits(ctx, inv.Model)
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
	var formatRejected bool
	var granularityRejected bool

	for _, route := range routes {
		var resp models.AudioTranscriptionResponse
		var execErr error
		if !routeSupportsAudioFormat(route.Metadata, inv.Format) {
			formatRejected = true
			continue
		}
		if len(inv.Granular) > 0 && !routeSupportsGranularities(route.Metadata, inv.Granular) {
			granularityRejected = true
			continue
		}
		req := models.AudioTranscriptionRequest{
			Model: route.ResolveDeployment(),
			Task:  inv.Task,
			Input: models.AudioInput{
				Reader:      bytes.NewReader(inv.Payload),
				Filename:    inv.Filename,
				ContentType: inv.Mime,
				Bytes:       int64(len(inv.Payload)),
				FormField:   inv.Field,
			},
			Prompt:                 inv.Prompt,
			Temperature:            inv.Temp,
			Language:               inv.Language,
			ResponseFormat:         inv.Format,
			TimestampGranularities: inv.Granular,
			User:                   inv.User,
			Stream:                 inv.Stream,
		}
		start := time.Now()
		if inv.Task == models.AudioTranscriptionTaskTranslate && route.AudioTranslate != nil {
			resp, execErr = route.AudioTranslate.Translate(ctx, req)
		} else if route.AudioTranscribe != nil {
			resp, execErr = route.AudioTranscribe.Transcribe(ctx, req)
		} else {
			continue
		}
		if execErr != nil {
			p.container.Routing.Engine.ReportFailure(inv.Model, route)
			lastLatency = time.Since(start)
			lastErr = execErr
			lastRoute = route
			p.recordProviderSample(ctx, rc, inv.Model, route, lastLatency, providermetrics.ResultError, execErr, resp.Usage)
			continue
		}
		lastRoute = route
		p.container.Routing.Engine.ReportSuccess(inv.Model, route)
		if tokens := int(resp.Usage.TotalTokens); tokens > 0 {
			if err := p.container.RateLimits.RateLimiter.TokenAllowance(ctx, keyKey, tokens, keyCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, "token accounting failed")
			}
			if err := p.container.RateLimits.RateLimiter.TokenAllowance(ctx, tenantKey, tokens, tenantCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, "token accounting failed")
			}
		}
		record := usagepipeline.Record{
			Context:   rc,
			Alias:     inv.Model,
			Provider:  route.Provider,
			Usage:     resp.Usage,
			Latency:   time.Since(start),
			Status:    fiber.StatusOK,
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   true,
		}
		budgetStatus, recordErr := p.container.Telemetry.UsageLogger.Record(ctx, record)
		if recordErr == nil {
			httputil.ApplyBudgetHeaders(c, budgetStatus)
		}
		p.recordProviderSample(ctx, rc, inv.Model, route, time.Since(start), providermetrics.ResultSuccess, nil, resp.Usage)

		// Enrich wide event
		if event, ok := logging.WideEventFromContext(ctx); ok {
			event.SetModelContext(inv.Model, route.Provider, route.Model, route.ResolveDeployment())
			latency := time.Since(start)
			event.SetExecutionMetrics(latency.Milliseconds(), 0, len(routes))
			event.SetUsageMetrics(
				int64(resp.Usage.PromptTokens),
				int64(resp.Usage.CompletionTokens),
				int64(resp.Usage.TotalTokens),
				budgetStatus.LastRecordCostMicros,
			)
			remainingCents := budgetStatus.LimitCents - budgetStatus.TotalCostCents
			if remainingCents < 0 {
				remainingCents = 0
			}
			event.SetBudgetStatus(budgetStatus.TotalCostCents, remainingCents, budgetStatus.Exceeded)
		}

		return writeAudioTranscriptionResponse(c, resp)
	}

	if lastErr == nil {
		switch {
		case formatRejected:
			return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support requested response_format")
		case granularityRejected:
			return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support requested timestamp granularity")
		default:
			return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support audio tasks")
		}
	}
	if lastRoute.Provider != "" {
		_, _ = p.container.Telemetry.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     inv.Model,
			Provider:  lastRoute.Provider,
			Latency:   lastLatency,
			Status:    fiber.StatusBadGateway,
			ErrorCode: lastErr.Error(),
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
	}
	p.recordProviderSample(ctx, rc, inv.Model, lastRoute, lastLatency, providermetrics.ResultError, lastErr, models.Usage{})
	return httputil.WriteError(c, fiber.StatusBadGateway, lastErr.Error())
}

func (p *audioPipeline) TranscribeStream(c *fiber.Ctx, inv audioInvocation) error {
	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	if !p.container.IsModelAllowed(rc.TenantID, inv.Model) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}
	if p.container.Routing == nil || p.container.Routing.Engine == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "routing not configured")
	}
	routes := p.container.Routing.Engine.SelectRoutes(inv.Model)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}

	traceID := traceIDFromContext(c)

	budget, err := p.container.Telemetry.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if budget.Exceeded {
		httputil.ApplyBudgetHeaders(c, budget)
		return httputil.WriteError(c, fiber.StatusForbidden, "tenant budget exceeded")
	}
	httputil.ApplyBudgetHeaders(c, budget)

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := p.container.AcquireRateLimits(ctx, inv.Model)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return httputil.WriteError(c, fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	var once sync.Once
	releaseOnce := func() { once.Do(release) }

	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration
	var formatRejected bool
	var granularityRejected bool
	var streamingRejected bool

	for _, route := range routes {
		if route.AudioTranscribeStream == nil || !routeSupportsAudioStream(route.Metadata) {
			streamingRejected = true
			continue
		}
		if !routeSupportsAudioFormat(route.Metadata, inv.Format) {
			formatRejected = true
			continue
		}
		if len(inv.Granular) > 0 && !routeSupportsGranularities(route.Metadata, inv.Granular) {
			granularityRejected = true
			continue
		}

		req := models.AudioTranscriptionRequest{
			Model: route.ResolveDeployment(),
			Task:  inv.Task,
			Input: models.AudioInput{
				Reader:      bytes.NewReader(inv.Payload),
				Filename:    inv.Filename,
				ContentType: inv.Mime,
				Bytes:       int64(len(inv.Payload)),
				FormField:   inv.Field,
			},
			Prompt:                 inv.Prompt,
			Temperature:            inv.Temp,
			Language:               inv.Language,
			ResponseFormat:         inv.Format,
			TimestampGranularities: inv.Granular,
			User:                   inv.User,
			Stream:                 true,
		}

		start := time.Now()
		chunks, cancel, err := route.AudioTranscribeStream.TranscribeStream(ctx, req)
		if err != nil {
			lastErr = err
			lastLatency = time.Since(start)
			p.container.Routing.Engine.ReportFailure(inv.Model, route)
			continue
		}
		lastRoute = route

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		streamStart := time.Now()

		// Capture wide event before entering stream writer
		wideEvent, hasWideEvent := logging.WideEventFromContext(ctx)
		if hasWideEvent {
			wideEvent.MarkStreaming()
			wideEvent.SetModelContext(inv.Model, route.Provider, route.Model, route.ResolveDeployment())
		}

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			defer cancel()
			defer releaseOnce()

			recordStatus := fiber.StatusOK
			recordSuccess := false
			usageCaptured := false
			var streamUsage models.Usage
			var firstTokenLatency time.Duration
			var firstTokenMeasured bool
			var streamErr error

			recordUsage := func() {
				if usageCaptured {
					if tokens := int(streamUsage.TotalTokens); tokens > 0 {
						if err := p.container.RateLimits.RateLimiter.TokenAllowance(ctx, keyKey, tokens, keyCfg); err != nil {
							if errors.Is(err, limits.ErrLimitExceeded) {
								recordStatus = fiber.StatusTooManyRequests
							} else {
								recordStatus = fiber.StatusInternalServerError
							}
							recordSuccess = false
						}
						if err := p.container.RateLimits.RateLimiter.TokenAllowance(ctx, tenantKey, tokens, tenantCfg); err != nil {
							if errors.Is(err, limits.ErrLimitExceeded) {
								recordStatus = fiber.StatusTooManyRequests
							} else {
								recordStatus = fiber.StatusInternalServerError
							}
							recordSuccess = false
						}
					}
				}

				latency := time.Since(streamStart)
				if firstTokenMeasured && firstTokenLatency > 0 {
					latency = firstTokenLatency
				}
				record := usagepipeline.Record{
					Context:   rc,
					Alias:     inv.Model,
					Provider:  route.Provider,
					Usage:     streamUsage,
					Latency:   latency,
					Status:    recordStatus,
					TraceID:   traceID,
					Timestamp: time.Now().UTC(),
					Success:   recordSuccess && recordStatus == fiber.StatusOK,
				}
				budgetStatus, recordErr := p.container.Telemetry.UsageLogger.Record(ctx, record)
				if recordErr == nil {
					httputil.ApplyBudgetHeaders(c, budgetStatus)
				}
				result := providermetrics.ResultSuccess
				if !record.Success {
					result = providermetrics.ResultError
				}
				p.recordProviderSample(ctx, rc, inv.Model, route, latency, result, nil, streamUsage)

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
					wideEvent.SetBudgetStatus(budgetStatus.TotalCostCents, remainingCents, budgetStatus.Exceeded)
					if streamErr != nil {
						wideEvent.SetError("stream_error", "", streamErr.Error(), false)
					}
					wideEvent.Finalize(totalDuration.Milliseconds(), recordStatus, 0)
					wideEvent.Emit(slog.Default())
				}
			}

			defer recordUsage()
			defer func() {
				if recordSuccess && recordStatus == fiber.StatusOK {
					p.container.Routing.Engine.ReportSuccess(inv.Model, route)
				} else {
					p.container.Routing.Engine.ReportFailure(inv.Model, route)
				}
			}()

			for chunk := range chunks {
				if chunk.Err != nil {
					recordStatus = fiber.StatusBadGateway
					streamErr = chunk.Err
					lastErr = chunk.Err
					return
				}
				if len(chunk.Payload) == 0 {
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

				if _, err := w.WriteString("data: "); err != nil {
					recordStatus = fiber.StatusInternalServerError
					streamErr = err
					return
				}
				if _, err := w.Write(chunk.Payload); err != nil {
					recordStatus = fiber.StatusInternalServerError
					streamErr = err
					return
				}
				if _, err := w.WriteString("\n\n"); err != nil {
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

			if _, err := w.WriteString("data: [DONE]\n\n"); err != nil {
				recordStatus = fiber.StatusInternalServerError
				streamErr = err
			} else if err := w.Flush(); err != nil {
				recordStatus = fiber.StatusInternalServerError
				streamErr = err
			}
		})

		return nil
	}

	if lastErr == nil {
		switch {
		case formatRejected:
			return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support requested response_format")
		case granularityRejected:
			return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support requested timestamp granularity")
		case streamingRejected:
			return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support audio streaming")
		default:
			return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support audio tasks")
		}
	}
	if lastRoute.Provider != "" {
		_, _ = p.container.Telemetry.UsageLogger.Record(context.Background(), usagepipeline.Record{
			Context:   rc,
			Alias:     inv.Model,
			Provider:  lastRoute.Provider,
			Latency:   lastLatency,
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

func (p *audioPipeline) Speech(c *fiber.Ctx, req models.AudioSpeechRequest) error {
	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	alias := req.Model
	if !p.container.IsModelAllowed(rc.TenantID, alias) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}
	if p.container.Routing == nil || p.container.Routing.Engine == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "routing not configured")
	}
	routes := p.container.Routing.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}
	traceID := traceIDFromContext(c)
	budget, err := p.container.Telemetry.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if budget.Exceeded {
		httputil.ApplyBudgetHeaders(c, budget)
		return httputil.WriteError(c, fiber.StatusForbidden, "tenant budget exceeded")
	}
	httputil.ApplyBudgetHeaders(c, budget)

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := p.container.AcquireRateLimits(ctx, alias)
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
		providerReq := req
		providerReq.Model = route.ResolveDeployment()
		providerReq.Voice = resolveSpeechVoice(route.Metadata, providerReq.Voice)
		providerReq.Format = resolveSpeechFormat(route.Metadata, providerReq.Format)
		start := time.Now()
		lastRoute = route
		if providerReq.Stream {
			if route.TextToSpeechStream == nil {
				continue
			}
			lastLatency = time.Since(start)
			return httputil.WriteError(c, fiber.StatusNotImplemented, "streaming speech is not supported")
		}
		if route.TextToSpeech == nil {
			continue
		}
		resp, synthErr := route.TextToSpeech.Synthesize(ctx, providerReq)
		if synthErr != nil {
			lastErr = synthErr
			lastLatency = time.Since(start)
			p.container.Routing.Engine.ReportFailure(alias, route)
			p.recordProviderSample(ctx, rc, alias, route, lastLatency, providermetrics.ResultError, synthErr, resp.Usage)
			continue
		}
		p.container.Routing.Engine.ReportSuccess(alias, route)
		if tokens := int(resp.Usage.TotalTokens); tokens > 0 {
			if err := p.container.RateLimits.RateLimiter.TokenAllowance(ctx, keyKey, tokens, keyCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, "token accounting failed")
			}
			if err := p.container.RateLimits.RateLimiter.TokenAllowance(ctx, tenantKey, tokens, tenantCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, "token accounting failed")
			}
		}
		latency := time.Since(start)
		record := usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  route.Provider,
			Usage:     resp.Usage,
			Latency:   latency,
			Status:    fiber.StatusOK,
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   true,
		}
		budgetStatus, recordErr := p.container.Telemetry.UsageLogger.Record(ctx, record)
		if recordErr == nil {
			httputil.ApplyBudgetHeaders(c, budgetStatus)
		}
		p.recordProviderSample(ctx, rc, alias, route, latency, providermetrics.ResultSuccess, nil, resp.Usage)

		// Enrich wide event
		if event, ok := logging.WideEventFromContext(ctx); ok {
			event.SetModelContext(alias, route.Provider, route.Model, route.ResolveDeployment())
			event.SetExecutionMetrics(latency.Milliseconds(), 0, len(routes))
			event.SetUsageMetrics(
				int64(resp.Usage.PromptTokens),
				int64(resp.Usage.CompletionTokens),
				int64(resp.Usage.TotalTokens),
				budgetStatus.LastRecordCostMicros,
			)
			remainingCents := budgetStatus.LimitCents - budgetStatus.TotalCostCents
			if remainingCents < 0 {
				remainingCents = 0
			}
			event.SetBudgetStatus(budgetStatus.TotalCostCents, remainingCents, budgetStatus.Exceeded)
		}

		return writeAudioSpeechResponse(c, providerReq, resp)
	}

	if lastErr == nil {
		lastErr = errors.New("no backend available")
	}
	if lastRoute.Provider != "" {
		_, _ = p.container.Telemetry.UsageLogger.Record(ctx, usagepipeline.Record{
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
	p.recordProviderSample(ctx, rc, alias, lastRoute, lastLatency, providermetrics.ResultError, lastErr, models.Usage{})
	return httputil.WriteError(c, fiber.StatusBadGateway, lastErr.Error())
}
