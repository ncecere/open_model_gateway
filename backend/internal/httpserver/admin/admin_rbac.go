package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	adminrbacsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminrbac"
)

func requireTenantRole(c *fiber.Ctx, container *app.Container, tenantID uuid.UUID, role db.MembershipRole) error {
	userID, ok := adminUserIDFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "missing admin context")
	}

	superAdmin := false
	if user, ok := adminUserFromContext(c.UserContext()); ok && user.IsSuperAdmin {
		superAdmin = true
	}

	if container.AdminRBAC == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "rbac service unavailable")
	}
	if err := container.AdminRBAC.RequireTenantRole(c.UserContext(), tenantID, userID, role, superAdmin); err != nil {
		return mapRBACError(c, err)
	}
	return nil
}

func requireAnyRole(c *fiber.Ctx, container *app.Container, role db.MembershipRole) error {
	userID, ok := adminUserIDFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "missing admin context")
	}

	superAdmin := false
	if user, ok := adminUserFromContext(c.UserContext()); ok && user.IsSuperAdmin {
		superAdmin = true
	}

	if container.AdminRBAC == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "rbac service unavailable")
	}
	if err := container.AdminRBAC.RequireAnyRole(c.UserContext(), userID, role, superAdmin); err != nil {
		return mapRBACError(c, err)
	}
	return nil
}

func mapRBACError(c *fiber.Ctx, err error) error {
	switch {
	case err == adminrbacsvc.ErrUnauthorized:
		return httputil.WriteError(c, fiber.StatusUnauthorized, err.Error())
	case err == adminrbacsvc.ErrForbidden:
		return httputil.WriteError(c, fiber.StatusForbidden, err.Error())
	default:
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
}
