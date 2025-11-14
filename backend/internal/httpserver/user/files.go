package user

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
)

type userFileResponse struct {
	ID          string  `json:"id"`
	TenantID    string  `json:"tenant_id"`
	Filename    string  `json:"filename"`
	Purpose     string  `json:"purpose"`
	ContentType string  `json:"content_type"`
	Bytes       int64   `json:"bytes"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   string  `json:"expires_at"`
	DeletedAt   *string `json:"deleted_at,omitempty"`
}

func (h *userHandler) registerFileRoutes(group fiber.Router) {
	group.Get("/tenants/:tenantID/files", h.listTenantFiles)
}

func (h *userHandler) listTenantFiles(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}

	if h.tenantSvc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	if h.container == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "files service unavailable")
	}

	tenantParam := strings.TrimSpace(c.Params("tenantID"))
	tenantUUID, err := uuid.Parse(tenantParam)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	if _, err := h.tenantSvc.GetTenantSummary(c.Context(), user, tenantUUID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	limit := parsePositiveQueryInt(c, "limit", 50, 200)
	offset := parseNonNegativeQueryInt(c, "offset", 0)

	records, err := h.container.Files.List(c.Context(), tenantUUID, int32(limit), int32(offset))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	files := make([]userFileResponse, 0, len(records))
	for _, record := range records {
		files = append(files, serializeUserFile(record))
	}

	return c.JSON(fiber.Map{
		"files": files,
	})
}

func serializeUserFile(record filesvc.FileRecord) userFileResponse {
	var deleted *string
	if record.DeletedAt != nil {
		value := record.DeletedAt.Format(time.RFC3339)
		deleted = &value
	}
	return userFileResponse{
		ID:          record.ID.String(),
		TenantID:    record.TenantID.String(),
		Filename:    record.Filename,
		Purpose:     record.Purpose,
		ContentType: record.ContentType,
		Bytes:       record.Bytes,
		CreatedAt:   record.CreatedAt.Format(time.RFC3339),
		ExpiresAt:   record.ExpiresAt.Format(time.RFC3339),
		DeletedAt:   deleted,
	}
}

func parsePositiveQueryInt(c *fiber.Ctx, key string, fallback int, max int) int {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if max > 0 && parsed > max {
		return max
	}
	return parsed
}

func parseNonNegativeQueryInt(c *fiber.Ctx, key string, fallback int) int {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}
