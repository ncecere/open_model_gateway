package batchworker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

// Worker processes queued batch jobs and executes the corresponding /v1/* calls.
type Worker struct {
	container    *app.Container
	executor     *executor.Executor
	logger       *slog.Logger
	pollInterval time.Duration
}

// New returns a worker instance bound to the provided container + executor.
func New(container *app.Container, exec *executor.Executor) *Worker {
	interval := 2 * time.Second
	return &Worker{
		container:    container,
		executor:     exec,
		logger:       slog.Default(),
		pollInterval: interval,
	}
}

// Run begins polling for queued batches until the context is canceled.
func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.container == nil || w.container.Batches == nil || w.executor == nil {
		return
	}

	for {
		if ctx.Err() != nil {
			return
		}

		handled, err := w.processNextBatch(ctx)
		if err != nil {
			w.logger.Error("batch worker: process batch", slog.String("error", err.Error()))
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}

		if !handled {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.pollInterval):
			}
		}
	}
}

func (w *Worker) processNextBatch(ctx context.Context) (bool, error) {
	batch, err := w.container.Batches.ClaimNextBatch(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	w.logger.Info("batch worker: claimed batch", slog.String("batch_id", batch.ID.String()), slog.String("endpoint", batch.Endpoint))
	if err := w.processBatch(ctx, batch); err != nil {
		return true, err
	}
	return true, nil
}

func (w *Worker) processBatch(ctx context.Context, batch batchsvc.Batch) error {
	rc, err := w.buildRequestContext(ctx, batch)
	if err != nil {
		w.logger.Error("batch worker: build request context", slog.String("batch_id", batch.ID.String()), slog.String("error", err.Error()))
		return w.failEntireBatch(ctx, batch, "context_error", err.Error())
	}

	tracePrefix := fmt.Sprintf("batch_%s_", batch.ID.String())
	var (
		completed int
		failed    int
	)

	writer := newResultWriter(w.container.Files, batch, fileTTL(batch))

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		itemRow, err := w.container.Batches.ClaimNextItem(ctx, batch.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				break
			}
			return err
		}

		itemID, err := fromPgUUID(itemRow.ID)
		if err != nil {
			return err
		}
		item := batchItem{
			ID:       itemID,
			Index:    itemRow.ItemIndex,
			CustomID: strings.TrimSpace(itemRow.CustomID.String),
			Input:    itemRow.Input,
		}

		respPayload, errPayload := w.executeItem(ctx, batch, rc, tracePrefix, item)
		if errPayload == nil {
			if err := w.container.Batches.CompleteItem(ctx, item.ID, respPayload); err != nil {
				return err
			}
			if err := writer.AppendSuccess(item, respPayload); err != nil {
				return err
			}
			completed++
		} else {
			if err := w.container.Batches.FailItem(ctx, item.ID, errPayload); err != nil {
				return err
			}
			if err := writer.AppendError(item, errPayload); err != nil {
				return err
			}
			failed++
		}
	}

	if err := w.container.Batches.IncrementCounts(ctx, batch.ID, completed, failed, 0); err != nil {
		return err
	}

	resultFileID, errorFileID, err := writer.Flush(ctx)
	if err != nil {
		return err
	}

	status := "completed"
	if completed == 0 && failed > 0 {
		status = "failed"
	}
	_, err = w.container.Batches.FinalizeBatch(ctx, batch.ID, status, resultFileID, errorFileID)
	return err
}

func (w *Worker) executeItem(ctx context.Context, batch batchsvc.Batch, rc *requestctx.Context, tracePrefix string, item batchItem) ([]byte, []byte) {
	switch batch.Endpoint {
	case "/v1/chat/completions":
		return w.runChatItem(ctx, rc, tracePrefix, item)
	case "/v1/embeddings":
		return w.runEmbeddingItem(ctx, rc, tracePrefix, item)
	case "/v1/images/generations":
		return w.runImageItem(ctx, rc, tracePrefix, item)
	default:
		errPayload := encodeErrorPayload("unsupported_endpoint", fmt.Sprintf("endpoint %s not supported yet", batch.Endpoint))
		return nil, errPayload
	}
}

