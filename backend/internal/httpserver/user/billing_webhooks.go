package user

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
	billinghooks "github.com/ncecere/open_model_gateway/backend/internal/services/billinghooks"
	"github.com/ncecere/open_model_gateway/backend/internal/timeutil"
)

type userBillingWebhookCreateRequest struct {
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Secret   string `json:"secret"`
	Enabled  *bool  `json:"enabled"`
}

type userBillingWebhookUpdateRequest struct {
	Name    *string `json:"name"`
	URL     *string `json:"url"`
	Secret  *string `json:"secret"`
	Enabled *bool   `json:"enabled"`
}

type userBillingWebhookDispatchRequest struct {
	Period   string `json:"period"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Timezone string `json:"timezone"`
}

type userBillingWebhookResponse struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Name      string `json:"name,omitempty"`
	URL       string `json:"url"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type userBillingWebhookEventResponse struct {
	ID          string `json:"id"`
	WebhookID   string `json:"webhook_id"`
	TenantID    string `json:"tenant_id"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	Success     bool   `json:"success"`
	StatusCode  *int32 `json:"status_code,omitempty"`
	Error       string `json:"error,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func (h *userHandler) registerBillingWebhookRoutes(group fiber.Router) {
	group.Get("/billing-webhooks", h.listBillingWebhooks)
	group.Post("/billing-webhooks", h.createBillingWebhook)
	group.Put("/billing-webhooks/:id", h.updateBillingWebhook)
	group.Delete("/billing-webhooks/:id", h.deleteBillingWebhook)
	group.Get("/billing-webhooks/:id/events", h.listBillingWebhookEvents)
	group.Post("/billing-webhooks/:id/dispatch", h.dispatchBillingWebhook)
}

func (h *userHandler) listBillingWebhooks(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	tenantID, err := parseTenantScope(c.Query("tenant_id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	if _, err := h.checkTenantCapability(c.Context(), user, tenantID, rbac.CapabilityManageBillingWebhooks); err != nil {
		return writeTenantCapabilityError(c, err)
	}
	webhooks, err := h.container.BillingWebhooks.ListByTenant(c.Context(), tenantID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"webhooks": mapUserBillingWebhooks(webhooks)})
}

func (h *userHandler) createBillingWebhook(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	var req userBillingWebhookCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	tenantID, err := parseTenantScope(req.TenantID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	if _, err := h.checkTenantCapability(c.Context(), user, tenantID, rbac.CapabilityManageBillingWebhooks); err != nil {
		return writeTenantCapabilityError(c, err)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	webhook, err := h.container.BillingWebhooks.Create(c.Context(), billinghooks.CreateParams{
		TenantID: tenantID,
		Name:     strings.TrimSpace(req.Name),
		URL:      strings.TrimSpace(req.URL),
		Secret:   strings.TrimSpace(req.Secret),
		Enabled:  enabled,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	if err := recordUserAudit(c, h.container, "billing_webhook.create", "billing_webhook", webhook.ID.String(), fiber.Map{
		"tenant_id": webhook.TenantID.String(),
		"name":      webhook.Name,
		"url":       webhook.URL,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(mapUserBillingWebhook(webhook))
}

func (h *userHandler) updateBillingWebhook(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid webhook id")
	}
	current, err := h.container.BillingWebhooks.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "webhook not found")
	}
	if _, err := h.checkTenantCapability(c.Context(), user, current.TenantID, rbac.CapabilityManageBillingWebhooks); err != nil {
		return writeTenantCapabilityError(c, err)
	}
	var req userBillingWebhookUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	name := current.Name
	url := current.URL
	secret := current.Secret
	enabled := current.Enabled
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.URL != nil {
		url = strings.TrimSpace(*req.URL)
	}
	if req.Secret != nil {
		secret = strings.TrimSpace(*req.Secret)
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	updated, err := h.container.BillingWebhooks.Update(c.Context(), id, billinghooks.UpdateParams{
		Name:    name,
		URL:     url,
		Secret:  secret,
		Enabled: enabled,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	if err := recordUserAudit(c, h.container, "billing_webhook.update", "billing_webhook", updated.ID.String(), fiber.Map{
		"tenant_id": updated.TenantID.String(),
		"name":      updated.Name,
		"url":       updated.URL,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(mapUserBillingWebhook(updated))
}

func (h *userHandler) deleteBillingWebhook(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid webhook id")
	}
	current, err := h.container.BillingWebhooks.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "webhook not found")
	}
	if _, err := h.checkTenantCapability(c.Context(), user, current.TenantID, rbac.CapabilityManageBillingWebhooks); err != nil {
		return writeTenantCapabilityError(c, err)
	}
	if err := h.container.BillingWebhooks.Delete(c.Context(), id); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if err := recordUserAudit(c, h.container, "billing_webhook.delete", "billing_webhook", current.ID.String(), fiber.Map{
		"tenant_id": current.TenantID.String(),
		"name":      current.Name,
		"url":       current.URL,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *userHandler) listBillingWebhookEvents(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid webhook id")
	}
	current, err := h.container.BillingWebhooks.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "webhook not found")
	}
	if _, err := h.checkTenantCapability(c.Context(), user, current.TenantID, rbac.CapabilityManageBillingWebhooks); err != nil {
		return writeTenantCapabilityError(c, err)
	}
	limit := int32(parsePositiveInt(c.Query("limit"), 25))
	offset := int32(parsePositiveInt(c.Query("offset"), 0))
	events, err := h.container.BillingWebhooks.ListEvents(c.Context(), id, limit, offset)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"events": mapUserBillingWebhookEvents(events)})
}

func (h *userHandler) dispatchBillingWebhook(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid webhook id")
	}
	current, err := h.container.BillingWebhooks.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "webhook not found")
	}
	if _, err := h.checkTenantCapability(c.Context(), user, current.TenantID, rbac.CapabilityManageBillingWebhooks); err != nil {
		return writeTenantCapabilityError(c, err)
	}
	var req userBillingWebhookDispatchRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	startPtr, endPtr, err := parseRangeParams(req.Start, req.End)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	window, err := resolveUserDispatchWindow(req.Period, req.Timezone, startPtr, endPtr, h.container.ReportingLoc())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	events, err := h.container.BillingWebhooks.DispatchSummary(c.Context(), current.TenantID, window.Start(), window.End())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if err := recordUserAudit(c, h.container, "billing_webhook.dispatch", "billing_webhook", current.ID.String(), fiber.Map{
		"tenant_id": current.TenantID.String(),
		"period":    req.Period,
		"timezone":  req.Timezone,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"events": mapUserBillingWebhookEvents(events)})
}

