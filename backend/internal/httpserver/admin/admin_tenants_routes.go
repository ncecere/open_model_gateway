package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ncecere/open_model_gateway/backend/internal/app"
)

func registerAdminTenantRoutes(router fiber.Router, container *app.Container) {
	handler := &tenantHandler{container: container, service: container.AdminTenants}

	group := router.Group("/tenants")
	group.Get("/", handler.list)
	group.Get("/personal", handler.listPersonal)
	group.Post("/", handler.create)
	group.Patch("/:tenantID", handler.updateDetails)
	group.Patch("/:tenantID/status", handler.updateStatus)
	group.Get("/:tenantID/budget", handler.getBudget)
	group.Put("/:tenantID/budget", handler.upsertBudget)
	group.Delete("/:tenantID/budget", handler.deleteBudget)
	group.Get("/:tenantID/rate-limits", handler.getRateLimits)
	group.Put("/:tenantID/rate-limits", handler.upsertRateLimits)
	group.Delete("/:tenantID/rate-limits", handler.deleteRateLimits)
	group.Get("/:tenantID/models", handler.getTenantModels)
	group.Put("/:tenantID/models", handler.upsertTenantModels)
	group.Delete("/:tenantID/models", handler.deleteTenantModels)
	group.Get("/:tenantID/api-keys", handler.listAPIKeys)
	group.Post("/:tenantID/api-keys", handler.createAPIKey)
	group.Delete("/:tenantID/api-keys/:apiKeyID", handler.revokeAPIKey)
	group.Get("/:tenantID/memberships", handler.listMemberships)
	group.Post("/:tenantID/memberships", handler.upsertMembership)
	group.Delete("/:tenantID/memberships/:userID", handler.removeMembership)
	group.Get("/:tenantID/batches", handler.listBatches)
	group.Get("/:tenantID/batches/:batchID", handler.getBatch)
	group.Post("/:tenantID/batches/:batchID/cancel", handler.cancelBatch)
	group.Get("/:tenantID/batches/:batchID/output", handler.downloadBatchOutput)
	group.Get("/:tenantID/batches/:batchID/errors", handler.downloadBatchErrors)
}
