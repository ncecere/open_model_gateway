package batchworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	"sync"
	"time"
)

type resultWriter struct {
	files    *filesvc.Service
	batch    batchsvc.Batch
	ttl      time.Duration
	success  bytes.Buffer
	failures bytes.Buffer
	mu       sync.Mutex
}

type batchResultLine struct {
	ID       string              `json:"id"`
	CustomID string              `json:"custom_id,omitempty"`
	Response *batchResultPayload `json:"response,omitempty"`
	Error    *batchResultPayload `json:"error,omitempty"`
}

type batchResultPayload struct {
	StatusCode int             `json:"status_code"`
	RequestID  string          `json:"request_id"`
	Body       json.RawMessage `json:"body"`
}

func newResultWriter(files *filesvc.Service, batch batchsvc.Batch, ttl time.Duration) *resultWriter {
	return &resultWriter{
		files: files,
		batch: batch,
		ttl:   ttl,
	}
}

func (w *resultWriter) AppendSuccess(item batchItem, statusCode int, requestID string, payload []byte) error {
	line := batchResultLine{
		ID: item.ID.String(),
		Response: &batchResultPayload{
			StatusCode: statusCode,
			RequestID:  requestID,
			Body:       json.RawMessage(payload),
		},
	}
	if item.CustomID != "" {
		line.CustomID = item.CustomID
	}
	return w.appendLine(&w.success, line)
}

func (w *resultWriter) AppendError(item batchItem, statusCode int, requestID string, payload []byte) error {
	line := batchResultLine{
		ID: item.ID.String(),
		Error: &batchResultPayload{
			StatusCode: statusCode,
			RequestID:  requestID,
			Body:       json.RawMessage(payload),
		},
	}
	if item.CustomID != "" {
		line.CustomID = item.CustomID
	}
	return w.appendLine(&w.failures, line)
}

func (w *resultWriter) appendLine(buf *bytes.Buffer, line batchResultLine) error {
	if buf == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	data, err := json.Marshal(line)
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
		msg := openAIChatMessage{
			Role:      choice.Message.Role,
			Content:   models.MarshalMessageContent(choice.Message),
			Reasoning: choice.Message.Reasoning,
		}
		choices = append(choices, openAIChatChoice{
			Index:        choice.Index,
			Message:      msg,
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
			ReasoningTokens:  resp.Usage.ReasoningTokens,
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

func convertModerationResponse(resp models.ModerationResponse, alias string) openAIModerationResponse {
	results := make([]openAIModerationResult, 0, len(resp.Results))
	for _, item := range resp.Results {
		results = append(results, openAIModerationResult{
			Categories:                item.Categories,
			CategoryAppliedInputTypes: item.CategoryAppliedInputTypes,
			CategoryScores:            item.CategoryScores,
			Flagged:                   item.Flagged,
		})
	}
	return openAIModerationResponse{
		ID:      resp.ID,
		Model:   alias,
		Results: results,
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
