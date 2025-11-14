package public

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
	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	"github.com/ncecere/open_model_gateway/backend/internal/storage/blob"
)

type filesHandler struct {
	container *app.Container
}

func (h *filesHandler) list(c *fiber.Ctx) error {
	rc, err := h.requireRequestContext(c)
	if err != nil {
		return err
	}
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	if h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "files service disabled")
	}
	records, err := h.container.Files.List(c.UserContext(), rc.TenantID, int32(limit), int32(offset))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	resp := make([]openAIFile, 0, len(records))
	for _, rec := range records {
		resp = append(resp, toOpenAIFile(rec))
	}
	return c.JSON(openAIFileList{Object: "list", Data: resp})
}

func (h *filesHandler) get(c *fiber.Ctx) error {
	rc, err := h.requireRequestContext(c)
	if err != nil {
		return err
	}
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid file id")
	}
	if h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "files service disabled")
	}
	reader, rec, err := h.container.Files.Open(c.UserContext(), rc.TenantID, id)
	if err != nil {
		return translateFileError(c, err)
	}
	reader.Close()
	return c.JSON(toOpenAIFile(rec))
}

func (h *filesHandler) delete(c *fiber.Ctx) error {
	rc, err := h.requireRequestContext(c)
	if err != nil {
		return err
	}
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid file id")
	}
	if h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "files service disabled")
	}
	if err := h.container.Files.Delete(c.UserContext(), rc.TenantID, id); err != nil {
		return translateFileError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *filesHandler) download(c *fiber.Ctx) error {
	rc, err := h.requireRequestContext(c)
	if err != nil {
		return err
	}
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid file id")
	}
	if h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "files service disabled")
	}
	reader, rec, err := h.container.Files.Open(c.UserContext(), rc.TenantID, id)
	if err != nil {
		return translateFileError(c, err)
	}
	defer reader.Close()
	c.Set("Content-Type", rec.ContentType)
	c.Set("Content-Length", strconv.FormatInt(rec.Bytes, 10))
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", rec.Filename))
	_, err = io.Copy(c, reader)
	return err
}

func (h *filesHandler) upload(c *fiber.Ctx) error {
	rc, err := h.requireRequestContext(c)
	if err != nil {
		return err
	}
	if h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "files service disabled")
	}
	form, err := c.MultipartForm()
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "multipart form required")
	}
	fileHeaders := form.File["file"]
	if len(fileHeaders) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "file is required")
	}
	file := fileHeaders[0]
	purpose := form.Value["purpose"]
	reader, err := file.Open()
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "failed to open file")
	}
	defer reader.Close()
	params := filesvc.UploadParams{
		TenantID:    rc.TenantID,
		Filename:    file.Filename,
		Purpose:     firstValue(purpose),
		ContentType: file.Header.Get("Content-Type"),
		ContentLen:  file.Size,
		Reader:      reader,
	}
	ttlStr := c.FormValue("expires_in")
	if ttlStr != "" {
		if ttlSeconds, err := strconv.Atoi(ttlStr); err == nil {
			params.TTL = time.Duration(ttlSeconds) * time.Second
		}
	}
	record, err := h.container.Files.Upload(c.UserContext(), params)
	if err != nil {
		return translateFileError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(toOpenAIFile(record))
}

func (h *filesHandler) createUpload(c *fiber.Ctx) error {
	return httputil.WriteError(c, fiber.StatusNotImplemented, "uploads not yet implemented")
}

func (h *filesHandler) requireRequestContext(c *fiber.Ctx) (*requestctx.Context, error) {
	rc, ok := requestctx.FromContext(c.UserContext())
	if !ok || rc == nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "request context missing")
	}
	return rc, nil
}

func translateFileError(c *fiber.Ctx, err error) error {
	switch {
	case err == nil:
		return nil
	case strings.Contains(err.Error(), "unsupported purpose"):
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	case errors.Is(err, blob.ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		return httputil.WriteError(c, fiber.StatusNotFound, "file not found")
	case errors.Is(err, fiber.ErrUnauthorized):
		return httputil.WriteError(c, fiber.StatusForbidden, err.Error())
	default:
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
}

func toOpenAIFile(rec filesvc.FileRecord) openAIFile {
	return openAIFile{
		ID:        rec.ID.String(),
		Object:    "file",
		Bytes:     rec.Bytes,
		CreatedAt: rec.CreatedAt.Unix(),
		Filename:  rec.Filename,
		Purpose:   rec.Purpose,
		ExpiresAt: rec.ExpiresAt.Unix(),
	}
}

type openAIFile struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Bytes     int64  `json:"bytes"`
	CreatedAt int64  `json:"created_at"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
	ExpiresAt int64  `json:"expires_at"`
}

type openAIFileList struct {
	Object string       `json:"object"`
	Data   []openAIFile `json:"data"`
}

func parseUUID(str string) (uuid.UUID, error) {
	return uuid.Parse(str)
}

func parseQueryInt(c *fiber.Ctx, key string, def int) int {
	if val := c.Query(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return def
}

func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
