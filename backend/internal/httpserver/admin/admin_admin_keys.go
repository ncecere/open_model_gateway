package admin

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	adminapikeys "github.com/ncecere/open_model_gateway/backend/internal/services/adminapikeys"
)

type adminKeyHandler struct {
	container *app.Container
}

type adminKeyCreateRequest struct {
	Name             string `json:"name"`
	Scope            string `json:"scope"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

type adminKeyResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Scope           string  `json:"scope"`
	Prefix          string  `json:"prefix"`
	OwnerUserID     *string `json:"owner_user_id,omitempty"`
	OwnerName       string  `json:"owner_name,omitempty"`
	OwnerEmail      string  `json:"owner_email,omitempty"`
	CreatedByUserID string  `json:"created_by_user_id"`
	CreatorName     string  `json:"creator_name,omitempty"`
	CreatorEmail    string  `json:"creator_email,omitempty"`
	ExpiresAt       *string `json:"expires_at,omitempty"`
	RevokedAt       *string `json:"revoked_at,omitempty"`
	LastUsedAt      *string `json:"last_used_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

type adminKeyCreateResponse struct {
	AdminKey adminKeyResponse `json:"admin_key"`
	Token    string           `json:"token"`
}

func registerAdminAdminKeyRoutes(router fiber.Router, container *app.Container) {
	handler := &adminKeyHandler{container: container}
	group := router.Group("/admin-keys")
	group.Get("/", handler.list)
	group.Post("/", handler.create)
	group.Delete("/:id", handler.revoke)
}

func (h *adminKeyHandler) list(c *fiber.Ctx) error {
	if h.container == nil || h.container.AdminAPIKeys == nil {
		return writeMissingAdminKeyService(c)
	}
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	user, ok := adminUserFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "missing admin context")
	}
	var rows []db.ListAdminAPIKeysRow
	var err error
	if user.IsSuperAdmin {
		rows, err = h.container.AdminAPIKeys.List(c.Context())
	} else {
		userID, ok := adminUserIDFromContext(c.UserContext())
		if !ok {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "missing admin context")
		}
		rows, err = h.container.AdminAPIKeys.ListByOwner(c.Context(), userID)
	}
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	items := make([]adminKeyResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAdminKeyRow(row))
	}
	return c.JSON(fiber.Map{"admin_keys": items})
}

func (h *adminKeyHandler) create(c *fiber.Ctx) error {
	if h.container == nil || h.container.AdminAPIKeys == nil {
		return writeMissingAdminKeyService(c)
	}
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	user, ok := adminUserFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "missing admin context")
	}
	userID, ok := adminUserIDFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "missing admin context")
	}

	var req adminKeyCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	if req.ExpiresInSeconds <= 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "expires_in_seconds required")
	}
	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope == "" {
		scope = adminapikeys.ScopeAdmin
	}
	if scope == adminapikeys.ScopeSystem && !user.IsSuperAdmin {
		return httputil.WriteError(c, fiber.StatusForbidden, "system keys require super admin")
	}

	var ownerID *uuid.UUID
	if scope == adminapikeys.ScopeAdmin {
		ownerID = &userID
	}

	expiresAt := time.Now().UTC().Add(time.Duration(req.ExpiresInSeconds) * time.Second)
	result, err := h.container.AdminAPIKeys.Create(c.Context(), adminapikeys.CreateParams{
		Name:            req.Name,
		Scope:           scope,
		OwnerUserID:     ownerID,
		CreatedByUserID: userID,
		ExpiresAt:       expiresAt,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	row := db.ListAdminAPIKeysRow{
		ID:              result.Key.ID,
		Name:            result.Key.Name,
		Prefix:          result.Key.Prefix,
		SecretHash:      result.Key.SecretHash,
		Scope:           result.Key.Scope,
		OwnerUserID:     result.Key.OwnerUserID,
		CreatedByUserID: result.Key.CreatedByUserID,
		ExpiresAt:       result.Key.ExpiresAt,
		RevokedAt:       result.Key.RevokedAt,
		LastUsedAt:      result.Key.LastUsedAt,
		CreatedAt:       result.Key.CreatedAt,
		OwnerEmail:      pgtype.Text{String: user.Email, Valid: ownerID != nil},
		OwnerName:       pgtype.Text{String: user.Name, Valid: ownerID != nil},
		CreatorEmail:    user.Email,
		CreatorName:     user.Name,
	}

	if err := recordAudit(c, h.container, "admin_api_key.create", "admin_api_key", result.Key.ID.String(), fiber.Map{
		"scope":      scope,
		"name":       result.Key.Name,
		"expires_at": expiresAt.Format(time.RFC3339),
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(adminKeyCreateResponse{
		AdminKey: mapAdminKeyRow(row),
		Token:    result.Token,
	})
}

func (h *adminKeyHandler) revoke(c *fiber.Ctx) error {
	if h.container == nil || h.container.AdminAPIKeys == nil {
		return writeMissingAdminKeyService(c)
	}
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	user, ok := adminUserFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "missing admin context")
	}
	userID, ok := adminUserIDFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "missing admin context")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid key id")
	}
	record, err := h.container.AdminAPIKeys.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "key not found")
	}
	if !user.IsSuperAdmin {
		if !record.OwnerUserID.Valid || record.OwnerUserID.Bytes != userID {
			return httputil.WriteError(c, fiber.StatusForbidden, "insufficient permissions")
		}
	}
	revoked, err := h.container.AdminAPIKeys.Revoke(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := recordAudit(c, h.container, "admin_api_key.revoke", "admin_api_key", revoked.ID.String(), fiber.Map{
		"scope": string(revoked.Scope),
		"name":  revoked.Name,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"revoked": true})
}

func mapAdminKeyRow(row db.ListAdminAPIKeysRow) adminKeyResponse {
	var ownerID *string
	if row.OwnerUserID.Valid {
		if id, err := fromPgUUID(row.OwnerUserID); err == nil {
			value := id.String()
			ownerID = &value
		}
	}
	expiresAt := formatPgTime(row.ExpiresAt)
	revokedAt := formatPgTime(row.RevokedAt)
	lastUsedAt := formatPgTime(row.LastUsedAt)
	creatorID := ""
	if id, err := fromPgUUID(row.CreatedByUserID); err == nil {
		creatorID = id.String()
	}
	createdAt := ""
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time.Format(time.RFC3339)
	}
	return adminKeyResponse{
		ID:              row.ID.String(),
		Name:            row.Name,
		Scope:           string(row.Scope),
		Prefix:          row.Prefix,
		OwnerUserID:     ownerID,
		OwnerName:       pgText(row.OwnerName),
		OwnerEmail:      pgText(row.OwnerEmail),
		CreatedByUserID: creatorID,
		CreatorName:     row.CreatorName,
		CreatorEmail:    row.CreatorEmail,
		ExpiresAt:       expiresAt,
		RevokedAt:       revokedAt,
		LastUsedAt:      lastUsedAt,
		CreatedAt:       createdAt,
	}
}

func pgText(value pgtype.Text) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func formatPgTime(value pgtype.Timestamptz) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Time.Format(time.RFC3339)
	return &formatted
}

func writeMissingAdminKeyService(c *fiber.Ctx) error {
	return httputil.WriteError(c, fiber.StatusNotImplemented, "admin api key service unavailable")
}