func (w *Worker) runChatItem(ctx context.Context, rc *requestctx.Context, tracePrefix string, item batchItem) ([]byte, []byte) {
	input, errPayload := decodeBatchRequest(item, "/v1/chat/completions")
	if errPayload != nil {
		return nil, errPayload
	}

	var body chatBody
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return nil, encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid chat body: %v", err))
	}
	alias := strings.TrimSpace(body.Model)
	if alias == "" {
		return nil, encodeErrorPayload("invalid_request_error", "model is required")
	}
	if body.Stream {
		return nil, encodeErrorPayload("invalid_request_error", "streaming is not supported in batches")
	}

	stop, err := parseStop(body.Stop)
	if err != nil {
		return nil, encodeErrorPayload("invalid_request_error", "invalid stop value")
	}

	messages := make([]models.ChatMessage, 0, len(body.Messages))
	for _, msg := range body.Messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if role == "" {
			role = "user"
		}
		messages = append(messages, models.ChatMessage{
			Role:    role,
			Content: msg.Content,
			Name:    msg.Name,
		})
	}
	if len(messages) == 0 {
		return nil, encodeErrorPayload("invalid_request_error", "messages are required")
	}

	if !w.container.IsModelAllowed(rc.TenantID, alias) {
		return nil, encodeErrorPayload("permission_error", "model not enabled for tenant")
	}

	req := models.ChatRequest{
		Messages:    messages,
		Temperature: body.Temperature,
		TopP:        body.TopP,
		MaxTokens:   body.MaxTokens,
		Stop:        stop,
	}

	callCtx := requestctx.WithContext(ctx, rc)
	traceID := fmt.Sprintf("%s%d", tracePrefix, item.Index)
	result, err := w.executor.Chat(callCtx, rc, alias, req, traceID)
	if err != nil {
		status, msg, ok := executor.AsAPIError(err)
		if !ok {
			return nil, encodeErrorPayload("provider_error", err.Error())
		}
		return nil, encodeErrorPayload(mapStatusToCode(status), msg)
	}
	record := usagepipeline.Record{
		Context:   rc,
		Alias:     alias,
		Provider:  result.Provider,
		Usage:     result.Response.Usage,
		Latency:   result.Latency,
		Status:    fiber.StatusOK,
		TraceID:   traceID,
		Timestamp: time.Now().UTC(),
		Success:   true,
	}
	if _, err := w.container.UsageLogger.Record(callCtx, record); err != nil {
		return nil, encodeErrorPayload("usage_error", err.Error())
	}

	response := convertChatResponse(result.Response, alias)
	data, err := json.Marshal(response)
	if err != nil {
		return nil, encodeErrorPayload("serialization_error", err.Error())
	}
	return data, nil
}

