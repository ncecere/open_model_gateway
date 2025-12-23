package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

func (h *tenantHandler) list(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
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
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
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
