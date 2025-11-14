package public

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
)

type batchHandler struct {
	container *app.Container
}

type createBatchRequest struct {
	InputFileID      string            `json:"input_file_id"`
	Endpoint         string            `json:"endpoint"`
	CompletionWindow string            `json:"completion_window"`
	Metadata         map[string]string `json:"metadata"`
	MaxConcurrency   *int              `json:"max_concurrency"`
}

type openAIBatchResponse struct {
	ID               string            `json:"id"`
	Object           string            `json:"object"`
	Endpoint         string            `json:"endpoint"`
	Status           string            `json:"status"`
	CompletionWindow string            `json:"completion_window"`
	CreatedAt        int64             `json:"created_at"`
	InProgressAt     *int64            `json:"in_progress_at"`
	CompletedAt      *int64            `json:"completed_at"`
	CancelledAt      *int64            `json:"cancelled_at"`
	FailedAt         *int64            `json:"failed_at"`
	FinalizingAt     *int64            `json:"finalizing_at"`
	ExpiresAt        *int64            `json:"expires_at"`
	InputFileID      string            `json:"input_file_id"`
	OutputFileID     *string           `json:"output_file_id"`
	ErrorFileID      *string           `json:"error_file_id"`
	RequestCounts    openAIBatchCounts `json:"request_counts"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type openAIBatchCounts struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
}

type openAIBatchList struct {
	Object string                `json:"object"`
	Data   []openAIBatchResponse `json:"data"`
}

func (h *batchHandler) create(c *fiber.Ctx) error {
	rc, err := h.requireContext(c)
	if err != nil {
		return err
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches not enabled")
	}
	var req createBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	inputID, err := uuid.Parse(req.InputFileID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid input_file_id")
	}
	maxConc := 0
	if req.MaxConcurrency != nil {
		maxConc = *req.MaxConcurrency
	}

	if alias := rc.TenantID; alias == (uuid.UUID{}) {
		slog.Error("batch create missing tenant id", slog.String("api_key", rc.APIKeyPrefix))
	}

	batch, err := h.container.Batches.Create(c.UserContext(), batchsvc.CreateParams{
		TenantID:         rc.TenantID,
		APIKeyID:         rc.APIKeyID,
		Endpoint:         req.Endpoint,
		CompletionWindow: req.CompletionWindow,
		InputFileID:      inputID,
		Metadata:         req.Metadata,
		MaxConcurrency:   maxConc,
	})
	if err != nil {
		return h.translateBatchError(c, err)
	}
	return c.Status(http.StatusOK).JSON(toOpenAIBatch(batch))
}

func (h *batchHandler) list(c *fiber.Ctx) error {
	rc, err := h.requireContext(c)
	if err != nil {
		return err
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches not enabled")
	}
	limit := parseQueryInt(c, "limit", 20)
	offset := parseQueryInt(c, "offset", 0)
	records, err := h.container.Batches.List(c.UserContext(), rc.TenantID, int32(limit), int32(offset))
	if err != nil {
		return h.translateBatchError(c, err)
	}
	out := make([]openAIBatchResponse, 0, len(records))
	for _, b := range records {
		out = append(out, toOpenAIBatch(b))
	}
	return c.JSON(openAIBatchList{Object: "list", Data: out})
}

func (h *batchHandler) get(c *fiber.Ctx) error {
	rc, err := h.requireContext(c)
	if err != nil {
		return err
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches not enabled")
	}
	batchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid batch id")
	}
	batch, err := h.container.Batches.Get(c.UserContext(), rc.TenantID, batchID)
	if err != nil {
		return h.translateBatchError(c, err)
	}
	return c.JSON(toOpenAIBatch(batch))
}

func (h *batchHandler) cancel(c *fiber.Ctx) error {
	rc, err := h.requireContext(c)
	if err != nil {
		return err
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches not enabled")
	}
	batchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid batch id")
	}
	batch, err := h.container.Batches.Cancel(c.UserContext(), rc.TenantID, batchID)
	if err != nil {
		return h.translateBatchError(c, err)
	}
	return c.JSON(toOpenAIBatch(batch))
}

func (h *batchHandler) output(c *fiber.Ctx) error {
	rc, err := h.requireContext(c)
	if err != nil {
		return err
	}
	if h.container.Batches == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches output not available")
	}
	batchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid batch id")
	}
	batch, err := h.container.Batches.Get(c.UserContext(), rc.TenantID, batchID)
	if err != nil {
		return h.translateBatchError(c, err)
	}
	if batch.ResultFileID == nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "batch output not available yet")
	}

	reader, fileRec, err := h.container.Files.Open(c.UserContext(), rc.TenantID, *batch.ResultFileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "output file not found")
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
		filename = fmt.Sprintf("batch_%s_output.jsonl", batchID.String())
	}
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Set("Cache-Control", "no-store")
	if _, err := io.Copy(c, reader); err != nil {
		slog.Warn("batch output download failed", slog.String("batch_id", batchID.String()), slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (h *batchHandler) errors(c *fiber.Ctx) error {
	rc, err := h.requireContext(c)
	if err != nil {
		return err
	}
	if h.container.Batches == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches error output not available")
	}
	batchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid batch id")
	}
	batch, err := h.container.Batches.Get(c.UserContext(), rc.TenantID, batchID)
	if err != nil {
		return h.translateBatchError(c, err)
	}
	if batch.ErrorFileID == nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "batch error file not available")
	}

	reader, fileRec, err := h.container.Files.Open(c.UserContext(), rc.TenantID, *batch.ErrorFileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "error file not found")
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
		filename = fmt.Sprintf("batch_%s_errors.jsonl", batchID.String())
	}
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Set("Cache-Control", "no-store")
	if _, err := io.Copy(c, reader); err != nil {
		slog.Warn("batch error download failed", slog.String("batch_id", batchID.String()), slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (h *batchHandler) requireContext(c *fiber.Ctx) (*requestctx.Context, error) {
	rc, ok := requestctx.FromContext(c.UserContext())
	if !ok || rc == nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "request context missing")
	}
	return rc, nil
}

func (h *batchHandler) translateBatchError(c *fiber.Ctx, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, batchsvc.ErrUnsupportedEndpoint), errors.Is(err, batchsvc.ErrFilePurposeMismatch):
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	case errors.Is(err, pgx.ErrNoRows):
		return httputil.WriteError(c, fiber.StatusNotFound, "batch not found or cannot transition")
	default:
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
}

func toOpenAIBatch(batch batchsvc.Batch) openAIBatchResponse {
	resp := openAIBatchResponse{
		ID:               batch.ID.String(),
		Object:           "batch",
		Endpoint:         batch.Endpoint,
		Status:           batch.Status,
		CompletionWindow: batch.CompletionWindow,
		CreatedAt:        batch.CreatedAt.Unix(),
		InputFileID:      batch.InputFileID.String(),
		Metadata:         batch.Metadata,
		RequestCounts: openAIBatchCounts{
			Total:     batch.RequestCountTotal,
			Completed: batch.RequestCountCompleted,
			Failed:    batch.RequestCountFailed,
			Cancelled: batch.RequestCountCancelled,
		},
	}
	if batch.InProgressAt != nil {
		ts := batch.InProgressAt.Unix()
		resp.InProgressAt = &ts
	}
	if batch.CompletedAt != nil {
		ts := batch.CompletedAt.Unix()
		resp.CompletedAt = &ts
	}
	if batch.CancelledAt != nil {
		ts := batch.CancelledAt.Unix()
		resp.CancelledAt = &ts
	}
	if batch.FailedAt != nil {
		ts := batch.FailedAt.Unix()
		resp.FailedAt = &ts
	}
	if batch.FinalizingAt != nil {
		ts := batch.FinalizingAt.Unix()
		resp.FinalizingAt = &ts
	}
	if batch.ExpiresAt != nil {
		ts := batch.ExpiresAt.Unix()
		resp.ExpiresAt = &ts
	}
	if batch.ResultFileID != nil {
		id := batch.ResultFileID.String()
		resp.OutputFileID = &id
	}
	if batch.ErrorFileID != nil {
		id := batch.ErrorFileID.String()
		resp.ErrorFileID = &id
	}
	return resp
}
