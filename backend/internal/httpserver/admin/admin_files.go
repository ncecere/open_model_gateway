package admin

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
)

func registerAdminFileRoutes(router fiber.Router, container *app.Container) {
	handler := &fileHandler{container: container}
	group := router.Group("/files")
	group.Get("/", handler.list)
	group.Get(":fileID", handler.get)
	group.Get(":fileID/content", handler.download)
	group.Delete(":fileID", handler.delete)
}

type fileHandler struct {
	container *app.Container
}

type fileResponse struct {
	ID             string  `json:"id"`
	TenantID       string  `json:"tenant_id"`
	TenantName     string  `json:"tenant_name"`
	Filename       string  `json:"filename"`
	Purpose        string  `json:"purpose"`
	ContentType    string  `json:"content_type"`
	Bytes          int64   `json:"bytes"`
	StorageBackend string  `json:"storage_backend"`
	Encrypted      bool    `json:"encrypted"`
	Checksum       string  `json:"checksum"`
	ExpiresAt      string  `json:"expires_at"`
	CreatedAt      string  `json:"created_at"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
	Status         string  `json:"status"`
	StatusDetails  string  `json:"status_details,omitempty"`
}

type fileListResponse struct {
	Object string         `json:"object"`
	Data   []fileResponse `json:"data"`
	Limit  int32          `json:"limit"`
	Offset int32          `json:"offset"`
	Total  int64          `json:"total"`
}

func (h *fileHandler) list(c *fiber.Ctx) error {
	if h.container == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "files service unavailable")
	}
	limit, offset := parseBatchPagination(c)

	var tenantID *uuid.UUID
	if tenantParam := strings.TrimSpace(c.Query("tenant_id")); tenantParam != "" && tenantParam != "all" {
		id, err := uuid.Parse(tenantParam)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant filter")
		}
		tenantID = &id
	}

	state := strings.ToLower(strings.TrimSpace(c.Query("state")))
	if state == "" {
		state = "active"
	}
	if state != "active" && state != "deleted" && state != "all" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid state filter")
	}

	purpose := strings.TrimSpace(c.Query("purpose"))
	search := strings.TrimSpace(c.Query("q"))

	records, total, err := h.container.Files.ListAll(
		c.UserContext(),
		tenantID,
		purpose,
		search,
		state,
		limit,
		offset,
	)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	responses := make([]fileResponse, 0, len(records))
	for _, record := range records {
		responses = append(responses, marshalFileRecord(record))
	}

	return c.JSON(fileListResponse{
		Object: "list",
		Data:   responses,
		Limit:  limit,
		Offset: offset,
		Total:  total,
	})
}

func (h *fileHandler) get(c *fiber.Ctx) error {
	if h.container == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "files service unavailable")
	}
	fileID, err := uuid.Parse(strings.TrimSpace(c.Params("fileID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid file id")
	}
	record, err := h.container.Files.GetWithTenant(c.UserContext(), fileID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "file not found")
	}
	return c.JSON(marshalFileRecord(record))
}

func (h *fileHandler) delete(c *fiber.Ctx) error {
	if h.container == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "files service unavailable")
	}
	fileID, err := uuid.Parse(strings.TrimSpace(c.Params("fileID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid file id")
	}
	if err := h.container.Files.DeleteByID(c.UserContext(), fileID); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *fileHandler) download(c *fiber.Ctx) error {
	if h.container == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "files service unavailable")
	}
	fileID, err := uuid.Parse(strings.TrimSpace(c.Params("fileID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid file id")
	}
	record, err := h.container.Files.GetWithTenant(c.UserContext(), fileID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "file not found")
	}
	reader, meta, err := h.container.Files.Open(c.UserContext(), record.TenantID, fileID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer reader.Close()
	c.Set("Content-Type", meta.ContentType)
	c.Set("Content-Length", strconv.FormatInt(meta.Bytes, 10))
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", meta.Filename))
	_, err = io.Copy(c, reader)
	return err
}

func marshalFileRecord(record filesvc.FileWithTenant) fileResponse {
	var deleted *string
	if record.DeletedAt != nil {
		ds := record.DeletedAt.Format(time.RFC3339)
		deleted = &ds
	}
	return fileResponse{
		ID:             record.ID.String(),
		TenantID:       record.TenantID.String(),
		TenantName:     record.TenantName,
		Filename:       record.Filename,
		Purpose:        record.Purpose,
		ContentType:    record.ContentType,
		Bytes:          record.Bytes,
		StorageBackend: record.StorageBackend,
		Encrypted:      record.Encrypted,
		Checksum:       record.Checksum,
		ExpiresAt:      record.ExpiresAt.Format(time.RFC3339),
		CreatedAt:      record.CreatedAt.Format(time.RFC3339),
		DeletedAt:      deleted,
		Status:         record.Status,
		StatusDetails:  record.StatusDetails,
	}
}
