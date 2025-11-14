package admin

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	usageservice "github.com/ncecere/open_model_gateway/backend/internal/services/usage"
)

type usageHandler struct {
	container *app.Container
	service   *usageservice.Service
}

func registerAdminUsageRoutes(router fiber.Router, container *app.Container) {
	handler := &usageHandler{
		container: container,
		service:   container.UsageService,
	}

	group := router.Group("/usage")
	group.Get("/summary", handler.summary)
	group.Get("/breakdown", handler.breakdown)
	group.Get("/compare", handler.compare)
	group.Get("/tenant/daily", handler.tenantDaily)
	group.Get("/user/daily", handler.userDaily)
	group.Get("/model/daily", handler.modelDaily)
}

func (h *usageHandler) summary(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "usage service unavailable")
	}

	period := strings.TrimSpace(c.Query("period"))
	if period == "" {
		period = "7d"
	}
	timezone := strings.TrimSpace(c.Query("timezone"))
	startPtr, endPtr, err := parseRangeParams(c.Query("start"), c.Query("end"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	var tenantPtr *uuid.UUID
	tenantIDParam := strings.TrimSpace(c.Query("tenant_id"))
	if tenantIDParam != "" {
		tenantUUID, err := uuid.Parse(tenantIDParam)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant_id")
		}
		if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleViewer); err != nil {
			return err
		}
		tenantPtr = &tenantUUID
	}

	summary, err := h.service.SummarizeAdminUsage(c.Context(), period, tenantPtr, timezone, startPtr, endPtr)
	if err != nil {
		switch {
		case errors.Is(err, usageservice.ErrInvalidPeriod):
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, usageservice.ErrInvalidTimezone):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid timezone")
		case errors.Is(err, usageservice.ErrInvalidRange):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid date range")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(summary)
}

func (h *usageHandler) breakdown(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "usage service unavailable")
	}

	group := strings.ToLower(strings.TrimSpace(c.Query("group")))
	if group == "" {
		group = "tenant"
	}
	limit := parsePositiveInt(c.Query("limit"), 5)
	period := strings.TrimSpace(c.Query("period"))
	if period == "" {
		period = "30d"
	}
	selectedEntity := strings.TrimSpace(c.Query("entity_id"))
	timezone := strings.TrimSpace(c.Query("timezone"))
	startPtr, endPtr, err := parseRangeParams(c.Query("start"), c.Query("end"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.BreakdownAdminUsage(c.Context(), usageservice.AdminBreakdownParams{
		Group:         group,
		Period:        period,
		Limit:         limit,
		EntityID:      selectedEntity,
		Timezone:      timezone,
		StartOverride: startPtr,
		EndOverride:   endPtr,
	})
	if err != nil {
		switch {
		case errors.Is(err, usageservice.ErrInvalidPeriod):
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, usageservice.ErrInvalidTimezone):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid timezone")
		case errors.Is(err, usageservice.ErrInvalidRange):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid date range")
		case errors.Is(err, usageservice.ErrInvalidBreakdownType):
			return httputil.WriteError(c, fiber.StatusBadRequest, "group must be tenant, model, or user")
		default:
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(result)
}

func (h *usageHandler) compare(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "usage service unavailable")
	}
	period := strings.TrimSpace(c.Query("period"))
	if period == "" {
		period = "30d"
	}
	timezone := strings.TrimSpace(c.Query("timezone"))
	startPtr, endPtr, err := parseRangeParams(c.Query("start"), c.Query("end"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	tenantIDs, err := parseUUIDListParam(c.Query("tenant_ids"), "tenant_id")
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	for _, id := range tenantIDs {
		if err := requireTenantRole(c, h.container, id, db.MembershipRoleViewer); err != nil {
			return err
		}
	}
	userIDs, err := parseUUIDListParam(c.Query("user_ids"), "user_id")
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	aliases := parseAliasList(c.Query("model_aliases"))
	if len(tenantIDs) == 0 && len(aliases) == 0 && len(userIDs) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "tenant_ids, user_ids, or model_aliases required")
	}
	result, err := h.service.CompareUsage(c.Context(), usageservice.CompareUsageParams{
		Period:       period,
		Timezone:     timezone,
		Start:        startPtr,
		End:          endPtr,
		TenantIDs:    tenantIDs,
		ModelAliases: aliases,
		UserIDs:      userIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, usageservice.ErrInvalidPeriod):
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, usageservice.ErrInvalidRange):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid start/end range")
		case errors.Is(err, usageservice.ErrInvalidTimezone):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid timezone")
		case errors.Is(err, usageservice.ErrNoEntitiesSelected):
			return httputil.WriteError(c, fiber.StatusBadRequest, "tenant_ids, user_ids, or model_aliases required")
		case errors.Is(err, usageservice.ErrEntityLimitExceeded):
			return httputil.WriteError(c, fiber.StatusBadRequest, "too many entities requested")
		default:
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(result)
}