func (w *Worker) runEmbeddingItem(ctx context.Context, rc *requestctx.Context, tracePrefix string, item batchItem) ([]byte, []byte) {
	input, errPayload := decodeBatchRequest(item, "/v1/embeddings")
	if errPayload != nil {
		return nil, errPayload
	}

	var body openAIEmbeddingRequest
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return nil, encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid embeddings body: %v", err))
	}
	body.Model = strings.TrimSpace(body.Model)
	if body.Model == "" {
		return nil, encodeErrorPayload("invalid_request_error", "model is required")
	}
	values, err := parseEmbeddingInput(body.Input)
	if err != nil {
		return nil, encodeErrorPayload("invalid_request_error", "invalid input field")
	}
	if len(values) == 0 {
		return nil, encodeErrorPayload("invalid_request_error", "input is required")
	}

	if !w.container.IsModelAllowed(rc.TenantID, body.Model) {
		return nil, encodeErrorPayload("permission_error", "model not enabled for tenant")
	}

	routes := w.container.Engine.SelectRoutes(body.Model)
	if len(routes) == 0 {
		return nil, encodeErrorPayload("service_unavailable", "no backend available for model")
	}

	callCtx := requestctx.WithContext(ctx, rc)
	traceID := fmt.Sprintf("%s%d", tracePrefix, item.Index)

	status, err := w.container.UsageLogger.CheckBudget(callCtx, rc, time.Now().UTC())
	if err != nil {
		return nil, encodeErrorPayload("budget_error", err.Error())
	}
	if status.Exceeded {
		_, _ = w.container.UsageLogger.Record(callCtx, usagepipeline.Record{
			Context:   rc,
			Alias:     body.Model,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
		return nil, encodeErrorPayload("budget_exceeded", "tenant budget exceeded")
	}

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := w.container.AcquireRateLimits(callCtx, body.Model)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return nil, encodeErrorPayload("rate_limit_error", "rate limit exceeded")
		}
		return nil, encodeErrorPayload("rate_limit_error", err.Error())
	}
	defer release()

	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration

	for _, route := range routes {
		if route.Embedding == nil {
			continue
		}
		lastRoute = route
		modelReq := models.EmbeddingsRequest{
			Model: route.ResolveDeployment(),
			Input: values,
		}
		start := time.Now()
		resp, err := route.Embedding.Embed(callCtx, modelReq)
		if err != nil {
			w.container.Engine.ReportFailure(body.Model, route)
			lastErr = err
			lastLatency = time.Since(start)
			continue
		}

		if tokens := int(resp.Usage.TotalTokens); tokens > 0 {
			if err := w.container.RateLimiter.TokenAllowance(callCtx, keyKey, tokens, keyCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return nil, encodeErrorPayload("rate_limit_error", "token limit exceeded")
				}
				return nil, encodeErrorPayload("rate_limit_error", err.Error())
			}
			if err := w.container.RateLimiter.TokenAllowance(callCtx, tenantKey, tokens, tenantCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return nil, encodeErrorPayload("rate_limit_error", "token limit exceeded")
				}
				return nil, encodeErrorPayload("rate_limit_error", err.Error())
			}
		}

		w.container.Engine.ReportSuccess(body.Model, route)
		record := usagepipeline.Record{
			Context:   rc,
			Alias:     body.Model,
			Provider:  route.Provider,
			Usage:     resp.Usage,
			Latency:   time.Since(start),
			Status:    fiber.StatusOK,
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   true,
		}
		if _, err := w.container.UsageLogger.Record(callCtx, record); err != nil {
			return nil, encodeErrorPayload("usage_error", err.Error())
		}

		payload := convertEmbeddingResponse(resp, body.Model)
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, encodeErrorPayload("serialization_error", err.Error())
		}
		return data, nil
	}

	if lastRoute.Provider != "" {
		_, _ = w.container.UsageLogger.Record(callCtx, usagepipeline.Record{
			Context:   rc,
			Alias:     body.Model,
			Provider:  lastRoute.Provider,
			Status:    fiber.StatusBadGateway,
			ErrorCode: errMessage(lastErr),
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
			Latency:   lastLatency,
		})
	}
	return nil, encodeErrorPayload("provider_error", errMessage(lastErr))
}

