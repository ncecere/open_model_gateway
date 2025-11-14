package admin

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
)

// Register wires up all /admin routes (auth + protected APIs).
func Register(app *fiber.App, container *app.Container) {
	authGroup := app.Group("/admin/auth")
	registerAdminAuthRoutes(authGroup, container)

	protected := app.Group("/admin", adminAuthMiddleware(container))
	registerAdminModelCatalogRoutes(protected, container)
	registerAdminAuditRoutes(protected, container)
	registerAdminDefaultModelRoutes(protected, container)
	registerAdminTenantRoutes(protected, container)
	registerAdminFileRoutes(protected, container)
	registerAdminBatchRoutes(protected, container)
	registerAdminUserRoutes(protected, container)
	registerAdminAPIKeyRoutes(protected, container)
	registerAdminUsageRoutes(protected, container)
	registerAdminSettingsRoutes(protected, container)
	registerAdminBudgetRoutes(protected, container)
	registerAdminRateLimitRoutes(protected, container)
	registerAdminProviderRoutes(protected, container)
}
