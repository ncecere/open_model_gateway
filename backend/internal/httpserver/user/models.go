package user

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
	usageservice "github.com/ncecere/open_model_gateway/backend/internal/services/usage"
)

const userModelPerformanceWindow = 24 * time.Hour

type userModelResponse struct {
	Alias                  string  `json:"alias"`
	Provider               string  `json:"provider"`
	ModelType              string  `json:"model_type"`
	PriceInput             float64 `json:"price_input"`
	PriceOutput            float64 `json:"price_output"`
	Currency               string  `json:"currency"`
	Enabled                bool    `json:"enabled"`
	ThroughputTokensPerSec float64 `json:"throughput_tokens_per_sec"`
	AvgLatencyMs           float64 `json:"avg_latency_ms"`
	Status                 string  `json:"status"`
}

func (h *userHandler) listModels(c *fiber.Ctx) error {
	if h.modelSvc == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "model catalog unavailable")
	}
	models, err := h.modelSvc.List(c.Context())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	var perf map[string]usageservice.ModelPerformanceStats
	if h.usage != nil {
		end := time.Now().UTC()
		start := end.Add(-userModelPerformanceWindow)
		perf, err = h.usage.ModelPerformance(c.Context(), start, end)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}
	var health map[string]router.RouteHealth
	if h.container != nil && h.container.Engine != nil {
		health = h.container.Engine.HealthStatus()
	}
	resp := make([]userModelResponse, 0, len(models))
	for _, model := range models {
		priceInput, _ := model.PriceInput.Float64()
		priceOutput, _ := model.PriceOutput.Float64()
		stats := usageservice.ModelPerformanceStats{}
		if perf != nil {
			if s, ok := perf[model.Alias]; ok {
				stats = s
			}
		}
		status := "online"
		if !model.Enabled {
			status = "disabled"
		} else if h, ok := health[model.Alias]; ok {
			switch {
			case h.TotalRoutes == 0:
				status = "unknown"
			case h.HealthyRoutes == 0:
				status = "offline"
			case h.HealthyRoutes < h.TotalRoutes:
				status = "degraded"
			default:
				status = "online"
			}
		}
		resp = append(resp, userModelResponse{
			Alias:                  model.Alias,
			Provider:               model.Provider,
			ModelType:              model.ModelType,
			PriceInput:             priceInput,
			PriceOutput:            priceOutput,
			Currency:               model.Currency,
			Enabled:                model.Enabled,
			ThroughputTokensPerSec: stats.ThroughputTokensPerSec,
			AvgLatencyMs:           stats.AvgLatencyMs,
			Status:                 status,
		})
	}
	return c.JSON(fiber.Map{"models": resp})
}
