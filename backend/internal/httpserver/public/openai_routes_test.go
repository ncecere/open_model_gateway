package public

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

func TestParseImageCostPrefersOperationSpecificKey(t *testing.T) {
	meta := map[string]string{
		"price_image_cents":           "100",
		"price_image_edit_cents":      "275",
		"price_image_variation_cents": "150",
	}

	if cost := parseImageCost(meta, imageOperationEdit); cost == nil || *cost != 275 {
		t.Fatalf("expected edit override to win, got %v", cost)
	}
	if cost := parseImageCost(meta, imageOperationVariation); cost == nil || *cost != 150 {
		t.Fatalf("expected variation override to win, got %v", cost)
	}
	if cost := parseImageCost(meta, imageOperationGeneration); cost == nil || *cost != 100 {
		t.Fatalf("expected base price for generations, got %v", cost)
	}
	if cost := parseImageCost(nil, imageOperationGeneration); cost != nil {
		t.Fatalf("expected nil when metadata missing, got %v", *cost)
	}
}

func TestLoadImageInputAcceptsImageContentType(t *testing.T) {
	fh := buildTestFileHeader(t, "image", "foo.png", "image/png", pngBytes())
	input, err := loadImageInput(fh)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if got, want := input.ContentType, "image/png"; got != want {
		t.Fatalf("expected content type %s got %s", want, got)
	}
}

func TestLoadImageInputRejectsOversized(t *testing.T) {
	payload := bytes.Repeat([]byte("a"), maxImageUploadBytes+1)
	fh := buildTestFileHeader(t, "image", "big.png", "image/png", payload)
	if _, err := loadImageInput(fh); err == nil || !strings.Contains(err.Error(), "must be <=") {
		t.Fatalf("expected size validation error, got %v", err)
	}
}

func TestLoadImageInputRequiresImageContentType(t *testing.T) {
	fh := buildTestFileHeader(t, "image", "text.txt", "text/plain", []byte("hello world"))
	if _, err := loadImageInput(fh); err == nil || !strings.Contains(err.Error(), "image uploads must use an image/") {
		t.Fatalf("expected content-type error, got %v", err)
	}
}

func TestWriteAudioTranscriptionResponseJSONFallback(t *testing.T) {
	resp := models.AudioTranscriptionResponse{
		Format: models.AudioResponseFormatJSON,
		Text:   "sample text",
	}
	httpResp := performAudioTranscriptionResponse(t, resp)
	defer httpResp.Body.Close()
	if ct := httpResp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected json content-type, got %s", ct)
	}
	body, _ := io.ReadAll(httpResp.Body)
	if got := strings.TrimSpace(string(body)); got != `{"text":"sample text"}` {
		t.Fatalf("unexpected body %s", got)
	}
}

func TestWriteAudioTranscriptionResponseTextPayload(t *testing.T) {
	resp := models.AudioTranscriptionResponse{
		Format:      models.AudioResponseFormatText,
		ContentType: "text/plain; charset=utf-8",
		Payload:     []byte("hi there"),
	}
	httpResp := performAudioTranscriptionResponse(t, resp)
	defer httpResp.Body.Close()
	if ct := httpResp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("expected text content-type, got %s", ct)
	}
	body, _ := io.ReadAll(httpResp.Body)
	if string(body) != "hi there" {
		t.Fatalf("unexpected body %s", string(body))
	}
}

func TestRouteSupportsAudioFormat(t *testing.T) {
	meta := map[string]string{"audio_formats": "json, text, vtt"}
	if !routeSupportsAudioFormat(meta, models.AudioResponseFormatJSON) {
		t.Fatalf("expected json format to be supported")
	}
	if routeSupportsAudioFormat(meta, models.AudioResponseFormatVerboseJSON) {
		t.Fatalf("expected verbose_json to be rejected")
	}
	if !routeSupportsAudioFormat(nil, models.AudioResponseFormatVerboseJSON) {
		t.Fatalf("expected missing metadata to allow format by default")
	}
}

func TestRouteSupportsGranularities(t *testing.T) {
	meta := map[string]string{"audio_timestamp_granularities": "word"}
	if !routeSupportsGranularities(meta, []models.AudioTimestampGranularity{models.AudioTimestampGranularityWord}) {
		t.Fatalf("expected word granularity to be supported")
	}
	if routeSupportsGranularities(meta, []models.AudioTimestampGranularity{models.AudioTimestampGranularitySegment}) {
		t.Fatalf("expected segment granularity to be rejected")
	}
	if !routeSupportsGranularities(nil, []models.AudioTimestampGranularity{models.AudioTimestampGranularitySegment}) {
		t.Fatalf("expected default granularity acceptance when metadata missing")
	}
}

func TestRouteSupportsAudioStream(t *testing.T) {
	if routeSupportsAudioStream(map[string]string{"audio_streaming": "true"}) == false {
		t.Fatalf("expected true metadata to allow streaming")
	}
	if routeSupportsAudioStream(map[string]string{"audio_streaming": "false"}) {
		t.Fatalf("expected false metadata to disable streaming")
	}
	if routeSupportsAudioStream(nil) {
		t.Fatalf("expected nil metadata to disable streaming")
	}
}

func buildTestFileHeader(t *testing.T, field, filename, contentType string, data []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write form data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(int64(len(data)) + 1024); err != nil {
		t.Fatalf("parse multipart: %v", err)
	}

	fh := req.MultipartForm.File[field][0]
	if contentType != "" {
		fh.Header.Set("Content-Type", contentType)
	}
	return fh
}

func pngBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x04, 0x00, 0x00, 0x00, 0xB5, 0x1C, 0x0C,
		0x02, 0x00, 0x00, 0x00, 0x0B, 0x49, 0x44, 0x41,
		0x54, 0x78, 0xDA, 0x63, 0xFC, 0xFF, 0x0F, 0x00,
		0x02, 0x83, 0x01, 0x80, 0xE6, 0xE1, 0x40, 0x99,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44,
		0xAE, 0x42, 0x60, 0x82,
	}
}

func performAudioTranscriptionResponse(t *testing.T, resp models.AudioTranscriptionResponse) *http.Response {
	t.Helper()
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return writeAudioTranscriptionResponse(c, resp)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	httpResp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	return httpResp
}