func (w *Worker) runImageItem(ctx context.Context, rc *requestctx.Context, tracePrefix string, item batchItem) ([]byte, []byte) {
	input, errPayload := decodeBatchRequest(item, "/v1/images/generations")
	if errPayload != nil {
		return nil, errPayload
	}

	var body openAIImageRequest
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return nil, encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid image body: %v", err))
	}
	alias := strings.TrimSpace(body.Model)
	body.Prompt = strings.TrimSpace(body.Prompt)
	if alias == "" {
		return nil, encodeErrorPayload("invalid_request_error", "model is required")
	}
	if body.Prompt == "" {
		return nil, encodeErrorPayload("invalid_request_error", "prompt is required")
	}

	n := body.N
	if n <= 0 {
		n = 1
	}
	if n > 10 {
		return nil, encodeErrorPayload("invalid_request_error", "n must be between 1 and 10")
	}

	if !w.container.IsModelAllowed(rc.TenantID, alias) {
		return nil, encodeErrorPayload("permission_error", "model not enabled for tenant")
	}

	routes := w.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return nil, encodeErrorPayload("service_unavailable", "no backend available for model")
	}

	callCtx := requestctx.WithContext(ctx, rc)
	traceID := fmt.Sprintf("%s%d", tracePrefix, item.Index)

	status, err := w.container.UsageLogger.CheckBudget(callCtx, rc, time.Now().UTC())
	if err != nil {
		return nil, encodeErrorPayload("budget_error", err.Error())
	}
	if status.Exceeded {
		_, _ = w.container.UsageLogger.Record(callCtx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  "budget",
			Status:    fiber.StatusForbidden,
			ErrorCode: "budget_exceeded",
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
		return nil, encodeErrorPayload("budget_exceeded", "tenant budget exceeded")
	}

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := w.container.AcquireRateLimits(callCtx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return nil, encodeErrorPayload("rate_limit_error", "rate limit exceeded")
		}
		return nil, encodeErrorPayload("rate_limit_error", err.Error())
	}
	defer release()

	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration

	for _, route := range routes {
		if route.Image == nil {
			continue
		}
		lastRoute = route

		modelReq := models.ImageRequest{
			Model:          route.ResolveDeployment(),
			Prompt:         body.Prompt,
			Size:           body.Size,
			ResponseFormat: body.ResponseFormat,
			Quality:        body.Quality,
			N:              n,
			User:           body.User,
			Background:     body.Background,
			Style:          body.Style,
		}

		start := time.Now()
		resp, err := route.Image.Generate(callCtx, modelReq)
		if err != nil {
			w.container.Engine.ReportFailure(alias, route)
			lastErr = err
			lastLatency = time.Since(start)
			continue
		}
		w.container.Engine.ReportSuccess(alias, route)

		if tokens := int(resp.Usage.TotalTokens); tokens > 0 {
			if err := w.container.RateLimiter.TokenAllowance(callCtx, keyKey, tokens, keyCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return nil, encodeErrorPayload("rate_limit_error", "token limit exceeded")
				}
				return nil, encodeErrorPayload("rate_limit_error", err.Error())
			}
			if err := w.container.RateLimiter.TokenAllowance(callCtx, tenantKey, tokens, tenantCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return nil, encodeErrorPayload("rate_limit_error", "token limit exceeded")
				}
				return nil, encodeErrorPayload("rate_limit_error", err.Error())
			}
		}

		var override *int64
		if priceStr := route.Metadata["price_image_cents"]; priceStr != "" {
			if cents, err := strconv.ParseInt(priceStr, 10, 64); err == nil {
				override = new(int64)
				*override = cents
			}
		}

		record := usagepipeline.Record{
			Context:           rc,
			Alias:             alias,
			Provider:          route.Provider,
			Usage:             resp.Usage,
			Latency:           time.Since(start),
			Status:            fiber.StatusOK,
			TraceID:           traceID,
			Timestamp:         time.Now().UTC(),
			Success:           true,
			OverrideCostCents: override,
		}
		if _, err := w.container.UsageLogger.Record(callCtx, record); err != nil {
			return nil, encodeErrorPayload("usage_error", err.Error())
		}

		response := convertImageResponse(resp)
		data, err := json.Marshal(response)
		if err != nil {
			return nil, encodeErrorPayload("serialization_error", err.Error())
		}
		return data, nil
	}

	if lastRoute.Provider != "" {
		_, _ = w.container.UsageLogger.Record(callCtx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  lastRoute.Provider,
			Status:    fiber.StatusBadGateway,
			ErrorCode: errMessage(lastErr),
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
			Latency:   lastLatency,
		})
	}
	return nil, encodeErrorPayload("provider_error", errMessage(lastErr))
}

func (w *Worker) buildRequestContext(ctx context.Context, batch batchsvc.Batch) (*requestctx.Context, error) {
	slog.Info("worker building request context", "batch_id", batch.ID.String(), "api_key_id", batch.APIKeyID.String())
	keyRow, err := w.container.Queries.GetAPIKeyByID(ctx, toPgUUID(batch.APIKeyID))
	if err != nil {
		return nil, err
	}
	if keyRow.RevokedAt.Valid {
		return nil, fmt.Errorf("api key revoked")
	}

	tenantRow, err := w.container.Queries.GetTenantByID(ctx, keyRow.TenantID)
	if err != nil {
		return nil, err
	}
	if tenantRow.Status != db.TenantStatusActive {
		return nil, fmt.Errorf("tenant is not active")
	}

	return app.BuildRequestContext(ctx, w.container, keyRow)
}

func (w *Worker) failEntireBatch(ctx context.Context, batch batchsvc.Batch, code, message string) error {
	writer := newResultWriter(w.container.Files, batch, fileTTL(batch))
	var failed int
	for {
		itemRow, err := w.container.Batches.ClaimNextItem(ctx, batch.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				break
			}
			return err
		}
		itemID, err := fromPgUUID(itemRow.ID)
		if err != nil {
			return err
		}
		payload := encodeErrorPayload(code, message)
		if err := w.container.Batches.FailItem(ctx, itemID, payload); err != nil {
			return err
		}
		failed++
		_ = writer.AppendError(batchItem{
			ID:       itemID,
			CustomID: strings.TrimSpace(itemRow.CustomID.String),
			Index:    itemRow.ItemIndex,
		}, payload)
	}

	if failed > 0 {
		if err := w.container.Batches.IncrementCounts(ctx, batch.ID, 0, failed, 0); err != nil {
			return err
		}
	}

	resultFileID, errorFileID, err := writer.Flush(ctx)
	if err != nil {
		return err
	}

	_, err = w.container.Batches.FinalizeBatch(ctx, batch.ID, "failed", resultFileID, errorFileID)
	return err
}

