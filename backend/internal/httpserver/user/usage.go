package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	usageservice "github.com/ncecere/open_model_gateway/backend/internal/services/usage"
)

func (h *userHandler) registerUsageRoutes(group fiber.Router) {
	group.Get("/usage", h.userUsage)
	group.Get("/usage/compare", h.userUsageCompare)
	group.Get("/usage/tenant/daily", h.userTenantDaily)
	group.Get("/usage/model/daily", h.userModelDaily)
	group.Get("/dashboard", h.userDashboard)
}

func (h *userHandler) userUsage(c *fiber.Ctx) error {
	return h.handleUsageRequest(c, strings.TrimSpace(c.Query("period")), "30d")
}

func (h *userHandler) userDashboard(c *fiber.Ctx) error {
	return h.handleUsageRequest(c, strings.TrimSpace(c.Query("period")), "7d")
}

func (h *userHandler) handleUsageRequest(c *fiber.Ctx, period, fallback string) error {
	if period == "" {
		period = fallback
	}
	startPtr, endPtr, err := parseRangeParams(c.Query("start"), c.Query("end"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	scopeValue := strings.TrimSpace(c.Query("scope"))
	timezone := strings.TrimSpace(c.Query("timezone"))
	var tenantFilter *uuid.UUID
	if scopeValue != "" && scopeValue != "personal" {
		tenantID, err := uuid.Parse(scopeValue)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid scope")
		}
		tenantFilter = &tenantID
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.usage == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "usage service unavailable")
	}
	summary, err := h.usage.SummarizeUserUsage(c.Context(), user, period, tenantFilter, timezone, startPtr, endPtr)
	if err != nil {
		switch {
		case errors.Is(err, usageservice.ErrInvalidPeriod):
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, usageservice.ErrInvalidTimezone):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid timezone")
		case errors.Is(err, usageservice.ErrInvalidRange):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid range")
		}
		if errors.Is(err, usageservice.ErrScopeForbidden) {
			return httputil.WriteError(c, fiber.StatusForbidden, "scope unavailable")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(summary)
}

func (h *userHandler) userUsageCompare(c *fiber.Ctx) error {
	period := strings.TrimSpace(c.Query("period"))
	if period == "" {
		period = "30d"
	}
	timezone := strings.TrimSpace(c.Query("timezone"))
	startPtr, endPtr, err := parseRangeParams(c.Query("start"), c.Query("end"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	tenantIDsParam, err := parseUUIDListParam(c.Query("tenant_ids"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	aliases := parseAliasList(c.Query("model_aliases"))
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	orderedScope, allowed, err := h.loadTenantScope(c.Context(), user)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if len(allowed) == 0 {
		return httputil.WriteError(c, fiber.StatusForbidden, "no tenant access available")
	}
	var tenantIDs []uuid.UUID
	if len(tenantIDsParam) > 0 {
		for _, id := range tenantIDsParam {
			if _, ok := allowed[id]; ok {
				tenantIDs = append(tenantIDs, id)
			}
		}
		if len(tenantIDs) == 0 && len(aliases) == 0 {
			return httputil.WriteError(c, fiber.StatusForbidden, "requested tenants unavailable")
		}
	} else if len(aliases) == 0 {
		tenantIDs = append(tenantIDs, orderedScope...)
	}
	if len(tenantIDs) == 0 && len(aliases) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "tenant_ids or model_aliases required")
	}
	result, err := h.usage.CompareUsage(c.Context(), usageservice.CompareUsageParams{
		Period:       period,
		Timezone:     timezone,
		Start:        startPtr,
		End:          endPtr,
		TenantIDs:    tenantIDs,
		ModelAliases: aliases,
		TenantScope:  orderedScope,
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
			return httputil.WriteError(c, fiber.StatusBadRequest, "tenant_ids or model_aliases required")
		case errors.Is(err, usageservice.ErrEntityLimitExceeded):
			return httputil.WriteError(c, fiber.StatusBadRequest, "too many entities requested")
		default:
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(result)
}

func (h *userHandler) userTenantDaily(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.usage == nil {
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
	startPtr, endPtr, err := parseRangeParams(c.Query("start"), c.Query("end"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	if startPtr == nil || endPtr == nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "start and end parameters are required")
	}
	_, allowed, err := h.loadTenantScope(c.Context(), user)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if _, ok := allowed[tenantID]; !ok {
		return httputil.WriteError(c, fiber.StatusForbidden, "tenant unavailable")
	}
	timezone := strings.TrimSpace(c.Query("timezone"))
	result, err := h.usage.TenantDailyUsage(c.Context(), tenantID, *startPtr, *endPtr, timezone)
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

func (h *userHandler) userModelDaily(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.usage == nil {
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
	scopeOrder, _, err := h.loadTenantScope(c.Context(), user)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if len(scopeOrder) == 0 {
		return httputil.WriteError(c, fiber.StatusForbidden, "no tenant access available")
	}
	timezone := strings.TrimSpace(c.Query("timezone"))
	result, err := h.usage.ModelDailyUsage(c.Context(), alias, *startPtr, *endPtr, timezone, scopeOrder)
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

func parseUUIDListParam(raw string) ([]uuid.UUID, error) {
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
			return nil, fmt.Errorf("invalid tenant_id %q", candidate)
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

func (h *userHandler) loadTenantScope(ctx context.Context, user db.User) ([]uuid.UUID, map[uuid.UUID]struct{}, error) {
	addAllowed := func(id uuid.UUID, allowed map[uuid.UUID]struct{}, ordered *[]uuid.UUID) {
		if id == uuid.Nil {
			return
		}
		if _, exists := allowed[id]; exists {
			return
		}
		allowed[id] = struct{}{}
		*ordered = append(*ordered, id)
	}
	orderedScope := make([]uuid.UUID, 0)
	allowed := make(map[uuid.UUID]struct{})
	if personal, err := uuidFromPg(user.PersonalTenantID); err == nil {
		addAllowed(personal, allowed, &orderedScope)
	}
	memberships, err := h.container.Queries.ListUserTenants(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, row := range memberships {
		tenantID, err := uuidFromPg(row.TenantID)
		if err != nil {
			continue
		}
		addAllowed(tenantID, allowed, &orderedScope)
	}
	return orderedScope, allowed, nil
}
