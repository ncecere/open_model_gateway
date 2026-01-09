package user

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
)

type tenantModelEntry struct {
	Alias            string `json:"alias"`
	Provider         string `json:"provider"`
	ModelType        string `json:"model_type"`
	Enabled          bool   `json:"enabled"`
	TenantAssignable bool   `json:"tenant_assignable"`
	Attached         bool   `json:"attached"`
}

func (h *userHandler) registerTenantModelRoutes(group fiber.Router) {
	group.Get("/tenants/:tenantID/models",
		h.requireTenantMembership("tenantID"),
		h.listTenantModels,
	)
	group.Post("/tenants/:tenantID/models/:alias",
		h.requireTenantCapability("tenantID", rbac.CapabilityAttachModels),
		h.attachTenantModel,
	)
	group.Delete("/tenants/:tenantID/models/:alias",
		h.requireTenantCapability("tenantID", rbac.CapabilityAttachModels),
		h.detachTenantModel,
	)
}

func (h *userHandler) listTenantModels(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.container == nil || h.container.AdminCatalog == nil || h.container.Data == nil || h.container.Data.Queries == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "model catalog unavailable")
	}

	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	if summary, ok := tenantSummaryFromLocals(c); !ok || summary.TenantID != tenantUUID {
		if _, err := h.checkTenantMembership(c.Context(), user, tenantUUID); err != nil {
			return writeTenantCapabilityError(c, err)
		}
	}

	attachedAliases, err := h.container.Data.Queries.ListTenantModels(c.Context(), toPgUUID(tenantUUID))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	attachedSet := make(map[string]struct{}, len(attachedAliases))
	for _, alias := range attachedAliases {
		attachedSet[strings.ToLower(strings.TrimSpace(alias))] = struct{}{}
	}

	catalog, err := h.container.AdminCatalog.List(c.Context())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	result := make([]tenantModelEntry, 0, len(catalog))
	for _, model := range catalog {
		alias := strings.TrimSpace(model.Alias)
		if alias == "" {
			continue
		}
		_, attached := attachedSet[strings.ToLower(alias)]
		if !model.TenantAssignable && !attached {
			continue
		}
		result = append(result, tenantModelEntry{
			Alias:            alias,
			Provider:         model.Provider,
			ModelType:        model.ModelType,
			Enabled:          model.Enabled,
			TenantAssignable: model.TenantAssignable,
			Attached:         attached,
		})
		delete(attachedSet, strings.ToLower(alias))
	}

	for alias := range attachedSet {
		result = append(result, tenantModelEntry{
			Alias:            alias,
			Enabled:          false,
			TenantAssignable: false,
			Attached:         true,
		})
	}

	return c.JSON(fiber.Map{"models": result})
}

func (h *userHandler) attachTenantModel(c *fiber.Ctx) error {
	_, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.container == nil || h.container.Data == nil || h.container.Data.Queries == nil || h.container.AdminCatalog == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "model catalog unavailable")
	}

	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	alias := strings.TrimSpace(c.Params("alias"))
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "alias required")
	}

	model, err := h.container.Data.Queries.GetModelByAlias(c.Context(), alias)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "model not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !model.TenantAssignable {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not approved for tenants")
	}

	if err := h.container.Data.Queries.InsertTenantModel(c.Context(), db.InsertTenantModelParams{
		TenantID: toPgUUID(tenantUUID),
		Alias:    alias,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := h.refreshTenantModelCache(c.Context(), tenantUUID); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := recordUserAudit(c, h.container, "tenant.models.attach", "tenant", tenantUUID.String(), fiber.Map{
		"alias": alias,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"attached": true})
}

func (h *userHandler) detachTenantModel(c *fiber.Ctx) error {
	_, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.container == nil || h.container.Data == nil || h.container.Data.Queries == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "model catalog unavailable")
	}

	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	alias := strings.TrimSpace(c.Params("alias"))
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "alias required")
	}

	if err := h.container.Data.Queries.DeleteTenantModel(c.Context(), db.DeleteTenantModelParams{
		TenantID: toPgUUID(tenantUUID),
		Alias:    alias,
	}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := h.refreshTenantModelCache(c.Context(), tenantUUID); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := recordUserAudit(c, h.container, "tenant.models.detach", "tenant", tenantUUID.String(), fiber.Map{
		"alias": alias,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *userHandler) refreshTenantModelCache(ctx context.Context, tenantID uuid.UUID) error {
	if h.container == nil || h.container.Data == nil || h.container.Data.Queries == nil {
		return errors.New("tenant model cache unavailable")
	}
	aliases, err := h.container.Data.Queries.ListTenantModels(ctx, toPgUUID(tenantID))
	if err != nil {
		return err
	}
	h.container.SetTenantModels(tenantID, aliases)
	return nil
}
