package public

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type openAIHandler struct {
	container          *app.Container
	executor           *executor.Executor
	chatPipeline       *chatPipeline
	chatStreamPipeline *chatStreamPipeline
	imagePipeline      *imagePipeline
	audioPipeline      *audioPipeline
	embeddingPipeline  *embeddingPipeline
	moderationPipeline *moderationPipeline
}

func newOpenAIHandler(container *app.Container) *openAIHandler {
	exec := executor.New(container)
	return &openAIHandler{
		container:          container,
		executor:           exec,
		chatPipeline:       newChatPipeline(container, exec),
		chatStreamPipeline: newChatStreamPipeline(container),
		imagePipeline:      newImagePipeline(container, exec),
		audioPipeline:      newAudioPipeline(container),
		embeddingPipeline:  newEmbeddingPipeline(container, exec),
		moderationPipeline: newModerationPipeline(container, exec),
	}
}

const (
	maxImageUploadBytes = 4 * 1024 * 1024
	maxImageInputs      = 16
)

func (h *openAIHandler) listModels(c *fiber.Ctx) error {
	ctx := c.UserContext()
	var rc *requestctx.Context
	if rctx, ok := requestctx.FromContext(ctx); ok {
		rc = rctx
		if status, err := h.container.UsageLogger.CheckBudget(ctx, rctx, time.Now().UTC()); err == nil {
			httputil.ApplyBudgetHeaders(c, status)
		}
	}

	aliases := h.container.Engine.ListAliases()
	models := make([]openAIModel, 0, len(aliases))
	now := time.Now().Unix()

	for alias, routes := range aliases {
		if len(routes) == 0 {
			continue
		}
		if rc != nil && !h.container.IsModelAllowed(rc.TenantID, alias) {
			continue
		}
		route := routes[0]
		deployment := route.Metadata["deployment"]
		models = append(models, openAIModel{
			ID:         alias,
			Object:     "model",
			OwnedBy:    route.Provider,
			Created:    now,
			Deployment: deployment,
		})
	}

	return c.JSON(openAIModelList{
		Object: "list",
		Data:   models,
	})
}

func (h *openAIHandler) runImageOperation(c *fiber.Ctx, cfg imageOperationConfig) error {
	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	if err := h.imagePipeline.Execute(c, rc, cfg); err != nil {
		return err
	}
	return nil
}

type openAIModel struct {
	ID         string `json:"id"`
	Object     string `json:"object"`
	OwnedBy    string `json:"owned_by"`
	Created    int64  `json:"created"`
	Deployment string `json:"deployment"`
}

type openAIModelList struct {
	Object string        `json:"object"`
	Data   []openAIModel `json:"data"`
}

type openAIChatMessage struct {
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	Name      string          `json:"name,omitempty"`
	Reasoning string          `json:"reasoning,omitempty"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature *float32            `json:"temperature,omitempty"`
	TopP        *float32            `json:"top_p,omitempty"`
	MaxTokens   *int32              `json:"max_tokens,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	StopRaw     json.RawMessage     `json:"stop,omitempty"`
}

type openAIChatChoice struct {
	Index        int               `json:"index"`
	Message      openAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

func convertOpenAIMessages(msgs []openAIChatMessage, label string) ([]models.ChatMessage, error) {
	if label == "" {
		label = "message"
	}
	sanitized := make([]models.ChatMessage, 0, len(msgs))
	for idx, m := range msgs {
		role := strings.ToLower(m.Role)
		if role == "" {
			role = "user"
		}
		textContent, parts, err := models.ParseMessageContent(m.Content)
		if err != nil {
			return nil, fmt.Errorf("invalid content for %s %d: %v", label, idx, err)
		}
		sanitized = append(sanitized, models.ChatMessage{
			Role:         role,
			Content:      textContent,
			ContentParts: parts,
			Name:         m.Name,
		})
	}
	return sanitized, nil
}

type openAIUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	ReasoningTokens  int32 `json:"reasoning_tokens,omitempty"`
	TotalTokens      int32 `json:"total_tokens"`
}

type openAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openAIChatChoice `json:"choices"`
	Usage   openAIUsage        `json:"usage"`
}

