package admin

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	adminratelimitsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminratelimit"
)

type rateLimitHandler struct {
	container *app.Container
	service   *adminratelimitsvc.Service
}

func registerAdminRateLimitRoutes(router fiber.Router, container *app.Container) {
	handler := &rateLimitHandler{
		container: container,
		service:   container.AdminRateLimits,
	}
	group := router.Group("/settings/rate-limits")
	group.Get("/", handler.getDefaults)
	group.Put("/", handler.updateDefaults)
}

func (h *rateLimitHandler) getDefaults(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	cfg := h.container.Config.RateLimits
	return c.JSON(fiber.Map{
		"requests_per_minute":      cfg.DefaultRequestsPerMinute,
		"tokens_per_minute":        cfg.DefaultTokensPerMinute,
		"parallel_requests_key":    cfg.DefaultParallelRequestsKey,
		"parallel_requests_tenant": cfg.DefaultParallelRequestsTenant,
	})
}

type rateLimitDefaultsRequest struct {
	RequestsPerMinute      int `json:"requests_per_minute"`
	TokensPerMinute        int `json:"tokens_per_minute"`
	ParallelRequestsKey    int `json:"parallel_requests_key"`
	ParallelRequestsTenant int `json:"parallel_requests_tenant"`
}

func (h *rateLimitHandler) updateDefaults(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "rate limit service unavailable")
	}
	var req rateLimitDefaultsRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	cfg, err := h.service.UpdateDefaults(c.Context(), adminratelimitsvc.DefaultUpdate{
		RequestsPerMinute:      req.RequestsPerMinute,
		TokensPerMinute:        req.TokensPerMinute,
		ParallelRequestsKey:    req.ParallelRequestsKey,
		ParallelRequestsTenant: req.ParallelRequestsTenant,
	})
	if err != nil {
		switch err {
		case adminratelimitsvc.ErrInvalidRateLimit:
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		case adminratelimitsvc.ErrServiceUnavailable:
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		default:
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}
	h.container.UpdateRateLimitConfig(cfg)

	if err := recordAudit(c, h.container, "rate_limits.default.update", "rate_limit_default", "global", fiber.Map{
		"requests_per_minute":      cfg.DefaultRequestsPerMinute,
		"tokens_per_minute":        cfg.DefaultTokensPerMinute,
		"parallel_requests_key":    cfg.DefaultParallelRequestsKey,
		"parallel_requests_tenant": cfg.DefaultParallelRequestsTenant,
	}); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"requests_per_minute":      cfg.DefaultRequestsPerMinute,
		"tokens_per_minute":        cfg.DefaultTokensPerMinute,
		"parallel_requests_key":    cfg.DefaultParallelRequestsKey,
		"parallel_requests_tenant": cfg.DefaultParallelRequestsTenant,
	})
}