type batchItem struct {
	ID       uuid.UUID
	Index    int64
	CustomID string
	Input    []byte
}

type batchRequest struct {
	CustomID string          `json:"custom_id"`
	Method   string          `json:"method"`
	URL      string          `json:"url"`
	Body     json.RawMessage `json:"body"`
	Headers  json.RawMessage `json:"headers"`
}

type chatBody struct {
	Model       string          `json:"model"`
	Messages    []chatMessage   `json:"messages"`
	Temperature *float32        `json:"temperature"`
	TopP        *float32        `json:"top_p"`
	MaxTokens   *int32          `json:"max_tokens"`
	Stream      bool            `json:"stream"`
	Stop        json.RawMessage `json:"stop"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type openAIEmbeddingRequest struct {
	Model string          `json:"model"`
	Input json.RawMessage `json:"input"`
}

type openAIImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Quality        string `json:"quality,omitempty"`
	N              int    `json:"n,omitempty"`
	User           string `json:"user,omitempty"`
	Background     string `json:"background,omitempty"`
	Style          string `json:"style,omitempty"`
}

func parseStop(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return []string{single}, nil
	}
	var multi []string
	if err := json.Unmarshal(raw, &multi); err == nil {
		return multi, nil
	}
	return nil, errors.New("invalid stop value")
}

type resultWriter struct {
	files    *filesvc.Service
	batch    batchsvc.Batch
	ttl      time.Duration
	success  bytes.Buffer
	failures bytes.Buffer
}

func newResultWriter(files *filesvc.Service, batch batchsvc.Batch, ttl time.Duration) *resultWriter {
	return &resultWriter{
		files: files,
		batch: batch,
		ttl:   ttl,
	}
}

func (w *resultWriter) AppendSuccess(item batchItem, payload []byte) error {
	return w.appendLine(&w.success, item, payload, nil)
}

func (w *resultWriter) AppendError(item batchItem, payload []byte) error {
	return w.appendLine(&w.failures, item, nil, payload)
}

func (w *resultWriter) appendLine(buf *bytes.Buffer, item batchItem, response, errPayload []byte) error {
	if buf == nil {
		return nil
	}
	entry := map[string]any{
		"id": item.ID.String(),
	}
	if item.CustomID != "" {
		entry["custom_id"] = item.CustomID
	}
	if len(response) > 0 {
		entry["response"] = json.RawMessage(response)
	}
	if len(errPayload) > 0 {
		entry["error"] = json.RawMessage(errPayload)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := buf.Write(data); err != nil {
		return err
	}
	return buf.WriteByte('\n')
}

func (w *resultWriter) Flush(ctx context.Context) (*uuid.UUID, *uuid.UUID, error) {
	var resultID *uuid.UUID
	var errorID *uuid.UUID

	if w.files == nil {
		return nil, nil, nil
	}

	if w.success.Len() > 0 {
		id, err := w.persist(ctx, w.success.Bytes(), fmt.Sprintf("batch_%s_output.jsonl", w.batch.ID))
		if err != nil {
			return nil, nil, err
		}
		resultID = id
	}
	if w.failures.Len() > 0 {
		id, err := w.persist(ctx, w.failures.Bytes(), fmt.Sprintf("batch_%s_errors.jsonl", w.batch.ID))
		if err != nil {
			return nil, nil, err
		}
		errorID = id
	}

	return resultID, errorID, nil
}

func (w *resultWriter) persist(ctx context.Context, data []byte, filename string) (*uuid.UUID, error) {
	if w.files == nil || len(data) == 0 {
		return nil, nil
	}
	reader := bytes.NewReader(data)
	rec, err := w.files.Upload(ctx, filesvc.UploadParams{
		TenantID:    w.batch.TenantID,
		Filename:    filename,
		Purpose:     filesvc.PurposeBatch,
		ContentType: "application/x-ndjson",
		ContentLen:  int64(len(data)),
		TTL:         w.ttl,
		Reader:      reader,
	})
	if err != nil {
		return nil, err
	}
	return &rec.ID, nil
}

func fileTTL(batch batchsvc.Batch) time.Duration {
	if batch.ExpiresAt == nil {
		return 0
	}
	ttl := time.Until(*batch.ExpiresAt)
	if ttl < 0 {
		return 0
	}
	return ttl
}

type errorPayload struct {
	Error openAIError `json:"error"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

func encodeErrorPayload(code, message string) []byte {
	payload := errorPayload{
		Error: openAIError{
			Message: message,
			Type:    "batch_error",
			Code:    code,
		},
	}
	data, _ := json.Marshal(payload)
	return data
}

func mapStatusToCode(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "invalid_request_error"
	case fiber.StatusForbidden:
		return "permission_error"
	case fiber.StatusTooManyRequests:
		return "rate_limit_error"
	case fiber.StatusServiceUnavailable:
		return "service_unavailable"
	default:
		return "provider_error"
	}
}