type openAIResponsesRequest struct {
	Model             string            `json:"model"`
	Input             json.RawMessage   `json:"input"`
	Instructions      string            `json:"instructions"`
	Temperature       *float32          `json:"temperature,omitempty"`
	TopP              *float32          `json:"top_p,omitempty"`
	MaxOutputTokens   *int32            `json:"max_output_tokens,omitempty"`
	Metadata          map[string]string `json:"metadata"`
	Stream            bool              `json:"stream,omitempty"`
	Tools             json.RawMessage   `json:"tools"`
	ToolChoice        json.RawMessage   `json:"tool_choice"`
	ResponseFormat    json.RawMessage   `json:"response_format"`
	Conversation      json.RawMessage   `json:"conversation"`
	PreviousResponse  string            `json:"previous_response_id"`
	ParallelToolCalls *bool             `json:"parallel_tool_calls,omitempty"`
}

type openAIResponsesPayload struct {
	ID                string                  `json:"id"`
	Object            string                  `json:"object"`
	CreatedAt         int64                   `json:"created_at"`
	Status            string                  `json:"status"`
	Model             string                  `json:"model"`
	Output            []openAIResponsesOutput `json:"output"`
	ParallelToolCalls bool                    `json:"parallel_tool_calls"`
	ToolChoice        string                  `json:"tool_choice"`
	Tools             []interface{}           `json:"tools"`
	Usage             openAIResponsesUsage    `json:"usage"`
	Instructions      string                  `json:"instructions,omitempty"`
	Metadata          map[string]string       `json:"metadata,omitempty"`
}

type openAIResponsesOutput struct {
	Type    string                         `json:"type"`
	ID      string                         `json:"id"`
	Status  string                         `json:"status"`
	Role    string                         `json:"role"`
	Content []openAIResponsesOutputContent `json:"content"`
}

type openAIResponsesOutputContent struct {
	Type        string        `json:"type"`
	Text        string        `json:"text,omitempty"`
	Annotations []interface{} `json:"annotations"`
}

type openAIResponsesUsage struct {
	InputTokens         int32                             `json:"input_tokens"`
	OutputTokens        int32                             `json:"output_tokens"`
	TotalTokens         int32                             `json:"total_tokens"`
	OutputTokensDetails openAIResponsesUsageOutputDetails `json:"output_tokens_details"`
}

type openAIResponsesUsageOutputDetails struct {
	ReasoningTokens int32 `json:"reasoning_tokens"`
}

type openAIResponseOptions struct {
	Instructions      string
	Metadata          map[string]string
	ParallelToolCalls bool
}

func (h *openAIHandler) chatCompletions(c *fiber.Ctx) error {
	var req openAIChatRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if len(req.Messages) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "messages are required")
	}
	stop, err := parseStop(req.StopRaw)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid stop field")
	}

	messages, err := convertOpenAIMessages(req.Messages, "message")
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	if !h.container.IsModelAllowed(rc.TenantID, req.Model) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	traceID := traceIDFromContext(c)
	alias := req.Model
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))

	modelReq := models.ChatRequest{
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stop:        stop,
	}

	if req.Stream {
		return h.chatStreamPipeline.Stream(c, rc, alias, traceID, idempotencyKey, modelReq)
	}

	return h.chatPipeline.Execute(c, rc, alias, traceID, idempotencyKey, modelReq)
}

func (h *openAIHandler) responses(c *fiber.Ctx) error {
	var req openAIResponsesRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	alias := strings.TrimSpace(req.Model)
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if len(req.Input) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "input is required")
	}
	if len(req.Tools) > 0 || len(req.ToolChoice) > 0 || len(req.ResponseFormat) > 0 || len(req.Conversation) > 0 || req.PreviousResponse != "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "tools, conversations, and response_format are not supported")
	}
	if err := validateResponseMetadata(req.Metadata); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	messages, err := buildResponseMessages(req.Instructions, req.Input)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	if !h.container.IsModelAllowed(rc.TenantID, alias) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	traceID := traceIDFromContext(c)
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))

	modelReq := models.ChatRequest{
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxOutputTokens,
	}

	parallel := true
	if req.ParallelToolCalls != nil {
		parallel = *req.ParallelToolCalls
	}
	options := openAIResponseOptions{
		Instructions:      strings.TrimSpace(req.Instructions),
		Metadata:          req.Metadata,
		ParallelToolCalls: parallel,
	}

	if req.Stream {
		return h.chatStreamPipeline.StreamResponses(c, rc, alias, traceID, idempotencyKey, modelReq, options)
	}

	return h.chatPipeline.ExecuteWithConverter(c, rc, alias, traceID, idempotencyKey, modelReq, func(resp models.ChatResponse, alias string) (interface{}, error) {
		return convertResponsesResponse(resp, alias, options), nil
	})
}

