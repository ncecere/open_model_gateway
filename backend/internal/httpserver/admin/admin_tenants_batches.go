package admin

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/batchdto"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

func (h *tenantHandler) listBatches(c *fiber.Ctx) error {
	tenantUUID, err := uuid.Parse(c.Params("tenantID"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches service unavailable")
	}

	limit, offset := parseBatchPagination(c)
	records, err := h.container.Batches.List(c.UserContext(), tenantUUID, limit, offset)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]batchdto.Batch, 0, len(records))
	for _, record := range records {
		out = append(out, batchdto.FromBatch(record))
	}
	return c.JSON(fiber.Map{
		"object": "list",
		"data":   out,
	})
}

func (h *tenantHandler) getBatch(c *fiber.Ctx) error {
	tenantUUID, batchUUID, ok := h.parseBatchRouteParams(c)
	if !ok {
		return nil
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches service unavailable")
	}
	record, err := h.container.Batches.Get(c.UserContext(), tenantUUID, batchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "batch not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(batchdto.FromBatch(record))
}

func (h *tenantHandler) cancelBatch(c *fiber.Ctx) error {
	tenantUUID, batchUUID, ok := h.parseBatchRouteParams(c)
	if !ok {
		return nil
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches service unavailable")
	}
	record, err := h.container.Batches.Cancel(c.UserContext(), tenantUUID, batchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "batch not found or cannot transition")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(batchdto.FromBatch(record))
}

func (h *tenantHandler) downloadBatchOutput(c *fiber.Ctx) error {
	return h.streamBatchFile(c, true)
}

func (h *tenantHandler) downloadBatchErrors(c *fiber.Ctx) error {
	return h.streamBatchFile(c, false)
}

func (h *tenantHandler) streamBatchFile(c *fiber.Ctx, output bool) error {
	tenantUUID, batchUUID, ok := h.parseBatchRouteParams(c)
	if !ok {
		return nil
	}
	if err := requireTenantRole(c, h.container, tenantUUID, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.container.Batches == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batch files unavailable")
	}

	batch, err := h.container.Batches.Get(c.UserContext(), tenantUUID, batchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "batch not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	var fileID *uuid.UUID
	filenameSuffix := "output"
	if output {
		fileID = batch.ResultFileID
	} else {
		fileID = batch.ErrorFileID
		filenameSuffix = "errors"
	}
	if fileID == nil {
		return httputil.WriteError(c, fiber.StatusNotFound, fmt.Sprintf("batch %s file not available", filenameSuffix))
	}

	reader, fileRec, err := h.container.Files.Open(c.UserContext(), tenantUUID, *fileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "file not found")
		}
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
		filename = fmt.Sprintf("batch_%s_%s.jsonl", batchUUID.String(), filenameSuffix)
	}
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Set("Cache-Control", "no-store")
	return c.SendStream(reader)
}

func (h *tenantHandler) parseBatchRouteParams(c *fiber.Ctx) (uuid.UUID, uuid.UUID, bool) {
	tenantUUID, err := uuid.Parse(c.Params("tenantID"))
	if err != nil {
		_ = httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	batchUUID, err := uuid.Parse(c.Params("batchID"))
	if err != nil {
		_ = httputil.WriteError(c, fiber.StatusBadRequest, "invalid batch id")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return tenantUUID, batchUUID, true
}
