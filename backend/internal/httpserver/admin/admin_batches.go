package admin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/batchdto"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

func registerAdminBatchRoutes(router fiber.Router, container *app.Container) {
	handler := &batchHandler{container: container}
	group := router.Group("/batches")
	group.Get("/", handler.list)
	group.Post("/:batchID/cancel", handler.cancel)
	group.Get("/:batchID/output", handler.downloadOutput)
	group.Get("/:batchID/errors", handler.downloadErrors)
}

type batchHandler struct {
	container *app.Container
}

func (h *batchHandler) list(c *fiber.Ctx) error {
	if h.container == nil || h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches service unavailable")
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

	statusParam := strings.TrimSpace(c.Query("status"))
	var statuses []string
	if statusParam != "" && statusParam != "all" {
		statuses = []string{statusParam}
	}

	search := strings.TrimSpace(c.Query("q"))

	batches, total, err := h.container.Batches.ListAll(
		c.UserContext(),
		tenantID,
		statuses,
		search,
		limit,
		offset,
	)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]batchdto.Batch, 0, len(batches))
	for _, batch := range batches {
		out = append(out, batchdto.FromBatchWithTenant(batch))
	}

	return c.JSON(fiber.Map{
		"object": "list",
		"data":   out,
		"limit":  limit,
		"offset": offset,
		"total":  total,
	})
}

func (h *batchHandler) cancel(c *fiber.Ctx) error {
	batchID, err := parseBatchID(c)
	if err != nil {
		return err
	}
	if h.container == nil || h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches service unavailable")
	}
	record, err := h.container.Batches.CancelByID(c.UserContext(), batchID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(batchdto.FromBatch(record))
}

func (h *batchHandler) downloadOutput(c *fiber.Ctx) error {
	return h.streamBatchFile(c, true)
}

func (h *batchHandler) downloadErrors(c *fiber.Ctx) error {
	return h.streamBatchFile(c, false)
}

func (h *batchHandler) streamBatchFile(c *fiber.Ctx, output bool) error {
	batchID, err := parseBatchID(c)
	if err != nil {
		return err
	}
	if h.container == nil || h.container.Batches == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batch files unavailable")
	}
	batch, err := h.container.Batches.GetByID(c.UserContext(), batchID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "batch not found")
	}

	fileID := batch.ResultFileID
	filenameSuffix := "output"
	if !output {
		fileID = batch.ErrorFileID
		filenameSuffix = "errors"
	}
	if fileID == nil {
		return httputil.WriteError(c, fiber.StatusNotFound, fmt.Sprintf("batch %s file not available", filenameSuffix))
	}

	reader, fileRec, err := h.container.Files.Open(c.UserContext(), batch.TenantID, *fileID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer reader.Close()

	contentType := fileRec.ContentType
	if contentType == "" {
		contentType = "application/x-ndjson"
	}
	c.Set("Content-Type", contentType)
	if fileRec.Bytes > 0 {
		c.Set("Content-Length", strconv.FormatInt(fileRec.Bytes, 10))
	}
	filename := fileRec.Filename
	if filename == "" {
		filename = fmt.Sprintf("batch_%s_%s.jsonl", batch.ID.String(), filenameSuffix)
	}
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Set("Cache-Control", "no-store")
	return c.SendStream(reader)
}

func parseBatchID(c *fiber.Ctx) (uuid.UUID, error) {
	batchID, err := uuid.Parse(strings.TrimSpace(c.Params("batchID")))
	if err != nil {
		return uuid.Nil, httputil.WriteError(c, fiber.StatusBadRequest, "invalid batch id")
	}
	return batchID, nil
}
