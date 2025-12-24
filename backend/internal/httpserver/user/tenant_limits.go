package user

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
	admintenantsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admintenant"
)

type tenantLimitRequest struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	ParallelRequests  int `json:"parallel_requests"`
}

type tenantLimitResponse struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	ParallelRequests  int `json:"parallel_requests"`
}

type tenantLimitPayload struct {
	Effective  tenantLimitResponse `json:"effective"`
	Ceiling    tenantLimitResponse `json:"ceiling"`
	Overridden bool                `json:"overridden"`
}

func (h *userHandler) registerTenantLimitRoutes(group fiber.Router) {
	group.Get("/tenants/:tenantID/limits",
		h.requireTenantMembership("tenantID"),
		h.getTenantLimits,
	)
	group.Put("/tenants/:tenantID/limits",
		h.requireTenantCapability("tenantID", rbac.CapabilityManageTenantLimits),
		h.updateTenantLimits,
	)
	group.Delete("/tenants/:tenantID/limits",
		h.requireTenantCapability("tenantID", rbac.CapabilityManageTenantLimits),
		h.deleteTenantLimits,
	)
}

func (h *userHandler) getTenantLimits(c *fiber.Ctx) error {
	if h.container == nil || h.container.AdminTenants == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	override, exists, err := h.container.AdminTenants.GetTenantRateLimitOverride(c.Context(), tenantUUID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	ceiling := h.container.DefaultTenantLimit
	effective := ceiling
	if exists {
		effective = override
	}
	return c.JSON(tenantLimitPayload{
		Effective:  mapTenantLimit(effective),
		Ceiling:    mapTenantLimit(ceiling),
		Overridden: exists,
	})
}

func (h *userHandler) updateTenantLimits(c *fiber.Ctx) error {
	if h.container == nil || h.container.AdminTenants == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	var req tenantLimitRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	if req.RequestsPerMinute <= 0 || req.TokensPerMinute <= 0 || req.ParallelRequests <= 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "rate limit values must be positive")
	}

	ceiling := h.container.DefaultTenantLimit
	if req.RequestsPerMinute > ceiling.RequestsPerMinute ||
		req.TokensPerMinute > ceiling.TokensPerMinute ||
		req.ParallelRequests > ceiling.ParallelRequests {
		return httputil.WriteError(c, fiber.StatusBadRequest, "tenant limits exceed configured ceiling")
	}

	cfg, err := h.container.AdminTenants.UpsertTenantRateLimitOverride(c.Context(), tenantUUID, limits.LimitConfig{
		RequestsPerMinute: req.RequestsPerMinute,
		TokensPerMinute:   req.TokensPerMinute,
		ParallelRequests:  req.ParallelRequests,
	})
	if err != nil {
		if errors.Is(err, admintenantsvc.ErrInvalidRateLimit) {
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := recordUserAudit(c, h.container, "tenant.limits.update", "tenant", tenantUUID.String(), fiber.Map{
		"requests_per_minute": cfg.RequestsPerMinute,
		"tokens_per_minute":   cfg.TokensPerMinute,
		"parallel_requests":   cfg.ParallelRequests,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(tenantLimitPayload{
		Effective:  mapTenantLimit(cfg),
		Ceiling:    mapTenantLimit(ceiling),
		Overridden: true,
	})
}

func (h *userHandler) deleteTenantLimits(c *fiber.Ctx) error {
	if h.container == nil || h.container.AdminTenants == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	if err := h.container.AdminTenants.DeleteTenantRateLimitOverride(c.Context(), tenantUUID); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if err := recordUserAudit(c, h.container, "tenant.limits.delete", "tenant", tenantUUID.String(), fiber.Map{}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func mapTenantLimit(cfg limits.LimitConfig) tenantLimitResponse {
	return tenantLimitResponse{
		RequestsPerMinute: cfg.RequestsPerMinute,
		TokensPerMinute:   cfg.TokensPerMinute,
		ParallelRequests:  cfg.ParallelRequests,
	}
}
