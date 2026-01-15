package admin

import (
	"context"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

func registerAdminSystemMetricsRoutes(router fiber.Router, container *app.Container) {
	handler := &systemMetricsHandler{container: container}
	group := router.Group("/system")
	group.Get("/metrics", handler.getMetrics)
}

type systemMetricsHandler struct {
	container *app.Container
}

// SystemMetricsResponse contains comprehensive system health and usage metrics
type SystemMetricsResponse struct {
	// Infrastructure metrics
	Infrastructure InfrastructureMetrics `json:"infrastructure"`

	// Request metrics (last 24 hours)
	Requests RequestMetrics `json:"requests"`

	// Token metrics (last 24 hours)
	Tokens TokenMetrics `json:"tokens"`

	// Provider health summary
	Providers ProviderHealthSummary `json:"providers"`

	// Runtime metrics
	Runtime RuntimeMetrics `json:"runtime"`

	// Timestamp
	CollectedAt time.Time `json:"collected_at"`
}

type InfrastructureMetrics struct {
	Database DatabaseMetrics `json:"database"`
	Redis    RedisMetrics    `json:"redis"`
}

type DatabaseMetrics struct {
	Status          string `json:"status"`
	TotalConns      int32  `json:"total_connections"`
	IdleConns       int32  `json:"idle_connections"`
	AcquiredConns   int32  `json:"acquired_connections"`
	MaxConns        int32  `json:"max_connections"`
	LatencyMs       int64  `json:"latency_ms"`
	ConnectionUsage float64 `json:"connection_usage_percent"`
}

type RedisMetrics struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
}