func (h *usageHandler) tenantDaily(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "usage service unavailable")
	}
	tenantIDStr := strings.TrimSpace(c.Query("tenant_id"))
	if tenantIDStr == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "tenant_id required")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant_id")
	}
	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleViewer); err != nil {
		return err
	}
	startPtr, endPtr, err := parseRangeParams(c.Query("start"), c.Query("end"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	if startPtr == nil || endPtr == nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "start and end parameters are required")
	}
	timezone := strings.TrimSpace(c.Query("timezone"))
	result, err := h.service.TenantDailyUsage(c.Context(), tenantID, *startPtr, *endPtr, timezone)
	if err != nil {
		switch {
		case errors.Is(err, usageservice.ErrInvalidRange):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid range")
		case errors.Is(err, usageservice.ErrInvalidTimezone):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid timezone")
		default:
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(result)
}

func (h *usageHandler) userDaily(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "usage service unavailable")
	}
	userIDStr := strings.TrimSpace(c.Query("user_id"))
	if userIDStr == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "user_id required")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid user_id")
	}
	startPtr, endPtr, err := parseRangeParams(c.Query("start"), c.Query("end"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	if startPtr == nil || endPtr == nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "start and end parameters are required")
	}
	timezone := strings.TrimSpace(c.Query("timezone"))
	result, err := h.service.UserDailyUsage(c.Context(), userID, *startPtr, *endPtr, timezone)
	if err != nil {
		switch {
		case errors.Is(err, usageservice.ErrInvalidRange):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid range")
		case errors.Is(err, usageservice.ErrInvalidTimezone):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid timezone")
		default:
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(result)
}

func (h *usageHandler) modelDaily(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "usage service unavailable")
	}
	alias := strings.TrimSpace(c.Query("model_alias"))
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model_alias required")
	}
	startPtr, endPtr, err := parseRangeParams(c.Query("start"), c.Query("end"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	if startPtr == nil || endPtr == nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "start and end parameters are required")
	}
	timezone := strings.TrimSpace(c.Query("timezone"))
	result, err := h.service.ModelDailyUsage(c.Context(), alias, *startPtr, *endPtr, timezone, nil)
	if err != nil {
		switch {
		case errors.Is(err, usageservice.ErrInvalidRange):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid range")
		case errors.Is(err, usageservice.ErrInvalidTimezone):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid timezone")
		default:
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(result)
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	if value, err := strconv.Atoi(raw); err == nil && value > 0 {
		return value
	}
	return fallback
}

func parseUUIDListParam(raw string, field string) ([]uuid.UUID, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, nil
	}
	parts := strings.Split(clean, ",")
	values := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		id, err := uuid.Parse(candidate)
		if err != nil {
			return nil, fmt.Errorf("invalid %s %q", field, candidate)
		}
		values = append(values, id)
	}
	return values, nil
}

func parseAliasList(raw string) []string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil
	}
	parts := strings.Split(clean, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		alias := strings.TrimSpace(part)
		if alias == "" {
			continue
		}
		values = append(values, alias)
	}
	return values
}

func parseRangeParams(startRaw, endRaw string) (*time.Time, *time.Time, error) {
	startStr := strings.TrimSpace(startRaw)
	endStr := strings.TrimSpace(endRaw)
	if startStr == "" && endStr == "" {
		return nil, nil, nil
	}
	if startStr == "" || endStr == "" {
		return nil, nil, fmt.Errorf("start and end must both be provided")
	}
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid start timestamp")
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid end timestamp")
	}
	return &start, &end, nil
}
