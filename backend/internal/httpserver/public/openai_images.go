package public

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

const (
	maxImageUploadBytes = 4 * 1024 * 1024
	maxImageInputs      = 16
)

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

func (h *openAIHandler) imageGenerations(c *fiber.Ctx) error {
	var req openAIImageRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Model = strings.TrimSpace(req.Model)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.Size = strings.TrimSpace(req.Size)
	req.Quality = strings.TrimSpace(req.Quality)
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
	imagePixels := models.ImagePixelCount(req.Size, n)

	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	baseReq := req
	return h.runImageOperation(c, imageOperationConfig{
		Alias:           req.Model,
		IdempotencyKey:  idempotencyKey,
		Operation:       imageOperationGeneration,
		ImagePixels:     imagePixels,
		PricingMetadata: buildImagePricingMetadata(imageOperationGeneration, req.Quality, req.Size),
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
	requestSize := strings.TrimSpace(c.FormValue("size"))
	pricingSize := requestSize
	if pricingSize == "" && len(images) > 0 {
		if derived, ok := models.ImageSizeFromInput(images[0]); ok {
			pricingSize = derived
		}
	}
	imagePixels := models.ImagePixelCount(pricingSize, n)
	quality := strings.TrimSpace(c.FormValue("quality"))
	baseReq := models.ImageEditRequest{
		Model:          model,
		Prompt:         prompt,
		Images:         images,
		Mask:           maskInput,
		Size:           requestSize,
		ResponseFormat: strings.TrimSpace(c.FormValue("response_format")),
		Quality:        quality,
		Background:     strings.TrimSpace(c.FormValue("background")),
		Style:          strings.TrimSpace(c.FormValue("style")),
		N:              n,
		User:           strings.TrimSpace(c.FormValue("user")),
	}
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	return h.runImageOperation(c, imageOperationConfig{
		Alias:           model,
		IdempotencyKey:  idempotencyKey,
		Operation:       imageOperationEdit,
		ImagePixels:     imagePixels,
		PricingMetadata: buildImagePricingMetadata(imageOperationEdit, quality, pricingSize),
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
	requestSize := strings.TrimSpace(c.FormValue("size"))
	pricingSize := requestSize
	if pricingSize == "" {
		if derived, ok := models.ImageSizeFromInput(baseImage); ok {
			pricingSize = derived
		}
	}
	imagePixels := models.ImagePixelCount(pricingSize, n)
	quality := strings.TrimSpace(c.FormValue("quality"))
	baseReq := models.ImageVariationRequest{
		Model:          model,
		Image:          baseImage,
		Size:           requestSize,
		ResponseFormat: strings.TrimSpace(c.FormValue("response_format")),
		Quality:        quality,
		Background:     strings.TrimSpace(c.FormValue("background")),
		Style:          strings.TrimSpace(c.FormValue("style")),
		N:              n,
		User:           strings.TrimSpace(c.FormValue("user")),
	}
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	return h.runImageOperation(c, imageOperationConfig{
		Alias:           model,
		IdempotencyKey:  idempotencyKey,
		Operation:       imageOperationVariation,
		ImagePixels:     imagePixels,
		PricingMetadata: buildImagePricingMetadata(imageOperationVariation, quality, pricingSize),
		Builder: func(callCtx context.Context, route providers.Route) (models.ImageResponse, error) {
			req := baseReq
			req.Model = route.ResolveDeployment()
			req.Image = baseReq.Image
			return route.Image.Variation(callCtx, req)
		},
	})
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

func buildImagePricingMetadata(op imageOperationType, quality string, resolution string) map[string]string {
	metadata := map[string]string{
		"operation": imageOperationLabel(op),
	}
	q := strings.TrimSpace(quality)
	if q == "" {
		q = "standard"
	}
	if q != "" {
		metadata["quality"] = q
	}
	if res := strings.TrimSpace(resolution); res != "" {
		metadata["resolution"] = res
	}
	return metadata
}

func imageOperationLabel(op imageOperationType) string {
	switch op {
	case imageOperationEdit:
		return "image_edit"
	case imageOperationVariation:
		return "image_variation"
	default:
		return "image_generation"
	}
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
