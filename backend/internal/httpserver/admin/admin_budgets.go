package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	adminbudgetsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminbudget"
)

type budgetHandler struct {
	container *app.Container
	service   *adminbudgetsvc.Service
}

func registerAdminBudgetRoutes(router fiber.Router, container *app.Container) {
	handler := &budgetHandler{
		container: container,
		service:   container.AdminBudgets,
	}

	group := router.Group("/budgets")
	group.Get("/default", handler.getDefault)
	group.Put("/default", handler.updateDefault)
	group.Get("/overrides", handler.listOverrides)
	group.Put("/overrides/:tenantID", handler.upsertOverride)
	group.Delete("/overrides/:tenantID", handler.deleteOverride)
}

func (h *budgetHandler) getDefault(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}

	cfg := h.container.Config.Budgets
	var metadata *budgetDefaultsMetadata
	if h.service != nil {
		record, err := h.service.GetDefaults(c.Context())
		if err == nil {
			cfg = app.BudgetConfigFromRecord(cfg, record)
			metadata, err = h.buildBudgetDefaultsMetadata(c.Context(), record)
			if err != nil {
				return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
			}
		} else if err != nil && !errors.Is(err, pgx.ErrNoRows) && !errors.Is(err, adminbudgetsvc.ErrServiceUnavailable) {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	return c.JSON(mapBudgetDefaultsResponse(cfg, metadata))
}

type budgetDefaultsRequest struct {
	DefaultUSD           float64  `json:"default_usd"`
	WarningThreshold     float64  `json:"warning_threshold"`
	RefreshSchedule      string   `json:"refresh_schedule"`
	AlertEmails          []string `json:"alert_emails"`
	AlertWebhooks        []string `json:"alert_webhooks"`
	AlertCooldownSeconds int32    `json:"alert_cooldown_seconds"`
}

func (h *budgetHandler) updateDefault(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "budget service unavailable")
	}
	adminID, ok := adminUserIDFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "admin identity missing")
	}

	var req budgetDefaultsRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	record, err := h.service.UpdateDefaults(c.Context(), adminbudgetsvc.DefaultUpdate{
		DefaultUSD:           req.DefaultUSD,
		WarningThreshold:     req.WarningThreshold,
		RefreshSchedule:      req.RefreshSchedule,
		AlertEmails:          req.AlertEmails,
		AlertWebhooks:        req.AlertWebhooks,
		AlertCooldownSeconds: req.AlertCooldownSeconds,
		UpdatedByUserID:      adminID,
	})
	if err != nil {
		return writeBudgetError(c, err)
	}

	updated := app.BudgetConfigFromRecord(h.container.Config.Budgets, record)
	h.container.UpdateBudgetConfig(updated)

	if err := recordAudit(c, h.container, "budget.default.update", "budget_default", "global", fiber.Map{
		"default_usd":            updated.DefaultUSD,
		"warning_threshold_perc": updated.WarningThresholdPerc,
		"refresh_schedule":       updated.RefreshSchedule,
		"alert_emails":           updated.Alert.Emails,
		"alert_webhooks":         updated.Alert.Webhooks,
		"alert_cooldown_seconds": int(updated.Alert.Cooldown / time.Second),
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	meta, err := h.buildBudgetDefaultsMetadata(c.Context(), record)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(mapBudgetDefaultsResponse(updated, meta))
}

func (h *budgetHandler) listOverrides(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "budget service unavailable")
	}

	tenantParam := strings.TrimSpace(c.Query("tenant_id"))
	if tenantParam != "" {
		tenantID, err := uuid.Parse(tenantParam)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant_id")
		}
		if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleViewer); err != nil {
			return err
		}
		override, err := h.service.GetOverride(c.Context(), tenantID)
		if err != nil {
			if err == pgx.ErrNoRows {
				return c.JSON(fiber.Map{"overrides": []fiber.Map{}})
			}
			return writeBudgetError(c, err)
		}
		payload := mapBudgetOverride(override)
		return c.JSON(fiber.Map{"overrides": []fiber.Map{payload}})
	}

	overrides, err := h.service.ListOverrides(c.Context())
	if err != nil {
		return writeBudgetError(c, err)
	}
	payload := make([]fiber.Map, 0, len(overrides))
	for _, ov := range overrides {
		payload = append(payload, mapBudgetOverride(ov))
	}
	return c.JSON(fiber.Map{"overrides": payload})
}

