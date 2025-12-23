package batchworker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (w *Worker) runImageItem(ctx context.Context, rc *requestctx.Context, traceID string, item batchItem) itemOutcome {
	input, errPayload := decodeBatchRequest(item, "/v1/images/generations")
	if errPayload != nil {
		return itemOutcome{statusCode: fiber.StatusBadRequest, requestID: traceID, errPayload: errPayload}
	}

	var body openAIImageRequest
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid image body: %v", err)),
		}
	}
	alias := strings.TrimSpace(body.Model)
	body.Prompt = strings.TrimSpace(body.Prompt)
	if alias == "" {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "model is required"),
		}
	}
	if body.Prompt == "" {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "prompt is required"),
		}
	}

	n := body.N
	if n <= 0 {
		n = 1
	}
	if n > 10 {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "n must be between 1 and 10"),
		}
	}

	if !w.container.IsModelAllowed(rc.TenantID, alias) {
		return itemOutcome{
			statusCode: fiber.StatusForbidden,
			requestID:  traceID,
			errPayload: encodeErrorPayload("permission_error", "model not enabled for tenant"),
		}
	}

	req := models.ImageRequest{
		Model:          alias,
		Prompt:         body.Prompt,
		Size:           body.Size,
		ResponseFormat: body.ResponseFormat,
		Quality:        body.Quality,
		N:              n,
		User:           body.User,
		Background:     body.Background,
		Style:          body.Style,
	}

	return w.executeImageOperation(ctx, rc, traceID, alias, newImageGenerationOperationConfig(req))
}

func (w *Worker) runImageEditItem(ctx context.Context, rc *requestctx.Context, traceID string, item batchItem) itemOutcome {
	input, errPayload := decodeBatchRequest(item, "/v1/images/edits")
	if errPayload != nil {
		return itemOutcome{statusCode: fiber.StatusBadRequest, requestID: traceID, errPayload: errPayload}
	}

	var body openAIImageEditBatchRequest
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid image edit body: %v", err)),
		}
	}

	alias := strings.TrimSpace(body.Model)
	prompt := strings.TrimSpace(body.Prompt)
	if alias == "" {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "model is required"),
		}
	}
	if prompt == "" {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "prompt is required"),
		}
	}
	if !w.container.IsModelAllowed(rc.TenantID, alias) {
		return itemOutcome{
			statusCode: fiber.StatusForbidden,
			requestID:  traceID,
			errPayload: encodeErrorPayload("permission_error", "model not enabled for tenant"),
		}
	}

	imageRefs, err := parseBatchFileRefs(body.Image, body.Images)
	if err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", err.Error()),
		}
	}
	if len(imageRefs) == 0 {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "at least one image reference is required"),
		}
	}

	images, errPayload := w.loadBatchImageInputs(ctx, rc, imageRefs, maxBatchEditImages)
	if errPayload != nil {
		return itemOutcome{statusCode: fiber.StatusBadRequest, requestID: traceID, errPayload: errPayload}
	}

	maskRefs, err := parseBatchFileRefs(body.Mask, body.Masks)
	if err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", err.Error()),
		}
	}
	var maskInput *models.ImageInput
	if len(maskRefs) > 0 {
		maskImages, maskErr := w.loadBatchImageInputs(ctx, rc, maskRefs, 1)
		if maskErr != nil {
			return itemOutcome{statusCode: fiber.StatusBadRequest, requestID: traceID, errPayload: maskErr}
		}
		mask := maskImages[0]
		maskInput = &mask
	}

	n := body.N
	if n <= 0 {
		n = 1
	}
	if n > 10 {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "n must be between 1 and 10"),
		}
	}

	baseReq := models.ImageEditRequest{
		Model:          alias,
		Prompt:         prompt,
		Images:         images,
		Size:           body.Size,
		ResponseFormat: body.ResponseFormat,
		Quality:        body.Quality,
		Background:     body.Background,
		Style:          body.Style,
		N:              n,
		User:           body.User,
	}
	if maskInput != nil {
		baseReq.Mask = maskInput
	}

	return w.executeImageOperation(ctx, rc, traceID, alias, newImageEditOperationConfig(baseReq))
}

