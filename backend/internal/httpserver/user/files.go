package user

import (
	"errors"
	"fmt"
	"io"
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
	ID            string  `json:"id"`
	TenantID      string  `json:"tenant_id"`
	Filename      string  `json:"filename"`
	Purpose       string  `json:"purpose"`
	ContentType   string  `json:"content_type"`
	Bytes         int64   `json:"bytes"`
	CreatedAt     string  `json:"created_at"`
	ExpiresAt     string  `json:"expires_at"`
	DeletedAt     *string `json:"deleted_at,omitempty"`
	Status        string  `json:"status"`
	StatusDetails string  `json:"status_details,omitempty"`
}

func (h *userHandler) registerFileRoutes(group fiber.Router) {
	group.Get("/tenants/:tenantID/files", h.listTenantFiles)
	group.Get("/tenants/:tenantID/files/:fileID/content", h.downloadTenantFile)
	group.Get("/files/settings", h.fileSettings)
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
	purpose := strings.TrimSpace(c.Query("purpose"))
	var afterID *uuid.UUID
	if cursor := strings.TrimSpace(c.Query("after")); cursor != "" {
		parsed, perr := uuid.Parse(cursor)
		if perr != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid after cursor")
		}
		afterID = &parsed
	}

	result, err := h.container.Files.List(c.Context(), tenantUUID, filesvc.ListOptions{
		Purpose: purpose,
		Limit:   int32(limit),
		AfterID: afterID,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	files := make([]userFileResponse, 0, len(result.Files))
	for _, record := range result.Files {
		files = append(files, serializeUserFile(record))
	}

	payload := fiber.Map{
		"files":    files,
		"has_more": result.HasMore,
	}
	if result.HasMore && result.LastID != nil {
		nextCursor := result.LastID.String()
		payload["next_cursor"] = nextCursor
	}
	return c.JSON(payload)
}

func (h *userHandler) downloadTenantFile(c *fiber.Ctx) error {
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
	fileID, err := uuid.Parse(strings.TrimSpace(c.Params("fileID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid file id")
	}
	reader, record, err := h.container.Files.Open(c.UserContext(), tenantUUID, fileID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "file not found")
	}
	defer reader.Close()
	c.Set("Content-Type", record.ContentType)
	c.Set("Content-Length", strconv.FormatInt(record.Bytes, 10))
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", record.Filename))
	_, err = io.Copy(c, reader)
	return err
}

func (h *userHandler) fileSettings(c *fiber.Ctx) error {
	if h.container == nil || h.container.Config == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "configuration unavailable")
	}
	filesCfg := h.container.Config.Files
	return c.JSON(fiber.Map{
		"default_ttl_seconds": int64(filesCfg.DefaultTTL.Seconds()),
		"max_ttl_seconds":     int64(filesCfg.MaxTTL.Seconds()),
	})
}

func serializeUserFile(record filesvc.FileRecord) userFileResponse {
	var deleted *string
	if record.DeletedAt != nil {
		value := record.DeletedAt.Format(time.RFC3339)
		deleted = &value
	}
	return userFileResponse{
		ID:            record.ID.String(),
		TenantID:      record.TenantID.String(),
		Filename:      record.Filename,
		Purpose:       record.Purpose,
		ContentType:   record.ContentType,
		Bytes:         record.Bytes,
		CreatedAt:     record.CreatedAt.Format(time.RFC3339),
		ExpiresAt:     record.ExpiresAt.Format(time.RFC3339),
		DeletedAt:     deleted,
		Status:        record.Status,
		StatusDetails: record.StatusDetails,
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