func convertChatResponse(resp models.ChatResponse, alias string) openAIChatResponse {
	choices := make([]openAIChatChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		choices = append(choices, openAIChatChoice{
			Index: choice.Index,
			Message: openAIChatMessage{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
			FinishReason: choice.FinishReason,
		})
	}
	return openAIChatResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: resp.Created.Unix(),
		Model:   alias,
		Choices: choices,
		Usage: openAIUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}

func convertEmbeddingResponse(resp models.EmbeddingsResponse, alias string) openAIEmbeddingResponse {
	data := make([]openAIEmbedding, 0, len(resp.Embeddings))
	for _, emb := range resp.Embeddings {
		data = append(data, openAIEmbedding{
			Index:     emb.Index,
			Embedding: emb.Vector,
			Object:    "embedding",
		})
	}
	return openAIEmbeddingResponse{
		Object: "list",
		Model:  alias,
		Data:   data,
		Usage: openAIUsage{
			PromptTokens: resp.Usage.PromptTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}
}

func convertImageResponse(resp models.ImageResponse) openAIImageResponse {
	data := make([]openAIImageData, 0, len(resp.Data))
	for _, item := range resp.Data {
		data = append(data, openAIImageData{
			B64JSON:       item.B64JSON,
			URL:           item.URL,
			RevisedPrompt: item.RevisedPrompt,
		})
	}
	created := resp.Created.Unix()
	if created < 0 {
		created = 0
	}
	return openAIImageResponse{
		Created: created,
		Data:    data,
	}
}

func parseEmbeddingInput(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, errors.New("input is required")
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return []string{str}, nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("input must be string or array of strings")
}

func decodeBatchRequest(item batchItem, expectedPath string) (batchRequest, []byte) {
	var req batchRequest
	if err := json.Unmarshal(item.Input, &req); err != nil {
		return batchRequest{}, encodeErrorPayload("invalid_batch_input", fmt.Sprintf("invalid JSON payload: %v", err))
	}
	method := strings.TrimSpace(req.Method)
	if method == "" {
		method = "POST"
	}
	if !strings.EqualFold(method, "POST") {
		return batchRequest{}, encodeErrorPayload("invalid_method", "batch entries must use POST")
	}
	path := strings.TrimSpace(req.URL)
	if path == "" {
		path = expectedPath
	}
	if path != expectedPath {
		return batchRequest{}, encodeErrorPayload("invalid_endpoint", "batch URL mismatch")
	}
	if len(req.Body) == 0 {
		return batchRequest{}, encodeErrorPayload("invalid_request_error", "body is required")
	}
	return req, nil
}

func errMessage(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}

type openAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openAIChatChoice `json:"choices"`
	Usage   openAIUsage        `json:"usage"`
}

type openAIChatChoice struct {
	Index        int               `json:"index"`
	Message      openAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

type openAIEmbedding struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
	Object    string    `json:"object"`
}

type openAIEmbeddingResponse struct {
	Object string            `json:"object"`
	Model  string            `json:"model"`
	Data   []openAIEmbedding `json:"data"`
	Usage  openAIUsage       `json:"usage"`
}

type openAIImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type openAIImageResponse struct {
	Created int64             `json:"created"`
	Data    []openAIImageData `json:"data"`
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	var out pgtype.UUID
	if id == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	copy(out.Bytes[:], id[:])
	out.Valid = true
	return out
}

func fromPgUUID(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, fmt.Errorf("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}
