package admin

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	admintenantsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admintenant"
)

func (h *tenantHandler) getRateLimits(c *fiber.Ctx) error {
	tenantUUID, err := parseTenantParam(c)
	if err != nil {
		return err
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	cfg, exists, err := h.service.GetTenantRateLimitOverride(c.Context(), tenantUUID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !exists {
		return httputil.WriteError(c, fiber.StatusNotFound, "rate limit override not set")
	}
	return c.JSON(mapTenantRateLimit(cfg))
}

func (h *tenantHandler) upsertRateLimits(c *fiber.Ctx) error {
	tenantUUID, err := parseTenantParam(c)
	if err != nil {
		return err
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleOwner); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	var req tenantRateLimitRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	cfg, err := h.service.UpsertTenantRateLimitOverride(c.Context(), tenantUUID, limits.LimitConfig{
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
	if err := recordAudit(c, h.container, "tenant.rate_limit.upsert", "tenant", tenantUUID.String(), fiber.Map{
		"requests_per_minute": cfg.RequestsPerMinute,
		"tokens_per_minute":   cfg.TokensPerMinute,
		"parallel_requests":   cfg.ParallelRequests,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(mapTenantRateLimit(cfg))
}

func (h *tenantHandler) deleteRateLimits(c *fiber.Ctx) error {
	tenantUUID, err := parseTenantParam(c)
	if err != nil {
		return err
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleOwner); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	if err := h.service.DeleteTenantRateLimitOverride(c.Context(), tenantUUID); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if err := recordAudit(c, h.container, "tenant.rate_limit.delete", "tenant", tenantUUID.String(), fiber.Map{}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func mapTenantRateLimit(cfg limits.LimitConfig) tenantRateLimitResponse {
	return tenantRateLimitResponse{
		RequestsPerMinute: cfg.RequestsPerMinute,
		TokensPerMinute:   cfg.TokensPerMinute,
		ParallelRequests:  cfg.ParallelRequests,
	}
}
