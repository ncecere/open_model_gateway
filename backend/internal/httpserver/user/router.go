package user

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
)

// Register wires up authenticated user-facing endpoints (used by the future user portal).
func Register(app *fiber.App, container *app.Container) {
	if app == nil || container == nil {
		return
	}

	handler := &userHandler{
		container: container,
		usage:     container.UsageService,
		tenantSvc: container.TenantService,
		modelSvc:  container.AdminCatalog,
	}

	group := app.Group("/user", userAuthMiddleware(container))
	group.Get("/profile", handler.profile)
	group.Patch("/profile", handler.updateProfile)
	group.Post("/profile/password", handler.changePassword)
	group.Get("/tenants", handler.tenants)
	group.Get("/tenants/:tenantID/summary", handler.tenantSummary)
	handler.registerAPIKeyRoutes(group)
	handler.registerUsageRoutes(group)
	handler.registerFileRoutes(group)
	group.Get("/models", handler.listModels)
	handler.registerBatchRoutes(group)
	handler.registerTenantManagementRoutes(group)
}
