package user

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
	tenantservice "github.com/ncecere/open_model_gateway/backend/internal/services/tenant"
)

const tenantSummaryKey = "tenantSummary"

var (
	errTenantServiceUnavailable = errors.New("tenant service unavailable")
	errMembershipRequired       = errors.New("membership required")
	errInsufficientPermissions  = errors.New("insufficient permissions")
)

func (h *userHandler) requireTenantCapability(param string, capability rbac.Capability) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := userFromContext(c.UserContext())
		if !ok {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
		}
		tenantID, err := parseTenantParam(c, param)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		}
		summary, err := h.checkTenantCapability(c.Context(), user, tenantID, capability)
		if err != nil {
			return writeTenantCapabilityError(c, err)
		}
		c.Locals(tenantSummaryKey, summary)
		return c.Next()
	}
}

func (h *userHandler) requireTenantMembership(param string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := userFromContext(c.UserContext())
		if !ok {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
		}
		tenantID, err := parseTenantParam(c, param)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		}
		summary, err := h.checkTenantMembership(c.Context(), user, tenantID)
		if err != nil {
			return writeTenantCapabilityError(c, err)
		}
		c.Locals(tenantSummaryKey, summary)
		return c.Next()
	}
}

func (h *userHandler) checkTenantCapability(ctx context.Context, user db.User, tenantID uuid.UUID, capability rbac.Capability) (tenantservice.Summary, error) {
	if h.tenantSvc == nil {
		return tenantservice.Summary{}, errTenantServiceUnavailable
	}
	summary, err := h.tenantSvc.GetTenantSummary(ctx, user, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return tenantservice.Summary{}, errMembershipRequired
		}
		return tenantservice.Summary{}, err
	}
	if !rbac.HasCapability(summary.Role, capability) {
		return tenantservice.Summary{}, errInsufficientPermissions
	}
	return summary, nil
}

func (h *userHandler) checkTenantMembership(ctx context.Context, user db.User, tenantID uuid.UUID) (tenantservice.Summary, error) {
	if h.tenantSvc == nil {
		return tenantservice.Summary{}, errTenantServiceUnavailable
	}
	summary, err := h.tenantSvc.GetTenantSummary(ctx, user, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return tenantservice.Summary{}, errMembershipRequired
		}
		return tenantservice.Summary{}, err
	}
	return summary, nil
}

func writeTenantCapabilityError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, errMembershipRequired):
		return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
	case errors.Is(err, errInsufficientPermissions):
		return httputil.WriteError(c, fiber.StatusForbidden, "insufficient permissions")
	case errors.Is(err, errTenantServiceUnavailable):
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	default:
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
}

func parseTenantParam(c *fiber.Ctx, param string) (uuid.UUID, error) {
	tenantID := strings.TrimSpace(c.Params(param))
	if tenantID == "" {
		return uuid.Nil, errors.New("invalid tenant id")
	}
	return uuid.Parse(tenantID)
}

func tenantSummaryFromLocals(c *fiber.Ctx) (tenantservice.Summary, bool) {
	if c == nil {
		return tenantservice.Summary{}, false
	}
	val := c.Locals(tenantSummaryKey)
	if val == nil {
		return tenantservice.Summary{}, false
	}
	summary, ok := val.(tenantservice.Summary)
	return summary, ok
}
