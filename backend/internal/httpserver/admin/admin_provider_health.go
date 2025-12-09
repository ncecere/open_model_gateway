package admin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/telemetry/provider"
)

func registerAdminProviderHealthRoutes(router fiber.Router, container *app.Container) {
	handler := &providerHealthHandler{container: container}
	group := router.Group("/providers")
	group.Get("/incidents", handler.listIncidents)
	group.Get("/degraded", handler.listDegraded)
	group.Get("/slis", handler.listSLIs)
	group.Get("/alerts", handler.listAlerts)
	group.Post("/seed/clear", handler.clearSeed)
}

type providerHealthHandler struct {
	container *app.Container
}

func (h *providerHealthHandler) listIncidents(c *fiber.Ctx) error {
	if h.container == nil || h.container.ProviderTelemetrySvc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "telemetry unavailable")
	}
	if seed := c.QueryBool("seed"); seed {
		_ = h.seedSampleIncident(c.Context())
	}
	provider := strings.TrimSpace(c.Query("provider"))
	alias := strings.TrimSpace(c.Query("alias"))
	status := strings.TrimSpace(c.Query("status"))
	limit := parseIntDefault(c.Query("limit"), 50, 1, 200)
	offset := parseIntDefault(c.Query("offset"), 0, 0, 10_000)

	params := db.ListProviderIncidentsParams{
		Provider:   provider,
		ModelAlias: alias,
		Status:     status,
		PageLimit:  int32(limit),
		PageOffset: int32(offset),
	}
	rows, err := h.container.Queries.ListProviderIncidents(c.Context(), params)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	total, _ := h.container.Queries.CountProviderIncidents(c.Context(), db.CountProviderIncidentsParams{
		Provider:   provider,
		ModelAlias: alias,
		Status:     status,
	})
	th := h.container.Config.Telemetry.Provider.Defaults

	incidents := make([]fiber.Map, 0, len(rows))
	for _, inc := range rows {
		incidents = append(incidents, fiber.Map{
			"id":            inc.ID,
			"provider":      inc.Provider,
			"alias":         inc.ModelAlias,
			"type":          inc.IncidentType,
			"status":        inc.Status,
			"window":        inc.WindowSeconds,
			"opened_at":     inc.OpenedAt,
			"resolved_at":   inc.ResolvedAt.Time,
			"window_start":  inc.WindowStartedAt,
			"window_end":    inc.WindowEndedAt,
			"request_count": inc.RequestCount,
			"error_count":   inc.ErrorCount,
			"timeout_count": inc.TimeoutCount,
			"latency_p95":   inc.LatencyP95Ms,
			"error_rate":    rate(inc.ErrorCount, inc.RequestCount),
			"timeout_rate":  rate(inc.TimeoutCount, inc.RequestCount),
			"thresholds": fiber.Map{
				"latency_p95_ms":         th.LatencyP95Ms,
				"error_rate_threshold":   th.ErrorRateThreshold,
				"timeout_rate_threshold": th.TimeoutRateThreshold,
				"min_samples":            th.MinSamples,
			},
			"metadata":     inc.Metadata,
			"sample_error": inc.SampleError.String,
		})
	}

	return c.JSON(fiber.Map{
		"incidents": incidents,
		"provider":  provider,
		"alias":     alias,
		"status":    status,
		"limit":     limit,
		"offset":    offset,
		"count":     len(incidents),
		"total":     total,
		"total_pages": func() int64 {
			if limit <= 0 {
				return 1
			}
			pages := total / int64(limit)
			if total%int64(limit) != 0 {
				pages++
			}
			if pages == 0 {
				return 1
			}
			return pages
		}(),
	})
}

func parseIntDefault(val string, def, min, max int) int {
	if v, err := strconv.Atoi(val); err == nil {
		if v < min {
			return min
		}
		if v > max {
			return max
		}
		return v
	}
	return def
}

func (h *providerHealthHandler) listDegraded(c *fiber.Ctx) error {
	if h.container == nil || h.container.ProviderTelemetrySvc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "telemetry unavailable")
	}
	degraded := h.container.ProviderTelemetrySvc.DegradedRoutes()
	return c.JSON(fiber.Map{"degraded": degraded})
}

func rate(num, denom int32) float64 {
	if denom == 0 {
		return 0
	}
	return float64(num) / float64(denom)
}

func (h *providerHealthHandler) listSLIs(c *fiber.Ctx) error {
	if h.container == nil || h.container.ProviderTelemetrySvc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "telemetry unavailable")
	}
	providerFilter := strings.TrimSpace(c.Query("provider"))
	aliasFilter := strings.TrimSpace(c.Query("alias"))
	if seed := c.QueryBool("seed"); seed {
		_ = h.seedSampleSLI(c.Context())
	}

	results, err := h.container.ProviderTelemetrySvc.CurrentSLIs(c.Context())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	slis := make([]fiber.Map, 0, len(results))
	for _, r := range results {
		if providerFilter != "" && providerFilter != r.Provider {
			continue
		}
		if aliasFilter != "" && aliasFilter != r.ModelAlias {
			continue
		}
		slis = append(slis, fiber.Map{
			"provider":       r.Provider,
			"alias":          r.ModelAlias,
			"route":          r.Route,
			"window_seconds": r.WindowSeconds,
			"window_start":   r.WindowStart,
			"window_end":     r.WindowEnd,
			"request_count":  r.RequestCount,
			"error_count":    r.ErrorCount,
			"timeout_count":  r.TimeoutCount,
			"latency_p95_ms": r.LatencyP95Ms,
			"error_rate":     r.ErrorRate,
			"timeout_rate":   r.TimeoutRate,
			"thresholds": fiber.Map{
				"latency_p95_ms":         r.Thresholds.LatencyP95Ms,
				"error_rate_threshold":   r.Thresholds.ErrorRateThreshold,
				"timeout_rate_threshold": r.Thresholds.TimeoutRateThreshold,
				"min_samples":            r.Thresholds.MinSamples,
			},
		})
	}
	return c.JSON(fiber.Map{
		"provider": providerFilter,
		"alias":    aliasFilter,
		"slis":     slis,
		"count":    len(slis),
	})
}

