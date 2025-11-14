package admin

import (
    "github.com/gofiber/fiber/v2"

    "github.com/ncecere/open_model_gateway/backend/internal/app"
    "github.com/ncecere/open_model_gateway/backend/internal/db"
    "github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

func registerAdminAPIKeyRoutes(router fiber.Router, container *app.Container) {
    handler := &apiKeyHandler{container: container}
    group := router.Group("/api-keys")
    group.Get("/", handler.list)
}

type apiKeyHandler struct {
    container *app.Container
}

func (h *apiKeyHandler) list(c *fiber.Ctx) error {
    if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
        return err
    }
    if h.container == nil || h.container.Queries == nil {
        return httputil.WriteError(c, fiber.StatusInternalServerError, "api key service unavailable")
    }
    rows, err := h.container.Queries.ListAllAPIKeys(c.Context())
    if err != nil {
        return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
    }
    responses := make([]apiKeyResponse, 0, len(rows))
    for _, row := range rows {
        tenantID, err := fromPgUUID(row.TenantID)
        if err != nil {
            return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
        }
        key := db.ApiKey{
            ID:          row.ID,
            TenantID:    row.TenantID,
            Prefix:      row.Prefix,
            Name:        row.Name,
            ScopesJson:  row.ScopesJson,
            QuotaJson:   row.QuotaJson,
            Kind:        row.Kind,
            OwnerUserID: row.OwnerUserID,
            CreatedAt:   row.CreatedAt,
            RevokedAt:   row.RevokedAt,
            LastUsedAt:  row.LastUsedAt,
        }
        ownerEmail := ""
        if row.OwnerEmail.Valid {
            ownerEmail = row.OwnerEmail.String
        }
        ownerName := ""
        if row.OwnerName.Valid {
            ownerName = row.OwnerName.String
        }
        issuer := resolveAPIKeyIssuer(key, row.TenantName, ownerName, ownerEmail)
        resp, err := buildAPIKeyResponse(c.Context(), h.container, key, tenantID, row.TenantName, issuer)
        if err != nil {
            return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
        }
        responses = append(responses, resp)
    }
    return c.JSON(fiber.Map{
        "api_keys": responses,
    })
}