type openAIEmbeddingRequest struct {
	Model string          `json:"model"`
	Input json.RawMessage `json:"input"`
}

type openAIModerationRequest struct {
	Model string          `json:"model"`
	Input json.RawMessage `json:"input"`
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

type openAIModerationResponse struct {
	ID      string                   `json:"id"`
	Model   string                   `json:"model"`
	Results []openAIModerationResult `json:"results"`
}

type openAIModerationResult struct {
	Categories                models.ModerationCategories                `json:"categories"`
	CategoryAppliedInputTypes models.ModerationCategoryAppliedInputTypes `json:"category_applied_input_types"`
	CategoryScores            models.ModerationCategoryScores            `json:"category_scores"`
	Flagged                   bool                                       `json:"flagged"`
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

type openAIImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type openAIImageResponse struct {
	Created int64             `json:"created"`
	Data    []openAIImageData `json:"data"`
}

func (h *openAIHandler) moderations(c *fiber.Ctx) error {
	var req openAIModerationRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	alias := strings.TrimSpace(req.Model)
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	inputs, err := parseEmbeddingInput(req.Input)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "input must be string or array of strings")
	}
	if len(inputs) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "input is required")
	}

	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	return h.moderationPipeline.Execute(c, rc, alias, inputs)
}

func (h *openAIHandler) embeddings(c *fiber.Ctx) error {
	var req openAIEmbeddingRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	inputs, err := parseEmbeddingInput(req.Input)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid input field")
	}

	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	modelReq := models.EmbeddingsRequest{
		Model: req.Model,
		Input: inputs,
	}

	return h.embeddingPipeline.Execute(c, rc, req.Model, modelReq)
}

func (h *openAIHandler) imageGenerations(c *fiber.Ctx) error {
	var req openAIImageRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Model = strings.TrimSpace(req.Model)
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if req.Prompt == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "prompt is required")
	}

	n := req.N
	if n <= 0 {
		n = 1
	}
	if n > 10 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "n must be between 1 and 10")
	}

	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	baseReq := req
	return h.runImageOperation(c, imageOperationConfig{
		Alias:          req.Model,
		IdempotencyKey: idempotencyKey,
		Operation:      imageOperationGeneration,
		Builder: func(callCtx context.Context, route providers.Route) (models.ImageResponse, error) {
			modelReq := models.ImageRequest{
				Model:          route.ResolveDeployment(),
				Prompt:         baseReq.Prompt,
				Size:           baseReq.Size,
				ResponseFormat: baseReq.ResponseFormat,
				Quality:        baseReq.Quality,
				N:              n,
				User:           baseReq.User,
				Background:     baseReq.Background,
				Style:          baseReq.Style,
			}
			return route.Image.Generate(callCtx, modelReq)
		},
	})
}

