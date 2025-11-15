package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	guardrailssvc "github.com/ncecere/open_model_gateway/backend/internal/services/guardrails"
)

type guardrailHandler struct {
	container *app.Container
}

type guardrailPayload struct {
	Config map[string]any `json:"config"`
}

func registerAdminGuardrailRoutes(router fiber.Router, container *app.Container) {
	handler := &guardrailHandler{container: container}
	router.Get("/guardrails/events", handler.listEvents)
	router.Get("/tenants/:tenantID/guardrails", handler.getTenantPolicy)
	router.Put("/tenants/:tenantID/guardrails", handler.putTenantPolicy)
	router.Delete("/tenants/:tenantID/guardrails", handler.deleteTenantPolicy)

	router.Get("/tenants/:tenantID/api-keys/:apiKeyID/guardrails", handler.getAPIKeyPolicy)
	router.Put("/tenants/:tenantID/api-keys/:apiKeyID/guardrails", handler.putAPIKeyPolicy)
	router.Delete("/tenants/:tenantID/api-keys/:apiKeyID/guardrails", handler.deleteAPIKeyPolicy)
}

func (h *guardrailHandler) service() *guardrailssvc.Service {
	if h.container == nil {
		return nil
	}
	return h.container.Guardrails
}

func (h *guardrailHandler) getTenantPolicy(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "guardrail service unavailable")
	}
	tenantID, err := uuid.Parse(c.Params("tenantID"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	policy, err := svc.GetTenantPolicy(c.Context(), tenantID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"config": policy.Config})
}

func (h *guardrailHandler) putTenantPolicy(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "guardrail service unavailable")
	}
	tenantID, err := uuid.Parse(c.Params("tenantID"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	var payload guardrailPayload
	if err := c.BodyParser(&payload); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	policy, err := svc.UpsertTenantPolicy(c.Context(), tenantID, payload.Config)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"config": policy.Config})
}

func (h *guardrailHandler) deleteTenantPolicy(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "guardrail service unavailable")
	}
	tenantID, err := uuid.Parse(c.Params("tenantID"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	if err := svc.DeleteTenantPolicy(c.Context(), tenantID); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *guardrailHandler) getAPIKeyPolicy(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "guardrail service unavailable")
	}
	keyID, err := uuid.Parse(c.Params("apiKeyID"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid api key id")
	}
	policy, err := svc.GetAPIKeyPolicy(c.Context(), keyID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"config": policy.Config})
}

func (h *guardrailHandler) putAPIKeyPolicy(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "guardrail service unavailable")
	}
	keyID, err := uuid.Parse(c.Params("apiKeyID"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid api key id")
	}
	var payload guardrailPayload
	if err := c.BodyParser(&payload); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	policy, err := svc.UpsertAPIKeyPolicy(c.Context(), keyID, payload.Config)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"config": policy.Config})
}

func (h *guardrailHandler) deleteAPIKeyPolicy(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "guardrail service unavailable")
	}
	keyID, err := uuid.Parse(c.Params("apiKeyID"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid api key id")
	}
	if err := svc.DeleteAPIKeyPolicy(c.Context(), keyID); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *guardrailHandler) listEvents(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "guardrail service unavailable")
	}
	params := guardrailssvc.ListEventsParams{}
	if tenantParam := strings.TrimSpace(c.Query("tenant_id")); tenantParam != "" {
		tenantID, err := uuid.Parse(tenantParam)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant_id")
		}
		params.TenantID = tenantID
	}
	if apiKeyParam := strings.TrimSpace(c.Query("api_key_id")); apiKeyParam != "" {
		apiKeyID, err := uuid.Parse(apiKeyParam)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid api_key_id")
		}
		params.APIKeyID = apiKeyID
	}
	params.Action = strings.TrimSpace(c.Query("action"))
	params.Stage = strings.TrimSpace(c.Query("stage"))
	params.Category = strings.TrimSpace(c.Query("category"))
	if startStr := strings.TrimSpace(c.Query("start")); startStr != "" {
		start, err := parseTimeParam(startStr)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid start timestamp")
		}
		params.Start = start
	}
	if endStr := strings.TrimSpace(c.Query("end")); endStr != "" {
		end, err := parseTimeParam(endStr)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid end timestamp")
		}
		params.End = end
	}
	params.Limit = int32(parseIntDefault(c.Query("limit"), 50))
	params.Offset = int32(parseIntDefault(c.Query("offset"), 0))
	result, err := svc.ListEvents(c.Context(), params)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(result)
}

func parseIntDefault(value string, fallback int) int {
	v := strings.TrimSpace(value)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	if parsed < 0 {
		return 0
	}
	return parsed
}

func parseTimeParam(value string) (time.Time, error) {
	if strings.Contains(value, "T") {
		return time.Parse(time.RFC3339, value)
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339, value)
}
