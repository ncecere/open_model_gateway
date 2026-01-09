package admin

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/timeutil"
	billinghooks "github.com/ncecere/open_model_gateway/backend/internal/services/billinghooks"
)

type billingWebhookHandler struct {
	container *app.Container
}

type billingWebhookCreateRequest struct {
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Secret   string `json:"secret"`
	Enabled  *bool  `json:"enabled"`
}

type billingWebhookUpdateRequest struct {
	Name    *string `json:"name"`
	URL     *string `json:"url"`
	Secret  *string `json:"secret"`
	Enabled *bool   `json:"enabled"`
}

type billingWebhookDispatchRequest struct {
	Period   string `json:"period"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Timezone string `json:"timezone"`
}

type billingWebhookResponse struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Name      string `json:"name,omitempty"`
	URL       string `json:"url"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type billingWebhookEventResponse struct {
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

func registerAdminBillingWebhookRoutes(router fiber.Router, container *app.Container) {
	handler := &billingWebhookHandler{container: container}
	group := router.Group("/billing-webhooks")
	group.Get("/", handler.list)
	group.Post("/", handler.create)
	group.Put("/:id", handler.update)
	group.Delete("/:id", handler.delete)
	group.Get("/:id/events", handler.listEvents)
	group.Post("/:id/dispatch", handler.dispatch)
	group.Post("/events/:eventID/retry", handler.retry)
}

func (h *billingWebhookHandler) list(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	tenantRaw := strings.TrimSpace(c.Query("tenant_id"))
	if tenantRaw == "" {
		if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
			return err
		}
		webhooks, err := h.container.BillingWebhooks.ListAdmin(c.Context())
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"webhooks": mapBillingWebhooks(webhooks)})
	}
	tenantID, err := uuid.Parse(tenantRaw)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant_id")
	}
	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleViewer); err != nil {
		return err
	}
	webhooks, err := h.container.BillingWebhooks.ListByTenant(c.Context(), tenantID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"webhooks": mapBillingWebhooks(webhooks)})
}

func (h *billingWebhookHandler) create(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	var req billingWebhookCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	tenantID, err := uuid.Parse(strings.TrimSpace(req.TenantID))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant_id")
	}
	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleAdmin); err != nil {
		return err
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
	return c.JSON(mapBillingWebhook(webhook))
}

func (h *billingWebhookHandler) update(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid webhook id")
	}
	current, err := h.container.BillingWebhooks.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "webhook not found")
	}
	if err := requireTenantRole(c, h.container, current.TenantID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	var req billingWebhookUpdateRequest
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
	return c.JSON(mapBillingWebhook(updated))
}

func (h *billingWebhookHandler) delete(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid webhook id")
	}
	current, err := h.container.BillingWebhooks.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "webhook not found")
	}
	if err := requireTenantRole(c, h.container, current.TenantID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if err := h.container.BillingWebhooks.Delete(c.Context(), id); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (h *billingWebhookHandler) listEvents(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid webhook id")
	}
	current, err := h.container.BillingWebhooks.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "webhook not found")
	}
	if err := requireTenantRole(c, h.container, current.TenantID, db.MembershipRoleViewer); err != nil {
		return err
	}
	limit := int32(parsePositiveInt(c.Query("limit"), 25))
	offset := int32(parsePositiveInt(c.Query("offset"), 0))
	events, err := h.container.BillingWebhooks.ListEvents(c.Context(), id, limit, offset)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"events": mapBillingWebhookEvents(events)})
}

func (h *billingWebhookHandler) dispatch(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid webhook id")
	}
	current, err := h.container.BillingWebhooks.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "webhook not found")
	}
	if err := requireTenantRole(c, h.container, current.TenantID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	var req billingWebhookDispatchRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	startPtr, endPtr, err := parseRangeParams(req.Start, req.End)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	window, err := resolveDispatchWindow(req.Period, req.Timezone, startPtr, endPtr, h.container.ReportingLoc())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	events, err := h.container.BillingWebhooks.DispatchSummary(c.Context(), current.TenantID, window.Start(), window.End())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"events": mapBillingWebhookEvents(events)})
}

func (h *billingWebhookHandler) retry(c *fiber.Ctx) error {
	if h.container == nil || h.container.BillingWebhooks == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "billing webhooks unavailable")
	}
	eventID, err := uuid.Parse(c.Params("eventID"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid event id")
	}
	row, err := h.container.Data.Queries.GetBillingWebhookEvent(c.Context(), toPgUUID(eventID))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "event not found")
	}
	tenantID, err := fromPgUUID(row.TenantID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant")
	}
	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	event, err := h.container.BillingWebhooks.RetryEvent(c.Context(), eventID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(mapBillingWebhookEvent(event))
}

func mapBillingWebhooks(webhooks []billinghooks.Webhook) []billingWebhookResponse {
	result := make([]billingWebhookResponse, 0, len(webhooks))
	for _, webhook := range webhooks {
		result = append(result, mapBillingWebhook(webhook))
	}
	return result
}

func mapBillingWebhook(webhook billinghooks.Webhook) billingWebhookResponse {
	return billingWebhookResponse{
		ID:        webhook.ID.String(),
		TenantID:  webhook.TenantID.String(),
		Name:      webhook.Name,
		URL:       webhook.URL,
		Enabled:   webhook.Enabled,
		CreatedAt: webhook.CreatedAt.Format(time.RFC3339),
		UpdatedAt: webhook.UpdatedAt.Format(time.RFC3339),
	}
}

func mapBillingWebhookEvents(events []billinghooks.Event) []billingWebhookEventResponse {
	result := make([]billingWebhookEventResponse, 0, len(events))
	for _, event := range events {
		result = append(result, mapBillingWebhookEvent(event))
	}
	return result
}

func mapBillingWebhookEvent(event billinghooks.Event) billingWebhookEventResponse {
	resp := billingWebhookEventResponse{
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

func resolveDispatchWindow(period, timezone string, start, end *time.Time, loc *time.Location) (timeutil.Window, error) {
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