func (h *openAIHandler) imageEdits(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "multipart form required")
	}
	model := strings.TrimSpace(c.FormValue("model"))
	prompt := strings.TrimSpace(c.FormValue("prompt"))
	if model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if prompt == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "prompt is required")
	}
	imageHeaders := gatherFormFiles(form, "image", "image[]")
	if len(imageHeaders) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "at least one image is required")
	}
	if len(imageHeaders) > maxImageInputs {
		return httputil.WriteError(c, fiber.StatusBadRequest, "a maximum of 16 images are supported")
	}
	images := make([]models.ImageInput, 0, len(imageHeaders))
	for _, fh := range imageHeaders {
		input, err := loadImageInput(fh)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "failed to read image upload")
		}
		images = append(images, input)
	}
	var maskInput *models.ImageInput
	maskHeaders := gatherFormFiles(form, "mask", "mask[]")
	if len(maskHeaders) > 0 {
		if len(maskHeaders) > 1 {
			return httputil.WriteError(c, fiber.StatusBadRequest, "only one mask file is supported")
		}
		mask, err := loadImageInput(maskHeaders[0])
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "failed to read mask upload")
		}
		maskInput = &mask
	}
	n, err := parseImageCount(c.FormValue("n"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	baseReq := models.ImageEditRequest{
		Model:          model,
		Prompt:         prompt,
		Images:         images,
		Mask:           maskInput,
		Size:           strings.TrimSpace(c.FormValue("size")),
		ResponseFormat: strings.TrimSpace(c.FormValue("response_format")),
		Quality:        strings.TrimSpace(c.FormValue("quality")),
		Background:     strings.TrimSpace(c.FormValue("background")),
		Style:          strings.TrimSpace(c.FormValue("style")),
		N:              n,
		User:           strings.TrimSpace(c.FormValue("user")),
	}
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	return h.runImageOperation(c, imageOperationConfig{
		Alias:          model,
		IdempotencyKey: idempotencyKey,
		Operation:      imageOperationEdit,
		Builder: func(callCtx context.Context, route providers.Route) (models.ImageResponse, error) {
			req := baseReq
			req.Model = route.ResolveDeployment()
			req.Images = cloneImageInputs(baseReq.Images)
			if baseReq.Mask != nil {
				maskCopy := *baseReq.Mask
				req.Mask = &maskCopy
			}
			return route.Image.Edit(callCtx, req)
		},
	})
}

func (h *openAIHandler) imageVariations(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "multipart form required")
	}
	model := strings.TrimSpace(c.FormValue("model"))
	if model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	imageHeaders := gatherFormFiles(form, "image", "image[]")
	if len(imageHeaders) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "image file is required")
	}
	if len(imageHeaders) > 1 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "only one image file is supported for variations")
	}
	baseImage, err := loadImageInput(imageHeaders[0])
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "failed to read image upload")
	}
	n, err := parseImageCount(c.FormValue("n"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	baseReq := models.ImageVariationRequest{
		Model:          model,
		Image:          baseImage,
		Size:           strings.TrimSpace(c.FormValue("size")),
		ResponseFormat: strings.TrimSpace(c.FormValue("response_format")),
		Quality:        strings.TrimSpace(c.FormValue("quality")),
		Background:     strings.TrimSpace(c.FormValue("background")),
		Style:          strings.TrimSpace(c.FormValue("style")),
		N:              n,
		User:           strings.TrimSpace(c.FormValue("user")),
	}
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	return h.runImageOperation(c, imageOperationConfig{
		Alias:          model,
		IdempotencyKey: idempotencyKey,
		Operation:      imageOperationVariation,
		Builder: func(callCtx context.Context, route providers.Route) (models.ImageResponse, error) {
			req := baseReq
			req.Model = route.ResolveDeployment()
			req.Image = baseReq.Image
			return route.Image.Variation(callCtx, req)
		},
	})
}

func parseStop(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return []string{str}, nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("invalid stop value")
}

func parseImageCount(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 1, nil
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("n must be between 1 and 10")
	}
	if val < 1 || val > 10 {
		return 0, errors.New("n must be between 1 and 10")
	}
	return val, nil
}

