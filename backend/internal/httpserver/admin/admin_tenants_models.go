package admin

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

func (h *tenantHandler) getTenantModels(c *fiber.Ctx) error {
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
	models, err := h.service.ListModels(c.Context(), tenantUUID)
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	return c.JSON(fiber.Map{"models": models})
}

func (h *tenantHandler) upsertTenantModels(c *fiber.Ctx) error {
	tenantUUID, err := parseTenantParam(c)
	if err != nil {
		return err
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleOwner); err != nil {
		return err
	}

	var req tenantModelsRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	finalList, err := h.service.SetModels(c.Context(), tenantUUID, req.Models)
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	// Get tenant name for audit log
	auditMeta := fiber.Map{"models": finalList}
	if tenant, terr := h.container.Data.Queries.GetTenantByID(c.Context(), toPgUUID(tenantUUID)); terr == nil {
		auditMeta["name"] = tenant.Name
	}

	if err := recordAudit(c, h.container, "tenant.models.set", "tenant", tenantUUID.String(), auditMeta); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"models": finalList})
}

func (h *tenantHandler) deleteTenantModels(c *fiber.Ctx) error {
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
	if err := h.service.DeleteModels(c.Context(), tenantUUID); err != nil {
		return writeTenantServiceError(c, err)
	}

	// Get tenant name for audit log
	auditMeta := fiber.Map{}
	if tenant, terr := h.container.Data.Queries.GetTenantByID(c.Context(), toPgUUID(tenantUUID)); terr == nil {
		auditMeta["name"] = tenant.Name
	}

	if err := recordAudit(c, h.container, "tenant.models.clear", "tenant", tenantUUID.String(), auditMeta); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}
