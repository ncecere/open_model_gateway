package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/batchdto"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
	adminbudgetsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminbudget"
	admintenantsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admintenant"
)

func registerAdminTenantRoutes(router fiber.Router, container *app.Container) {
	handler := &tenantHandler{container: container, service: container.AdminTenants}

	group := router.Group("/tenants")
	group.Get("/", handler.list)
	group.Get("/personal", handler.listPersonal)
	group.Post("/", handler.create)
	group.Patch("/:tenantID", handler.updateDetails)
	group.Patch("/:tenantID/status", handler.updateStatus)
	group.Get("/:tenantID/budget", handler.getBudget)
	group.Put("/:tenantID/budget", handler.upsertBudget)
	group.Delete("/:tenantID/budget", handler.deleteBudget)
	group.Get("/:tenantID/models", handler.getTenantModels)
	group.Put("/:tenantID/models", handler.upsertTenantModels)
	group.Delete("/:tenantID/models", handler.deleteTenantModels)
	group.Get("/:tenantID/api-keys", handler.listAPIKeys)
	group.Post("/:tenantID/api-keys", handler.createAPIKey)
	group.Delete("/:tenantID/api-keys/:apiKeyID", handler.revokeAPIKey)
	group.Get("/:tenantID/memberships", handler.listMemberships)
	group.Post("/:tenantID/memberships", handler.upsertMembership)
	group.Delete("/:tenantID/memberships/:userID", handler.removeMembership)
	group.Get("/:tenantID/batches", handler.listBatches)
	group.Get("/:tenantID/batches/:batchID", handler.getBatch)
	group.Post("/:tenantID/batches/:batchID/cancel", handler.cancelBatch)
	group.Get("/:tenantID/batches/:batchID/output", handler.downloadBatchOutput)
	group.Get("/:tenantID/batches/:batchID/errors", handler.downloadBatchErrors)
}

type tenantHandler struct {
	container *app.Container
	service   *admintenantsvc.Service
}

type listTenantResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	BudgetLimitUSD   float64   `json:"budget_limit_usd"`
	BudgetUsedUSD    float64   `json:"budget_used_usd"`
	WarningThreshold *float64  `json:"warning_threshold,omitempty"`
}