func (w *Worker) runImageVariationItem(ctx context.Context, rc *requestctx.Context, traceID string, item batchItem) itemOutcome {
	input, errPayload := decodeBatchRequest(item, "/v1/images/variations")
	if errPayload != nil {
		return itemOutcome{statusCode: fiber.StatusBadRequest, requestID: traceID, errPayload: errPayload}
	}

	var body openAIImageVariationBatchRequest
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid image variation body: %v", err)),
		}
	}

	alias := strings.TrimSpace(body.Model)
	if alias == "" {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "model is required"),
		}
	}

	imageRefs, err := parseBatchFileRefs(body.Image, body.Images)
	if err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", err.Error()),
		}
	}
	if len(imageRefs) == 0 {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "image reference is required"),
		}
	}
	if len(imageRefs) > 1 {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "only one image reference is supported for variations"),
		}
	}

	images, loadErr := w.loadBatchImageInputs(ctx, rc, imageRefs, 1)
	if loadErr != nil {
		return itemOutcome{statusCode: fiber.StatusBadRequest, requestID: traceID, errPayload: loadErr}
	}
	baseImage := images[0]

	n := body.N
	if n <= 0 {
		n = 1
	}
	if n > 10 {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "n must be between 1 and 10"),
		}
	}

	baseReq := models.ImageVariationRequest{
		Model:          alias,
		Image:          baseImage,
		Size:           body.Size,
		ResponseFormat: body.ResponseFormat,
		Quality:        body.Quality,
		Background:     body.Background,
		Style:          body.Style,
		N:              n,
		User:           body.User,
	}

	return w.executeImageOperation(ctx, rc, traceID, alias, newImageVariationOperationConfig(baseReq))
}

func (w *Worker) loadBatchImageInputs(ctx context.Context, rc *requestctx.Context, refs []string, limit int) ([]models.ImageInput, []byte) {
	if limit > 0 && len(refs) > limit {
		return nil, encodeErrorPayload("invalid_request_error", fmt.Sprintf("a maximum of %d files are supported", limit))
	}
	if len(refs) == 0 {
		return nil, encodeErrorPayload("invalid_request_error", "file reference is required")
	}
	if w.container.Files == nil {
		return nil, encodeErrorPayload("invalid_request_error", "file storage is not configured")
	}
	if rc == nil {
		return nil, encodeErrorPayload("invalid_request_error", "request context missing")
	}

	results := make([]models.ImageInput, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return nil, encodeErrorPayload("invalid_request_error", "empty file reference provided")
		}
		fileID, err := uuid.Parse(ref)
		if err != nil {
			return nil, encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid file id %q", ref))
		}
		reader, record, err := w.container.Files.Open(ctx, rc.TenantID, fileID)
		if err != nil {
			return nil, encodeErrorPayload("invalid_request_error", fmt.Sprintf("unable to load file %q", ref))
		}
		data, readErr := io.ReadAll(reader)
		reader.Close()
		if readErr != nil {
			return nil, encodeErrorPayload("invalid_request_error", fmt.Sprintf("failed to read file %q", ref))
		}
		if int64(len(data)) > maxBatchImageBytes {
			return nil, encodeErrorPayload("invalid_request_error", fmt.Sprintf("file %q exceeds %d MB limit", ref, maxBatchImageBytes/1024/1024))
		}
		contentType := strings.TrimSpace(record.ContentType)
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		if contentType == "" || !strings.HasPrefix(strings.ToLower(contentType), "image/") {
			return nil, encodeErrorPayload("invalid_request_error", fmt.Sprintf("file %q must be an image/* content type", ref))
		}
		results = append(results, models.ImageInput{
			Data:        data,
			Filename:    record.Filename,
			ContentType: contentType,
		})
	}
	return results, nil
}

func parseImageOverrideCost(metadata map[string]string, op imageOperationType) *int64 {
	if metadata == nil {
		return nil
	}
	var keys []string
	switch op {
	case imageOperationEdit:
		keys = []string{"price_image_edit_cents"}
	case imageOperationVariation:
		keys = []string{"price_image_variation_cents"}
	default:
		keys = []string{"price_image_cents"}
	}
	keys = append(keys, "price_image_cents")
	for _, key := range keys {
		if price := strings.TrimSpace(metadata[key]); price != "" {
			if cents, err := strconv.ParseInt(price, 10, 64); err == nil {
				return &cents
			}
		}
	}
	return nil
}

func cloneImageInputs(inputs []models.ImageInput) []models.ImageInput {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]models.ImageInput, len(inputs))
	copy(out, inputs)
	return out
}

const (
	maxBatchImageBytes = 4 * 1024 * 1024
	maxBatchEditImages = 16
)
