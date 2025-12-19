package user

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
	usageservice "github.com/ncecere/open_model_gateway/backend/internal/services/usage"
)

const userModelPerformanceWindow = 24 * time.Hour

type userModelResponse struct {
	Alias                  string              `json:"alias"`
	Provider               string              `json:"provider"`
	ModelType              string              `json:"model_type"`
	PriceInput             float64             `json:"price_input"`
	PriceOutput            float64             `json:"price_output"`
	Currency               string              `json:"currency"`
	Enabled                bool                `json:"enabled"`
	ThroughputTokensPerSec float64             `json:"throughput_tokens_per_sec"`
	AvgLatencyMs           float64             `json:"avg_latency_ms"`
	Status                 string              `json:"status"`
	PricingTiers           config.PricingTiers `json:"pricing_tiers,omitempty"`
}

func (h *userHandler) listModels(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.modelSvc == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "model catalog unavailable")
	}
	models, err := h.modelSvc.List(c.Context())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	scope := strings.TrimSpace(c.Query("scope"))
	tenantIDs, err := h.resolveModelTenants(c.Context(), user, scope)
	if err != nil {
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return httputil.WriteError(c, fiberErr.Code, fiberErr.Message)
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if len(tenantIDs) == 0 {
		return c.JSON(fiber.Map{"models": []userModelResponse{}})
	}
	allowedAliases, err := h.loadAllowedModelAliases(c.Context(), tenantIDs)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if len(allowedAliases) == 0 {
		return c.JSON(fiber.Map{"models": []userModelResponse{}})
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
		if _, ok := allowedAliases[strings.ToLower(strings.TrimSpace(model.Alias))]; !ok {
			continue
		}
		priceInput, _ := model.PriceInput.Float64()
		priceOutput, _ := model.PriceOutput.Float64()

		var pricing config.PricingTiers
		if len(model.PricingTiersJson) > 0 {
			if err := json.Unmarshal(model.PricingTiersJson, &pricing); err != nil {
				return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to decode pricing tiers")
			}
		}
		if priceInput == 0 {
			if fallback, ok := firstTierPrice(pricing, "input"); ok {
				priceInput = fallback
			}
		}
		if priceOutput == 0 {
			if fallback, ok := firstTierPrice(pricing, "output"); ok {
				priceOutput = fallback
			}
		}

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

func (h *userHandler) resolveModelTenants(ctx context.Context, user db.User, scope string) ([]uuid.UUID, error) {
	membershipIDs := make(map[uuid.UUID]struct{})
	var personalID uuid.UUID
	if user.PersonalTenantID.Valid {
		id, err := uuidFromPg(user.PersonalTenantID)
		if err != nil {
			return nil, err
		}
		personalID = id
		membershipIDs[id] = struct{}{}
	}
	if h.tenantSvc != nil {
		memberships, err := h.tenantSvc.ListUserMemberships(ctx, user)
		if err != nil {
			return nil, err
		}
		for _, membership := range memberships {
			membershipIDs[membership.TenantID] = struct{}{}
		}
	} else if scope != "" && !strings.EqualFold(scope, "all") && !strings.EqualFold(scope, "personal") {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "tenant service unavailable")
	}
	normalizedScope := strings.ToLower(strings.TrimSpace(scope))
	switch normalizedScope {
	case "", "all":
		if len(membershipIDs) == 0 {
			return nil, nil
		}
		ids := make([]uuid.UUID, 0, len(membershipIDs))
		for id := range membershipIDs {
			ids = append(ids, id)
		}
		return ids, nil
	case "personal":
		if personalID == uuid.Nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "personal tenant unavailable")
		}
		return []uuid.UUID{personalID}, nil
	default:
		tenantID, err := uuid.Parse(scope)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "invalid tenant scope")
		}
		if len(membershipIDs) == 0 {
			return nil, fiber.NewError(fiber.StatusForbidden, "tenant access denied")
		}
		if _, ok := membershipIDs[tenantID]; !ok {
			return nil, fiber.NewError(fiber.StatusForbidden, "tenant access denied")
		}
		return []uuid.UUID{tenantID}, nil
	}
}

func (h *userHandler) loadAllowedModelAliases(ctx context.Context, tenantIDs []uuid.UUID) (map[string]struct{}, error) {
	allowed := make(map[string]struct{})
	if len(tenantIDs) == 0 {
		return allowed, nil
	}
	if h == nil || h.container == nil || h.container.Queries == nil {
		return nil, errors.New("model access store unavailable")
	}
	seen := make(map[uuid.UUID]struct{}, len(tenantIDs))
	for _, tenantID := range tenantIDs {
		if tenantID == uuid.Nil {
			continue
		}
		if _, ok := seen[tenantID]; ok {
			continue
		}
		seen[tenantID] = struct{}{}
		aliases, err := h.container.Queries.ListTenantModels(ctx, toPgUUID(tenantID))
		if err != nil {
			return nil, err
		}
		for _, alias := range aliases {
			if norm := strings.ToLower(strings.TrimSpace(alias)); norm != "" {
				allowed[norm] = struct{}{}
			}
		}
	}
	return allowed, nil
}

func firstTierPrice(pricing config.PricingTiers, bucket string) (float64, bool) {
	if len(pricing) == 0 {
		return 0, false
	}
	for key, tiers := range pricing {
		if strings.EqualFold(key, bucket) && len(tiers) > 0 {
			return tiers[0].PricePerUnit, true
		}
	}
	return 0, false
}