type RequestMetrics struct {
	Total24h       int64   `json:"total_24h"`
	Total1h        int64   `json:"total_1h"`
	ErrorCount24h  int64   `json:"error_count_24h"`
	ErrorRate24h   float64 `json:"error_rate_24h"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P95LatencyMs   float64 `json:"p95_latency_ms"`
	RequestsPerMin float64 `json:"requests_per_minute"`
}

type TokenMetrics struct {
	Total24h        int64   `json:"total_24h"`
	Input24h        int64   `json:"input_24h"`
	Output24h       int64   `json:"output_24h"`
	TokensPerMin    float64 `json:"tokens_per_minute"`
	AvgTokensPerReq float64 `json:"avg_tokens_per_request"`
}

type ProviderHealthSummary struct {
	TotalProviders   int `json:"total_providers"`
	HealthyProviders int `json:"healthy_providers"`
	DegradedRoutes   int `json:"degraded_routes"`
	ActiveIncidents  int `json:"active_incidents"`
}

type RuntimeMetrics struct {
	Goroutines   int    `json:"goroutines"`
	HeapAllocMB  int    `json:"heap_alloc_mb"`
	HeapSysMB    int    `json:"heap_sys_mb"`
	NumGC        uint32 `json:"num_gc"`
	UptimeHours  float64 `json:"uptime_hours"`
}

var startTime = time.Now()

func (h *systemMetricsHandler) getMetrics(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}

	ctx := c.Context()
	now := time.Now()

	response := SystemMetricsResponse{
		CollectedAt: now,
	}

	// Collect infrastructure metrics
	response.Infrastructure = h.collectInfrastructureMetrics(ctx)

	// Collect request metrics from usage service
	response.Requests = h.collectRequestMetrics(ctx, now)

	// Collect token metrics
	response.Tokens = h.collectTokenMetrics(ctx, now)

	// Collect provider health summary
	response.Providers = h.collectProviderHealth(ctx)

	// Collect runtime metrics
	response.Runtime = h.collectRuntimeMetrics()

	return c.JSON(response)
}

func (h *systemMetricsHandler) collectInfrastructureMetrics(ctx context.Context) InfrastructureMetrics {
	metrics := InfrastructureMetrics{}

	// Database metrics
	if h.container.DBPool != nil {
		start := time.Now()
		err := h.container.DBPool.Ping(ctx)
		latency := time.Since(start).Milliseconds()

		stats := h.container.DBPool.Stat()
		metrics.Database = DatabaseMetrics{
			Status:          "operational",
			TotalConns:      stats.TotalConns(),
			IdleConns:       stats.IdleConns(),
			AcquiredConns:   stats.AcquiredConns(),
			MaxConns:        stats.MaxConns(),
			LatencyMs:       latency,
			ConnectionUsage: 0,
		}
		if stats.MaxConns() > 0 {
			metrics.Database.ConnectionUsage = float64(stats.AcquiredConns()) / float64(stats.MaxConns()) * 100
		}
		if err != nil {
			metrics.Database.Status = "degraded"
		}
	} else {
		metrics.Database.Status = "unavailable"
	}

	// Redis metrics
	if h.container.Redis != nil {
		start := time.Now()
		err := h.container.Redis.Ping(ctx).Err()
		latency := time.Since(start).Milliseconds()

		metrics.Redis = RedisMetrics{
			Status:    "operational",
			LatencyMs: latency,
		}
		if err != nil {
			metrics.Redis.Status = "degraded"
		}
	} else {
		metrics.Redis.Status = "unavailable"
	}

	return metrics
}

func (h *systemMetricsHandler) collectRequestMetrics(ctx context.Context, now time.Time) RequestMetrics {
	metrics := RequestMetrics{}

	if h.container.UsageService == nil || h.container.Data == nil || h.container.Data.Queries == nil {
		return metrics
	}

	// Get 24h request stats
	start24h := now.Add(-24 * time.Hour)
	start1h := now.Add(-1 * time.Hour)

	// Query aggregated request metrics
	rows, err := h.container.Data.Queries.AggregateRequestMetricsByModel(ctx, db.AggregateRequestMetricsByModelParams{
		Ts:   toPgTimestamptz(start24h),
		Ts_2: toPgTimestamptz(now),
	})
	if err == nil {
		for _, row := range rows {
			metrics.Total24h += row.Requests
		}
	}

	// Get 1h request count for rate calculation
	rows1h, err := h.container.Data.Queries.AggregateRequestMetricsByModel(ctx, db.AggregateRequestMetricsByModelParams{
		Ts:   toPgTimestamptz(start1h),
		Ts_2: toPgTimestamptz(now),
	})
	if err == nil {
		for _, row := range rows1h {
			metrics.Total1h += row.Requests
		}
	}

	// Calculate requests per minute (based on last hour)
	if metrics.Total1h > 0 {
		metrics.RequestsPerMin = float64(metrics.Total1h) / 60.0
	}

	// Get latency metrics
	latencyRows, err := h.container.Data.Queries.AggregateLatencyByModel(ctx, db.AggregateLatencyByModelParams{
		Ts:   toPgTimestamptz(start24h),
		Ts_2: toPgTimestamptz(now),
	})
	if err == nil && len(latencyRows) > 0 {
		var totalAvg, totalP95 float64
		var count int
		for _, row := range latencyRows {
			if row.SampleCount > 0 {
				totalAvg += row.AvgLatencyMs
				totalP95 += row.P95LatencyMs
				count++
			}
		}
		if count > 0 {
			metrics.AvgLatencyMs = totalAvg / float64(count)
			metrics.P95LatencyMs = totalP95 / float64(count)
		}
	}

	// Get error count from provider telemetry if available
	if h.container.ProviderTelemetrySvc != nil {
		slis, err := h.container.ProviderTelemetrySvc.CurrentSLIs(ctx)
		if err == nil {
			var totalErrors, totalRequests int64
			for _, sli := range slis {
				totalErrors += sli.ErrorCount + sli.TimeoutCount
				totalRequests += sli.RequestCount
			}
			metrics.ErrorCount24h = totalErrors
			if totalRequests > 0 {
				metrics.ErrorRate24h = float64(totalErrors) / float64(totalRequests) * 100
			}
		}
	}

	return metrics
}

func (h *systemMetricsHandler) collectTokenMetrics(ctx context.Context, now time.Time) TokenMetrics {
	metrics := TokenMetrics{}

	if h.container.Data == nil || h.container.Data.Queries == nil {
		return metrics
	}

	start24h := now.Add(-24 * time.Hour)
	start1h := now.Add(-1 * time.Hour)

	// Get token stats from usage records
	rows, err := h.container.Data.Queries.AggregateRequestMetricsByModel(ctx, db.AggregateRequestMetricsByModelParams{
		Ts:   toPgTimestamptz(start24h),
		Ts_2: toPgTimestamptz(now),
	})
	if err == nil {
		var totalRequests int64
		for _, row := range rows {
			metrics.Total24h += row.Tokens
			totalRequests += row.Requests
		}
		if totalRequests > 0 {
			metrics.AvgTokensPerReq = float64(metrics.Total24h) / float64(totalRequests)
		}
	}

	// Get 1h token count for throughput calculation
	rows1h, err := h.container.Data.Queries.AggregateRequestMetricsByModel(ctx, db.AggregateRequestMetricsByModelParams{
		Ts:   toPgTimestamptz(start1h),
		Ts_2: toPgTimestamptz(now),
	})
	if err == nil {
		var tokens1h int64
		for _, row := range rows1h {
			tokens1h += row.Tokens
		}
		metrics.TokensPerMin = float64(tokens1h) / 60.0
	}

	return metrics
}

func (h *systemMetricsHandler) collectProviderHealth(_ context.Context) ProviderHealthSummary {
	summary := ProviderHealthSummary{}

	// Get route health from engine
	if h.container.Engine != nil {
		healthStatus := h.container.Engine.HealthStatus()
		providers := make(map[string]bool)
		for alias, health := range healthStatus {
			// Track unique providers
			providers[alias] = true
			if health.HealthyRoutes < health.TotalRoutes {
				summary.DegradedRoutes += health.TotalRoutes - health.HealthyRoutes
			}
		}
		summary.TotalProviders = len(providers)
		summary.HealthyProviders = summary.TotalProviders // Will adjust below if degraded
	}

	// Get degraded routes from telemetry service
	if h.container.ProviderTelemetrySvc != nil {
		degraded := h.container.ProviderTelemetrySvc.DegradedRoutes()
		summary.DegradedRoutes = len(degraded)
		summary.ActiveIncidents = len(degraded) // Degraded routes represent active incidents

		// Adjust healthy providers based on degraded routes
		if summary.DegradedRoutes > 0 && summary.HealthyProviders > 0 {
			// Simple heuristic: count unique degraded providers
			degradedProviders := make(map[string]bool)
			for _, route := range degraded {
				degradedProviders[route] = true
			}
			if len(degradedProviders) > 0 {
				summary.HealthyProviders = summary.TotalProviders - len(degradedProviders)
				if summary.HealthyProviders < 0 {
					summary.HealthyProviders = 0
				}
			}
		}
	}

	return summary
}

func (h *systemMetricsHandler) collectRuntimeMetrics() RuntimeMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return RuntimeMetrics{
		Goroutines:  runtime.NumGoroutine(),
		HeapAllocMB: int(memStats.HeapAlloc / 1024 / 1024),
		HeapSysMB:   int(memStats.HeapSys / 1024 / 1024),
		NumGC:       memStats.NumGC,
		UptimeHours: time.Since(startTime).Hours(),
	}
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
