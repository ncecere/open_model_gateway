package admin

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	adminbudgetsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminbudget"
)

func (h *tenantHandler) getBudget(c *fiber.Ctx) error {
	tenantUUID, err := parseTenantParam(c)
	if err != nil {
		return err
	}

	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleViewer); err != nil {
		return err
	}

	if h.container.AdminBudgets == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "budget service unavailable")
	}
	override, err := h.container.AdminBudgets.GetOverride(c.Context(), tenantUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "budget override not set")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	payload := mapBudgetOverride(override)
	if err := recordAudit(c, h.container, "tenant.budget.view", "tenant", tenantUUID.String(), payload); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(payload)
}

func (h *tenantHandler) upsertBudget(c *fiber.Ctx) error {
	tenantUUID, err := parseTenantParam(c)
	if err != nil {
		return err
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleOwner); err != nil {
		return err
	}

	if h.container.AdminBudgets == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "budget service unavailable")
	}

	var req budgetOverrideRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	override, err := h.container.AdminBudgets.UpsertOverride(c.Context(), tenantUUID, adminbudgetsvc.OverrideRequest{
		BudgetUSD:            req.BudgetUSD,
		WarningThreshold:     req.WarningThreshold,
		RefreshSchedule:      req.RefreshSchedule,
		AlertEmails:          req.AlertEmails,
		AlertWebhooks:        req.AlertWebhooks,
		AlertCooldownSeconds: req.AlertCooldownSeconds,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := recordAudit(c, h.container, "tenant.budget.upsert", "tenant", tenantUUID.String(), fiber.Map{
		"budget_usd":             req.BudgetUSD,
		"warning_threshold":      req.WarningThreshold,
		"refresh_schedule":       override.RefreshSchedule,
		"alert_emails":           override.AlertEmails,
		"alert_webhooks":         override.AlertWebhooks,
		"alert_cooldown_seconds": override.AlertCooldownSeconds,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(mapBudgetOverride(override))
}

func (h *tenantHandler) deleteBudget(c *fiber.Ctx) error {
	tenantUUID, err := parseTenantParam(c)
	if err != nil {
		return err
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleOwner); err != nil {
		return err
	}

	if h.container.AdminBudgets == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "budget service unavailable")
	}
	if err := h.container.AdminBudgets.DeleteOverride(c.Context(), tenantUUID); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := recordAudit(c, h.container, "tenant.budget.delete", "tenant", tenantUUID.String(), fiber.Map{}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}
