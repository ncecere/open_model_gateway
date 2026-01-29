package admin

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/middleware"
)

// Register wires up all /admin routes (auth + protected APIs).
func Register(app *fiber.App, container *app.Container, authRL config.AuthRateLimitConfig) {
	authGroup := app.Group("/admin/auth")

	// Apply brute-force rate limiting to login and refresh endpoints.
	authGroup.Use("/login", middleware.AuthRateLimit(authRL))
	authGroup.Use("/refresh", middleware.AuthRateLimit(authRL))

	registerAdminAuthRoutes(authGroup, container)

	// CSRF origin check for cookie-authenticated admin routes.
	trustedOrigins := []string{}
	if baseURL := strings.TrimSpace(container.Config.Public.BaseURL); baseURL != "" {
		trustedOrigins = append(trustedOrigins, baseURL)
	}
	protected := app.Group("/admin",
		middleware.CSRFOriginCheck(container.Config.Admin.Session.CookieName, trustedOrigins),
		adminAuthMiddleware(container),
	)
	registerAdminModelCatalogRoutes(protected, container)
	registerAdminAuditRoutes(protected, container)
	registerAdminDefaultModelRoutes(protected, container)
	registerAdminTenantRoutes(protected, container)
	registerAdminFileRoutes(protected, container)
	registerAdminBatchRoutes(protected, container)
	registerAdminUserRoutes(protected, container)
	registerAdminAPIKeyRoutes(protected, container)
	registerAdminUsageRoutes(protected, container)
	registerAdminUsageExportRoutes(protected, container)
	registerAdminSettingsRoutes(protected, container)
	registerAdminAdminKeyRoutes(protected, container)
	registerAdminBudgetRoutes(protected, container)
	registerAdminBillingWebhookRoutes(protected, container)
	registerAdminRateLimitRoutes(protected, container)
	registerAdminProviderRoutes(protected, container)
	registerAdminProviderHealthRoutes(protected, container)
	registerAdminSystemMetricsRoutes(protected, container)
}
