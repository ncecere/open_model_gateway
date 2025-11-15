package admin

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
	admincatalogsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admincatalog"
)

func registerAdminModelCatalogRoutes(router fiber.Router, container *app.Container) {
	handler := &modelCatalogHandler{
		container: container,
		service:   container.AdminCatalog,
	}

	group := router.Group("/model-catalog")
	group.Get("/", handler.list)
	group.Get("/status", handler.status)
	group.Post("/", handler.upsert)
	group.Delete("/:alias", handler.remove)
}

type modelCatalogHandler struct {
	container *app.Container
	service   *admincatalogsvc.Service
}

func (h *modelCatalogHandler) list(c *fiber.Ctx) error {
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "model catalog service unavailable")
	}
	items, err := h.service.List(c.Context())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(items)
}

func (h *modelCatalogHandler) status(c *fiber.Ctx) error {
	if h.service == nil || h.container == nil || h.container.Engine == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "model catalog service unavailable")
	}
	models, err := h.service.List(c.Context())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	health := h.container.Engine.HealthStatus()
	resp := make([]fiber.Map, 0, len(models))
	for _, model := range models {
		status := deriveModelStatus(model.Enabled, health[model.Alias])
		resp = append(resp, fiber.Map{
			"alias":  model.Alias,
			"status": status,
		})
	}
	return c.JSON(fiber.Map{"statuses": resp})
}

func (h *modelCatalogHandler) upsert(c *fiber.Ctx) error {
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "model catalog service unavailable")
	}
	var payload admincatalogsvc.ModelPayload
	if err := c.BodyParser(&payload); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	entry, err := h.service.Upsert(c.Context(), payload)
	if err != nil {
		return writeCatalogError(c, err)
	}

	if err := recordAudit(c, h.container, "model_catalog.upsert", "model", entry.Alias, fiber.Map{
		"provider":       entry.Provider,
		"provider_model": entry.ProviderModel,
		"deployment":     entry.Deployment,
		"enabled":        entry.Enabled,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(entry)
}

func (h *modelCatalogHandler) remove(c *fiber.Ctx) error {
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "model catalog service unavailable")
	}
	alias := strings.TrimSpace(c.Params("alias"))
	if err := h.service.Remove(c.Context(), alias); err != nil {
		return writeCatalogError(c, err)
	}
	if err := recordAudit(c, h.container, "model_catalog.remove", "model", alias, fiber.Map{}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func writeCatalogError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, admincatalogsvc.ErrAliasRequired),
		errors.Is(err, admincatalogsvc.ErrProviderRequired),
		errors.Is(err, admincatalogsvc.ErrModelRequired),
		errors.Is(err, admincatalogsvc.ErrDeploymentRequired):
		status = fiber.StatusBadRequest
	case errors.Is(err, admincatalogsvc.ErrServiceUnavailable):
		status = fiber.StatusInternalServerError
	}
	return httputil.WriteError(c, status, err.Error())
}

func deriveModelStatus(enabled bool, health router.RouteHealth) string {
	if !enabled {
		return "disabled"
	}
	if health.TotalRoutes == 0 {
		return "unknown"
	}
	if health.HealthyRoutes == 0 {
		return "offline"
	}
	if health.HealthyRoutes < health.TotalRoutes {
		return "degraded"
	}
	return "online"
}