type listPersonalTenantResponse struct {
	TenantID         string    `json:"tenant_id"`
	UserID           string    `json:"user_id"`
	UserEmail        string    `json:"user_email"`
	UserName         string    `json:"user_name"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	BudgetLimitUSD   float64   `json:"budget_limit_usd"`
	BudgetUsedUSD    float64   `json:"budget_used_usd"`
	WarningThreshold *float64  `json:"warning_threshold,omitempty"`
	MembershipCount  int64     `json:"membership_count"`
}

type createTenantRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type updateTenantStatusRequest struct {
	Status string `json:"status"`
}

type updateTenantDetailsRequest struct {
	Name string `json:"name"`
}

type createAPIKeyRequest struct {
	Name   string        `json:"name"`
	Scopes []string      `json:"scopes"`
	Quota  *quotaPayload `json:"quota"`
}

type quotaPayload struct {
	BudgetUSD        float64 `json:"budget_usd,omitempty"`
	BudgetCents      int64   `json:"budget_cents,omitempty"`
	WarningThreshold float64 `json:"warning_threshold,omitempty"`
}

type membershipRequest struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

type tenantModelsRequest struct {
	Models []string `json:"models"`
}

type membershipResponse struct {
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *tenantHandler) list(c *fiber.Ctx) error {
	limit := int32(50)
	offset := int32(0)

	if val := strings.TrimSpace(c.Query("limit")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	if val := strings.TrimSpace(c.Query("offset")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	items, err := h.service.List(c.Context(), limit, offset)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]listTenantResponse, 0, len(items))
	for _, item := range items {
		out = append(out, listTenantResponse{
			ID:               item.ID.String(),
			Name:             item.Name,
			Status:           string(item.Status),
			CreatedAt:        item.CreatedAt,
			BudgetLimitUSD:   item.BudgetLimitUSD,
			BudgetUsedUSD:    item.BudgetUsedUSD,
			WarningThreshold: item.WarningThresh,
		})
	}

	return c.JSON(fiber.Map{
		"tenants": out,
		"limit":   limit,
		"offset":  offset,
	})
}

func (h *tenantHandler) listPersonal(c *fiber.Ctx) error {
	limit := int32(50)
	offset := int32(0)

	if val := strings.TrimSpace(c.Query("limit")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	if val := strings.TrimSpace(c.Query("offset")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	items, err := h.service.ListPersonal(c.Context(), limit, offset)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]listPersonalTenantResponse, 0, len(items))
	for _, item := range items {
		out = append(out, listPersonalTenantResponse{
			TenantID:         item.TenantID.String(),
			UserID:           item.UserID.String(),
			UserEmail:        item.UserEmail,
			UserName:         item.UserName,
			Status:           string(item.Status),
			CreatedAt:        item.CreatedAt,
			BudgetLimitUSD:   item.BudgetLimitUSD,
			BudgetUsedUSD:    item.BudgetUsedUSD,
			WarningThreshold: item.WarningThresh,
			MembershipCount:  item.MembershipCount,
		})
	}

	return c.JSON(fiber.Map{
		"personal_tenants": out,
		"limit":            limit,
		"offset":           offset,
	})
}

func (h *tenantHandler) create(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}

	var req createTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Name = strings.TrimSpace(req.Name)
	status := strings.TrimSpace(req.Status)

	if req.Name == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "name is required")
	}
	if status == "" {
		status = string(db.TenantStatusActive)
	}
	if status != string(db.TenantStatusActive) && status != string(db.TenantStatusSuspended) {
		return httputil.WriteError(c, fiber.StatusBadRequest, "status must be active or suspended")
	}

	record, err := h.service.CreateTenant(c.Context(), req.Name, db.TenantStatus(status))
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	created, err := timeFromPg(record.CreatedAt)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "invalid tenant created_at")
	}

	tenantID, err := fromPgUUID(record.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "invalid tenant id")
	}

	response := listTenantResponse{
		ID:             tenantID.String(),
		Name:           record.Name,
		Status:         string(record.Status),
		CreatedAt:      created,
		BudgetLimitUSD: h.container.Config.Budgets.DefaultUSD,
		BudgetUsedUSD:  0,
	}

	if err := recordAudit(c, h.container, "tenant.create", "tenant", response.ID, fiber.Map{
		"name":   response.Name,
		"status": response.Status,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

func (h *tenantHandler) updateDetails(c *fiber.Ctx) error {
	tenantUUID, err := parseTenantParam(c)
	if err != nil {
		return err
	}

	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleOwner); err != nil {
		return err
	}

	var req updateTenantDetailsRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "name is required")
	}
	if len(name) > 128 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "name must be <= 128 characters")
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	record, err := h.service.UpdateTenantName(c.Context(), tenantUUID, name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return httputil.WriteError(c, fiber.StatusBadRequest, "tenant name already exists")
		}
		return writeTenantServiceError(c, err)
	}

	created, err := timeFromPg(record.CreatedAt)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "invalid tenant created_at")
	}

	tenantIDOut, err := fromPgUUID(record.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "invalid tenant id")
	}

	resp := listTenantResponse{
		ID:             tenantIDOut.String(),
		Name:           record.Name,
		Status:         string(record.Status),
		CreatedAt:      created,
		BudgetLimitUSD: h.container.Config.Budgets.DefaultUSD,
		BudgetUsedUSD:  0,
	}

	if err := recordAudit(c, h.container, "tenant.update_name", "tenant", tenantIDOut.String(), fiber.Map{
		"name": record.Name,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(resp)
}

func (h *tenantHandler) updateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	if err := requireTenantRole(c, h.container, id, db.MembershipRoleOwner); err != nil {
		return err
	}

	var req updateTenantStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}

	status := strings.TrimSpace(req.Status)
	if status != string(db.TenantStatusActive) && status != string(db.TenantStatusSuspended) {
		return httputil.WriteError(c, fiber.StatusBadRequest, "status must be active or suspended")
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	record, err := h.service.UpdateTenantStatus(c.Context(), id, db.TenantStatus(status))
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	created, err := timeFromPg(record.CreatedAt)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "invalid tenant created_at")
	}

	tenantIDOut, err := fromPgUUID(record.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "invalid tenant id")
	}

	response := listTenantResponse{
		ID:        tenantIDOut.String(),
		Name:      record.Name,
		Status:    string(record.Status),
		CreatedAt: created,
	}

	if err := recordAudit(c, h.container, "tenant.update_status", "tenant", response.ID, fiber.Map{
		"status": response.Status,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(response)
}

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

	return c.JSON(mapBudgetOverride(override))
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

	if err := recordAudit(c, h.container, "tenant.models.set", "tenant", tenantUUID.String(), fiber.Map{
		"models": finalList,
	}); err != nil {
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

	if err := recordAudit(c, h.container, "tenant.models.clear", "tenant", tenantUUID.String(), fiber.Map{}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *tenantHandler) listAPIKeys(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	if err := requireTenantRole(c, h.container, id, db.MembershipRoleViewer); err != nil {
		return err
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	keys, err := h.service.ListAPIKeys(c.Context(), id)
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	tenantName := h.lookupTenantName(c.Context(), id)
	responses := make([]apiKeyResponse, 0, len(keys))
	for _, key := range keys {
		issuer := resolveAPIKeyIssuer(key, tenantName, "", "")
		resp, err := buildAPIKeyResponse(c.Context(), h.container, key, id, tenantName, issuer)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
		responses = append(responses, resp)
	}

	return c.JSON(fiber.Map{
		"api_keys": responses,
	})
}

func (h *tenantHandler) createAPIKey(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleAdmin); err != nil {
		return err
	}

	var req createAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "name is required")
	}

	var scopesJSON []byte
	if len(req.Scopes) > 0 {
		var err error
		scopesJSON, err = json.Marshal(req.Scopes)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid scopes")
		}
	} else {
		scopesJSON = []byte("[]")
	}

	var quotaJSON []byte
	if req.Quota != nil {
		var err error
		quotaJSON, err = json.Marshal(req.Quota)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid quota")
		}
	} else {
		quotaJSON = []byte("{}")
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	result, err := h.service.CreateAPIKey(c.Context(), tenantID, req.Name, scopesJSON, quotaJSON)
	if err != nil {
		return writeTenantServiceError(c, err)
	}
	tenantName := h.lookupTenantName(c.Context(), tenantID)
	issuer := resolveAPIKeyIssuer(result.Key, tenantName, "", "")
	response, err := buildAPIKeyResponse(c.Context(), h.container, result.Key, tenantID, tenantName, issuer)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := recordAudit(c, h.container, "api_key.create", "api_key", response.ID, fiber.Map{
		"tenant_id": tenantID.String(),
		"prefix":    response.Prefix,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(createAPIKeyResponse{
		APIKeyResponse: response,
		Secret:         result.Secret,
		Token:          result.Token,
	})
}

func (h *tenantHandler) revokeAPIKey(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	apiKeyID, err := uuid.Parse(strings.TrimSpace(c.Params("apiKeyID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid api key id")
	}

	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleAdmin); err != nil {
		return err
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}

	record, err := h.service.RevokeAPIKey(c.Context(), tenantID, apiKeyID)
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	tenantName := h.lookupTenantName(c.Context(), tenantID)
	issuer := resolveAPIKeyIssuer(record, tenantName, "", "")
	response, err := buildAPIKeyResponse(c.Context(), h.container, record, tenantID, tenantName, issuer)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	response.Revoked = true

	if err := recordAudit(c, h.container, "api_key.revoke", "api_key", response.ID, fiber.Map{
		"tenant_id": tenantID.String(),
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(response)
}

func (h *tenantHandler) listMemberships(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleAdmin); err != nil {
		return err
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	members, err := h.service.ListMemberships(c.Context(), tenantID)
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	out := make([]membershipResponse, 0, len(members))
	for _, member := range members {
		out = append(out, membershipResponse{
			TenantID:  member.TenantID.String(),
			UserID:    member.UserID.String(),
			Email:     member.Email,
			Role:      string(member.Role),
			CreatedAt: member.Created,
		})
	}

	return c.JSON(fiber.Map{"memberships": out})
}

func (h *tenantHandler) upsertMembership(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleOwner); err != nil {
		return err
	}

	var req membershipRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "email is required")
	}
	role, ok := rbac.ParseRole(req.Role)
	if !ok {
		return httputil.WriteError(c, fiber.StatusBadRequest, "role must be owner, admin, viewer, or user")
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	result, err := h.service.UpsertMembership(c.Context(), tenantID, req.Email, role, strings.TrimSpace(req.Password))
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	resp := membershipResponse{
		TenantID:  result.TenantID.String(),
		UserID:    result.UserID.String(),
		Email:     result.Email,
		Role:      string(result.Role),
		CreatedAt: result.Created,
	}

	if err := recordAudit(c, h.container, "membership.upsert", "tenant", tenantID.String(), fiber.Map{
		"user_id": resp.UserID,
		"role":    resp.Role,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *tenantHandler) removeMembership(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	userID, err := uuid.Parse(strings.TrimSpace(c.Params("userID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid user id")
	}

	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleOwner); err != nil {
		return err
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	if err := h.service.RemoveMembership(c.Context(), tenantID, userID); err != nil {
		return writeTenantServiceError(c, err)
	}

	if err := recordAudit(c, h.container, "membership.remove", "tenant", tenantID.String(), fiber.Map{
		"user_id": userID.String(),
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *tenantHandler) lookupTenantName(ctx context.Context, tenantID uuid.UUID) string {
	if h.container == nil || h.container.Queries == nil {
		return ""
	}
	record, err := h.container.Queries.GetTenantByID(ctx, toPgUUID(tenantID))
	if err != nil {
		return ""
	}
	return record.Name
}

func (h *tenantHandler) listBatches(c *fiber.Ctx) error {
	tenantUUID, err := uuid.Parse(c.Params("tenantID"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches service unavailable")
	}

	limit, offset := parseBatchPagination(c)
	records, err := h.container.Batches.List(c.UserContext(), tenantUUID, limit, offset)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]batchdto.Batch, 0, len(records))
	for _, record := range records {
		out = append(out, batchdto.FromBatch(record))
	}
	return c.JSON(fiber.Map{
		"object": "list",
		"data":   out,
	})
}

func (h *tenantHandler) getBatch(c *fiber.Ctx) error {
	tenantUUID, batchUUID, ok := h.parseBatchRouteParams(c)
	if !ok {
		return nil
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches service unavailable")
	}
	record, err := h.container.Batches.Get(c.UserContext(), tenantUUID, batchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "batch not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(batchdto.FromBatch(record))
}

func (h *tenantHandler) cancelBatch(c *fiber.Ctx) error {
	tenantUUID, batchUUID, ok := h.parseBatchRouteParams(c)
	if !ok {
		return nil
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches service unavailable")
	}
	record, err := h.container.Batches.Cancel(c.UserContext(), tenantUUID, batchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "batch not found or cannot transition")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(batchdto.FromBatch(record))
}

func (h *tenantHandler) downloadBatchOutput(c *fiber.Ctx) error {
	return h.streamBatchFile(c, true)
}

func (h *tenantHandler) downloadBatchErrors(c *fiber.Ctx) error {
	return h.streamBatchFile(c, false)
}

func (h *tenantHandler) streamBatchFile(c *fiber.Ctx, output bool) error {
	tenantUUID, batchUUID, ok := h.parseBatchRouteParams(c)
	if !ok {
		return nil
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.container.Batches == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batch files unavailable")
	}

	batch, err := h.container.Batches.Get(c.UserContext(), tenantUUID, batchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "batch not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	var fileID *uuid.UUID
	filenameSuffix := "output"
	if output {
		fileID = batch.ResultFileID
	} else {
		fileID = batch.ErrorFileID
		filenameSuffix = "errors"
	}
	if fileID == nil {
		return httputil.WriteError(c, fiber.StatusNotFound, fmt.Sprintf("batch %s file not available", filenameSuffix))
	}

	reader, fileRec, err := h.container.Files.Open(c.UserContext(), tenantUUID, *fileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "file not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer reader.Close()

	contentType := fileRec.ContentType
	if contentType == "" {
		contentType = "application/x-ndjson"
	}
	c.Set("Content-Type", contentType)
	if fileRec.Bytes > 0 {
		c.Set("Content-Length", strconv.FormatInt(fileRec.Bytes, 10))
	}
	filename := fileRec.Filename
	if filename == "" {
		filename = fmt.Sprintf("batch_%s_%s.jsonl", batchUUID.String(), filenameSuffix)
	}
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Set("Cache-Control", "no-store")
	return c.SendStream(reader)
}

func (h *tenantHandler) parseBatchRouteParams(c *fiber.Ctx) (uuid.UUID, uuid.UUID, bool) {
	tenantUUID, err := uuid.Parse(c.Params("tenantID"))
	if err != nil {
		_ = httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	batchUUID, err := uuid.Parse(c.Params("batchID"))
	if err != nil {
		_ = httputil.WriteError(c, fiber.StatusBadRequest, "invalid batch id")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return tenantUUID, batchUUID, true
}

func parseBatchPagination(c *fiber.Ctx) (int32, int32) {
	limit := int32(50)
	offset := int32(0)
	if val := strings.TrimSpace(c.Query("limit")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	if val := strings.TrimSpace(c.Query("offset")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}
	return limit, offset
}

func writeTenantServiceError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, admintenantsvc.ErrInvalidModelList),
		errors.Is(err, admintenantsvc.ErrModelNotFound),
		errors.Is(err, admintenantsvc.ErrLocalAuthDisabled):
		status = fiber.StatusBadRequest
	case errors.Is(err, admintenantsvc.ErrAPIKeyTenantMismatch):
		status = fiber.StatusNotFound
	case errors.Is(err, admintenantsvc.ErrServiceUnavailable):
		status = fiber.StatusInternalServerError
	}
	return httputil.WriteError(c, status, err.Error())
}
