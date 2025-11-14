package admin

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/catalog"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

func registerAdminDefaultModelRoutes(router fiber.Router, container *app.Container) {
	handler := &defaultModelHandler{container: container}
	group := router.Group("/settings/default-models")
	group.Get("/", handler.list)
	group.Post("/", handler.create)
	group.Delete("/:alias", handler.delete)
}

type defaultModelHandler struct {
	container *app.Container
}

type defaultModelRequest struct {
	Alias string `json:"alias"`
}

func (h *defaultModelHandler) list(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	models, err := h.container.DefaultModels.List(c.Context())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"models": models})
}

func (h *defaultModelHandler) create(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	var req defaultModelRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	alias := strings.TrimSpace(req.Alias)
	if err := h.container.DefaultModels.Add(c.Context(), alias); err != nil {
		return writeDefaultModelError(c, err)
	}
	if err := h.syncPersonalDefaults(c); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"alias": strings.ToLower(alias)})
}

func (h *defaultModelHandler) delete(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	alias := strings.TrimSpace(c.Params("alias"))
	if err := h.container.DefaultModels.Remove(c.Context(), alias); err != nil {
		return writeDefaultModelError(c, err)
	}
	if err := h.syncPersonalDefaults(c); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *defaultModelHandler) syncPersonalDefaults(c *fiber.Ctx) error {
	if h.container == nil || h.container.Accounts == nil {
		return nil
	}
	if err := h.container.Accounts.SyncDefaultModels(c.Context()); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to sync personal tenants")
	}
	return nil
}

func writeDefaultModelError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, catalog.ErrAliasRequired), errors.Is(err, catalog.ErrAliasUnknown):
		status = fiber.StatusBadRequest
	case errors.Is(err, catalog.ErrServiceUnavailable):
		status = fiber.StatusInternalServerError
	}
	return httputil.WriteError(c, status, err.Error())
}