func (h *providerHealthHandler) listAlerts(c *fiber.Ctx) error {
	if h.container == nil || h.container.ProviderTelemetrySvc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "telemetry unavailable")
	}
	providerFilter := strings.TrimSpace(c.Query("provider"))
	alias := strings.TrimSpace(c.Query("alias"))
	limit := parseIntDefault(c.Query("limit"), 50, 1, 200)
	offset := parseIntDefault(c.Query("offset"), 0, 0, 10_000)

	params := db.ListProviderIncidentsParams{
		Provider:   providerFilter,
		ModelAlias: alias,
		Status:     string(provider.IncidentOpen),
		PageLimit:  int32(limit),
		PageOffset: int32(offset),
	}
	rows, err := h.container.Queries.ListProviderIncidents(c.Context(), params)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	active := make([]fiber.Map, 0, len(rows))
	for _, inc := range rows {
		active = append(active, fiber.Map{
			"id":          inc.ID,
			"provider":    inc.Provider,
			"alias":       inc.ModelAlias,
			"type":        inc.IncidentType,
			"opened_at":   inc.OpenedAt,
			"window":      inc.WindowSeconds,
			"request_cnt": inc.RequestCount,
			"errors":      inc.ErrorCount,
			"timeouts":    inc.TimeoutCount,
			"latency_p95": inc.LatencyP95Ms,
			"metadata":    inc.Metadata,
		})
	}
	return c.JSON(fiber.Map{
		"provider": providerFilter,
		"alias":    alias,
		"alerts":   active,
		"count":    len(active),
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *providerHealthHandler) seedSampleSLI(ctx context.Context) error {
	if h.container == nil || h.container.ProviderTelemetry == nil {
		return nil
	}
	now := time.Now().UTC()
	samples := []provider.Sample{
		{Provider: "openai", ModelAlias: "gpt-5", Route: "east", Result: provider.ResultSuccess, Latency: 1200 * time.Millisecond, Timestamp: now.Add(-2 * time.Minute)},
		{Provider: "openai", ModelAlias: "gpt-5", Route: "east", Result: provider.ResultError, ErrorClass: "5xx", Latency: 2500 * time.Millisecond, Timestamp: now.Add(-90 * time.Second)},
		{Provider: "openai", ModelAlias: "gpt-5", Route: "west", Result: provider.ResultTimeout, ErrorClass: "timeout", Latency: 4 * time.Second, Timestamp: now.Add(-75 * time.Second)},
		{Provider: "anthropic", ModelAlias: "claude-3", Route: "us-east", Result: provider.ResultSuccess, Latency: 800 * time.Millisecond, Timestamp: now.Add(-80 * time.Second)},
	}
	for _, s := range samples {
		_ = h.container.ProviderTelemetry.RecordSample(ctx, s)
	}
	return nil
}

func (h *providerHealthHandler) seedSampleIncident(ctx context.Context) error {
	if h.container == nil || h.container.Queries == nil {
		return nil
	}
	_, _ = h.container.Queries.UpsertProviderIncident(ctx, db.UpsertProviderIncidentParams{
		Provider:        "sample",
		ModelAlias:      "sample-alias",
		IncidentType:    "error_rate",
		Status:          "open",
		WindowSeconds:   300,
		OpenedAt:        provider.ToPgTime(time.Now().Add(-2 * time.Minute)),
		ResolvedAt:      pgtype.Timestamptz{Valid: false},
		WindowStartedAt: provider.ToPgTime(time.Now().Add(-5 * time.Minute)),
		WindowEndedAt:   provider.ToPgTime(time.Now()),
		SampleError:     provider.ToPgText("sample error"),
		RequestCount:    100,
		ErrorCount:      20,
		TimeoutCount:    5,
		LatencyP95Ms:    provider.ToPgInt4(2500),
		Metadata:        []byte(`{"seed":true}`),
	})
	return nil
}

func (h *providerHealthHandler) clearSeed(c *fiber.Ctx) error {
	if h.container == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "container unavailable")
	}
	ctx := c.Context()
	if h.container.DBPool != nil {
		_, _ = h.container.DBPool.Exec(ctx, `delete from provider_incidents where provider='sample' or metadata @> '{"seed":true}'`)
	}
	if h.container.Redis != nil {
		iter := h.container.Redis.Scan(ctx, 0, "telemetry:provider:sample*", 0).Iterator()
		var keys []string
		for iter.Next(ctx) {
			keys = append(keys, iter.Val())
		}
		if len(keys) > 0 {
			_, _ = h.container.Redis.Del(ctx, keys...).Result()
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}