func mapUserBillingWebhooks(webhooks []billinghooks.Webhook) []userBillingWebhookResponse {
	result := make([]userBillingWebhookResponse, 0, len(webhooks))
	for _, webhook := range webhooks {
		result = append(result, mapUserBillingWebhook(webhook))
	}
	return result
}

func mapUserBillingWebhook(webhook billinghooks.Webhook) userBillingWebhookResponse {
	return userBillingWebhookResponse{
		ID:        webhook.ID.String(),
		TenantID:  webhook.TenantID.String(),
		Name:      webhook.Name,
		URL:       webhook.URL,
		Enabled:   webhook.Enabled,
		CreatedAt: webhook.CreatedAt.Format(time.RFC3339),
		UpdatedAt: webhook.UpdatedAt.Format(time.RFC3339),
	}
}

func mapUserBillingWebhookEvents(events []billinghooks.Event) []userBillingWebhookEventResponse {
	result := make([]userBillingWebhookEventResponse, 0, len(events))
	for _, event := range events {
		result = append(result, mapUserBillingWebhookEvent(event))
	}
	return result
}

func mapUserBillingWebhookEvent(event billinghooks.Event) userBillingWebhookEventResponse {
	resp := userBillingWebhookEventResponse{
		ID:          event.ID.String(),
		WebhookID:   event.WebhookID.String(),
		TenantID:    event.TenantID.String(),
		PeriodStart: event.PeriodStart.Format(time.RFC3339),
		PeriodEnd:   event.PeriodEnd.Format(time.RFC3339),
		Success:     event.Success,
		Error:       event.Error,
		CreatedAt:   event.CreatedAt.Format(time.RFC3339),
	}
	if event.StatusCode != nil {
		resp.StatusCode = event.StatusCode
	}
	return resp
}

func parseTenantScope(raw string) (uuid.UUID, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return uuid.Nil, fmt.Errorf("tenant_id required")
	}
	return uuid.Parse(clean)
}

func canManageBillingWebhooks(role db.MembershipRole) bool {
	return rbac.HasCapability(role, rbac.CapabilityManageBillingWebhooks)
}

func resolveUserDispatchWindow(period, timezone string, start, end *time.Time, loc *time.Location) (timeutil.Window, error) {
	loc = timeutil.EnsureLocation(loc)
	if tz := strings.TrimSpace(timezone); tz != "" {
		loaded, err := time.LoadLocation(tz)
		if err != nil {
			return timeutil.Window{}, fmt.Errorf("invalid timezone")
		}
		loc = loaded
	}
	if start != nil || end != nil {
		if start == nil || end == nil {
			return timeutil.Window{}, fmt.Errorf("start and end must both be provided")
		}
		return timeutil.NewWindowFromRange(*start, *end, loc, "custom")
	}
	period = strings.TrimSpace(period)
	if period == "" {
		period = "30d"
	}
	return timeutil.NewWindow(period, time.Now().In(loc), loc)
}