func loadImageInput(fh *multipart.FileHeader) (models.ImageInput, error) {
	file, err := fh.Open()
	if err != nil {
		return models.ImageInput{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return models.ImageInput{}, err
	}
	if int64(len(data)) > maxImageUploadBytes {
		return models.ImageInput{}, fmt.Errorf("image uploads must be <= %d MB", maxImageUploadBytes/1024/1024)
	}
	contentType := strings.TrimSpace(fh.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" || !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return models.ImageInput{}, errors.New("image uploads must use an image/* content type")
	}
	return models.ImageInput{
		Data:        data,
		Filename:    fh.Filename,
		ContentType: contentType,
	}, nil
}

func cloneImageInputs(inputs []models.ImageInput) []models.ImageInput {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]models.ImageInput, len(inputs))
	copy(out, inputs)
	return out
}

func parseImageOverrideCost(metadata map[string]string, op imageOperationType) *int64 {
	return parseImageCost(metadata, op)
}

func parseImageCost(metadata map[string]string, op imageOperationType) *int64 {
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

func errMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func gatherFormFiles(form *multipart.Form, names ...string) []*multipart.FileHeader {
	if form == nil {
		return nil
	}
	var headers []*multipart.FileHeader
	for _, name := range names {
		if files := form.File[name]; len(files) > 0 {
			headers = append(headers, files...)
		}
	}
	return headers
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

func traceIDFromContext(c *fiber.Ctx) string {
	if v := c.Locals("requestid"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func validateResponseMetadata(md map[string]string) error {
	if len(md) == 0 {
		return nil
	}
	if len(md) > 16 {
		return fmt.Errorf("metadata supports at most 16 key/value pairs")
	}
	for key, value := range md {
		if len(key) > 64 {
			return fmt.Errorf("metadata key %q exceeds 64 characters", key)
		}
		if len(value) > 512 {
			return fmt.Errorf("metadata value for %q exceeds 512 characters", key)
		}
	}
	return nil
}

func buildResponseMessages(instructions string, input json.RawMessage) ([]models.ChatMessage, error) {
	messages := make([]models.ChatMessage, 0, 1)
	if instr := strings.TrimSpace(instructions); instr != "" {
		messages = append(messages, models.ChatMessage{Role: "system", Content: instr})
	}
	inputMessages, err := parseResponseInputItems(input)
	if err != nil {
		return nil, err
	}
	messages = append(messages, inputMessages...)
	return messages, nil
}

func parseResponseInputItems(raw json.RawMessage) ([]models.ChatMessage, error) {
	if len(raw) == 0 {
		return nil, errors.New("input is required")
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []models.ChatMessage{{Role: "user", Content: text}}, nil
	}
	var items []openAIChatMessage
	if err := json.Unmarshal(raw, &items); err == nil {
		if len(items) == 0 {
			return nil, errors.New("input array must contain at least one item")
		}
		return convertOpenAIMessages(items, "input item")
	}
	return nil, errors.New("input must be a string or array of message objects")
}

func convertResponsesResponse(resp models.ChatResponse, alias string, opts openAIResponseOptions) openAIResponsesPayload {
	return buildResponsesPayload(resp, alias, opts, "")
}

func buildResponsesPayload(resp models.ChatResponse, alias string, opts openAIResponseOptions, statusOverride string) openAIResponsesPayload {
	outputs := make([]openAIResponsesOutput, 0, len(resp.Choices))
	overallStatus := statusOverride
	if overallStatus == "" {
		overallStatus = "completed"
	}
	for _, choice := range resp.Choices {
		status := mapResponseStatus(choice.FinishReason)
		if status != "completed" && statusOverride == "" {
			overallStatus = status
		}
		outputs = append(outputs, openAIResponsesOutput{
			Type:   "message",
			ID:     fmt.Sprintf("%s-%d", resp.ID, choice.Index),
			Status: status,
			Role:   choice.Message.Role,
			Content: []openAIResponsesOutputContent{
				buildResponseOutputContent(choice.Message),
			},
		})
	}
	usage := openAIResponsesUsage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
		OutputTokensDetails: openAIResponsesUsageOutputDetails{
			ReasoningTokens: resp.Usage.ReasoningTokens,
		},
	}
	payload := openAIResponsesPayload{
		ID:                resp.ID,
		Object:            "response",
		CreatedAt:         resp.Created.Unix(),
		Status:            overallStatus,
		Model:             alias,
		Output:            outputs,
		ParallelToolCalls: opts.ParallelToolCalls,
		ToolChoice:        "auto",
		Tools:             []interface{}{},
		Usage:             usage,
	}
	if instr := strings.TrimSpace(opts.Instructions); instr != "" {
		payload.Instructions = instr
	}
	if len(opts.Metadata) > 0 {
		payload.Metadata = opts.Metadata
	}
	return payload
}

func mapResponseStatus(reason string) string {

	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "length", "content_filter":
		return "incomplete"
	case "":
		return "completed"
	default:
		return "completed"
	}
}

func buildResponseOutputContent(msg models.ChatMessage) openAIResponsesOutputContent {
	return openAIResponsesOutputContent{
		Type:        "output_text",
		Text:        msg.Text(),
		Annotations: []interface{}{},
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

type openAIStreamDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Reasoning string          `json:"reasoning,omitempty"`
}

type openAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

type openAIStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openAIStreamChoice `json:"choices"`
}

func convertStreamChunk(chunk models.ChatChunk, alias string) openAIStreamChunk {
	choices := make([]openAIStreamChoice, 0, len(chunk.Choices))
	for _, choice := range chunk.Choices {
		delta := openAIStreamDelta{
			Role:      choice.Delta.Role,
			Content:   models.MarshalMessageContent(choice.Delta),
			Reasoning: choice.Delta.Reasoning,
		}
		choices = append(choices, openAIStreamChoice{
			Index:        choice.Index,
			Delta:        delta,
			FinishReason: choice.FinishReason,
		})
	}

	return openAIStreamChunk{
		ID:      chunk.ID,
		Object:  "chat.completion.chunk",
		Created: chunk.Created.Unix(),
		Model:   alias,
		Choices: choices,
	}
}

func convertModerationHTTPResponse(resp models.ModerationResponse, alias string) openAIModerationResponse {
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
