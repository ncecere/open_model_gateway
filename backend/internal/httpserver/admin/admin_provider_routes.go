package admin

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	adminprovidersvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminprovider"
)

type providerHandler struct {
	container *app.Container
	service   *adminprovidersvc.Service
}

func registerAdminProviderRoutes(router fiber.Router, container *app.Container) {
	handler := &providerHandler{container: container, service: container.AdminProviders}
	router.Get("/providers", handler.list)
}

func (h *providerHandler) list(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "provider service unavailable")
	}
	defs, err := h.service.List(c.Context())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"providers": defs})
}