type budgetOverrideRequest struct {
	BudgetUSD            float64  `json:"budget_usd"`
	WarningThreshold     float64  `json:"warning_threshold"`
	RefreshSchedule      string   `json:"refresh_schedule"`
	AlertEmails          []string `json:"alert_emails"`
	AlertWebhooks        []string `json:"alert_webhooks"`
	AlertCooldownSeconds *int32   `json:"alert_cooldown_seconds"`
}

func (h *budgetHandler) upsertOverride(c *fiber.Ctx) error {
	tenantUUID, err := parseTenantParam(c)
	if err != nil {
		return err
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleOwner); err != nil {
		return err
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "budget service unavailable")
	}

	var req budgetOverrideRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	override, err := h.service.UpsertOverride(c.Context(), tenantUUID, adminbudgetsvc.OverrideRequest{
		BudgetUSD:            req.BudgetUSD,
		WarningThreshold:     req.WarningThreshold,
		RefreshSchedule:      req.RefreshSchedule,
		AlertEmails:          req.AlertEmails,
		AlertWebhooks:        req.AlertWebhooks,
		AlertCooldownSeconds: req.AlertCooldownSeconds,
	})
	if err != nil {
		return writeBudgetError(c, err)
	}

	if err := recordAudit(c, h.container, "budget.override.upsert", "budget_override", tenantUUID.String(), fiber.Map{
		"budget_usd":        req.BudgetUSD,
		"warning_threshold": req.WarningThreshold,
		"refresh_schedule":  req.RefreshSchedule,
		"alert_emails":      req.AlertEmails,
		"alert_webhooks":    req.AlertWebhooks,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(mapBudgetOverride(override))
}

func (h *budgetHandler) deleteOverride(c *fiber.Ctx) error {
	tenantUUID, err := parseTenantParam(c)
	if err != nil {
		return err
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleOwner); err != nil {
		return err
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "budget service unavailable")
	}

	if err := h.service.DeleteOverride(c.Context(), tenantUUID); err != nil {
		return writeBudgetError(c, err)
	}

	if err := recordAudit(c, h.container, "budget.override.delete", "budget_override", tenantUUID.String(), fiber.Map{}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func parseTenantParam(c *fiber.Ctx) (uuid.UUID, error) {
	tenantIDParam := strings.TrimSpace(c.Params("tenantID"))
	if tenantIDParam == "" {
		return uuid.UUID{}, httputil.WriteError(c, fiber.StatusBadRequest, "tenant id required")
	}
	tenantUUID, err := uuid.Parse(tenantIDParam)
	if err != nil {
		return uuid.UUID{}, httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	return tenantUUID, nil
}

func mapBudgetOverride(ov db.TenantBudgetOverride) fiber.Map {
	warn, _ := ov.WarningThreshold.Float64()
	tenantID, _ := fromPgUUID(ov.TenantID)
	created, _ := timeFromPg(ov.CreatedAt)
	updated, _ := timeFromPg(ov.UpdatedAt)
	budget, _ := ov.BudgetUsd.Float64()
	lastAlertAt, _ := timeFromPg(ov.LastAlertAt)
	lastAlert := ""
	if ov.LastAlertAt.Valid {
		lastAlert = lastAlertAt.Format(time.RFC3339)
	}
	return fiber.Map{
		"tenant_id":              tenantID.String(),
		"budget_usd":             budget,
		"warning_threshold":      warn,
		"refresh_schedule":       ov.RefreshSchedule,
		"alert_emails":           ov.AlertEmails,
		"alert_webhooks":         ov.AlertWebhooks,
		"alert_cooldown_seconds": ov.AlertCooldownSeconds,
		"last_alert_at":          lastAlert,
		"last_alert_level":       ov.LastAlertLevel.String,
		"created_at":             created.Format(time.RFC3339),
		"updated_at":             updated.Format(time.RFC3339),
	}
}

func mapBudgetDefaultsResponse(cfg config.BudgetConfig, meta *budgetDefaultsMetadata) fiber.Map {
	response := fiber.Map{
		"default_usd":            cfg.DefaultUSD,
		"warning_threshold_perc": cfg.WarningThresholdPerc,
		"refresh_schedule":       cfg.RefreshSchedule,
		"alert": fiber.Map{
			"enabled":          cfg.Alert.Enabled,
			"emails":           cfg.Alert.Emails,
			"webhooks":         cfg.Alert.Webhooks,
			"cooldown_seconds": int(cfg.Alert.Cooldown / time.Second),
		},
	}
	if meta != nil {
		metaMap := fiber.Map{}
		if meta.CreatedAt != "" {
			metaMap["created_at"] = meta.CreatedAt
		}
		if meta.UpdatedAt != "" {
			metaMap["updated_at"] = meta.UpdatedAt
		}
		if ref := mapBudgetUserRef(meta.CreatedBy); ref != nil {
			metaMap["created_by"] = ref
		}
		if ref := mapBudgetUserRef(meta.UpdatedBy); ref != nil {
			metaMap["updated_by"] = ref
		}
		if len(metaMap) > 0 {
			response["metadata"] = metaMap
		}
	}
	return response
}

type budgetDefaultsMetadata struct {
	CreatedAt string
	UpdatedAt string
	CreatedBy *budgetUserRef
	UpdatedBy *budgetUserRef
}

type budgetUserRef struct {
	ID    string
	Name  string
	Email string
}

func mapBudgetUserRef(ref *budgetUserRef) fiber.Map {
	if ref == nil {
		return nil
	}
	return fiber.Map{
		"id":    ref.ID,
		"name":  ref.Name,
		"email": ref.Email,
	}
}

func (h *budgetHandler) buildBudgetDefaultsMetadata(ctx context.Context, record db.BudgetDefault) (*budgetDefaultsMetadata, error) {
	meta := &budgetDefaultsMetadata{}
	if created, err := timeFromPg(record.CreatedAt); err == nil {
		meta.CreatedAt = created.Format(time.RFC3339)
	}
	if updated, err := timeFromPg(record.UpdatedAt); err == nil {
		meta.UpdatedAt = updated.Format(time.RFC3339)
	}
	var err error
	meta.CreatedBy, err = h.lookupBudgetUser(ctx, record.CreatedByUserID)
	if err != nil {
		return nil, err
	}
	meta.UpdatedBy, err = h.lookupBudgetUser(ctx, record.UpdatedByUserID)
	if err != nil {
		return nil, err
	}
	if meta.CreatedAt == "" && meta.UpdatedAt == "" && meta.CreatedBy == nil && meta.UpdatedBy == nil {
		return nil, nil
	}
	return meta, nil
}

func (h *budgetHandler) lookupBudgetUser(ctx context.Context, id pgtype.UUID) (*budgetUserRef, error) {
	if !id.Valid || h.container == nil || h.container.Queries == nil {
		return nil, nil
	}
	user, err := h.container.Queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	userID, err := fromPgUUID(user.ID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = user.Email
	}
	return &budgetUserRef{
		ID:    userID.String(),
		Name:  name,
		Email: user.Email,
	}, nil
}

func writeBudgetError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, adminbudgetsvc.ErrInvalidDefault),
		errors.Is(err, adminbudgetsvc.ErrInvalidThreshold),
		errors.Is(err, adminbudgetsvc.ErrInvalidOverride):
		status = fiber.StatusBadRequest
	case errors.Is(err, adminbudgetsvc.ErrServiceUnavailable):
		status = fiber.StatusInternalServerError
	}
	return httputil.WriteError(c, status, err.Error())
}
