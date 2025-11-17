package public

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
	"github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

func TestHandleAudioTranscriptionRejectsMissingMultipart(t *testing.T) {
	handler := newAudioHandlerForTests()
	app := fiber.New()
	app.Post("/audio/transcriptions", func(c *fiber.Ctx) error {
		return handler.handleAudioTranscription(c, models.AudioTranscriptionTaskTranscribe)
	})

	req := httptest.NewRequest(http.MethodPost, "/audio/transcriptions", strings.NewReader("not multipart"))
	req.Header.Set(fiber.HeaderContentType, "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestHandleAudioTranscriptionRequiresModel(t *testing.T) {
	handler := newAudioHandlerForTests()
	resp := executeAudioTranscriptionRequest(t, handler, models.AudioTranscriptionTaskTranscribe, map[string]string{}, []byte("hello"))
	assertStatusContains(t, resp, http.StatusBadRequest, "model is required")
}

func TestHandleAudioTranscriptionRequiresFile(t *testing.T) {
	handler := newAudioHandlerForTests()
	app := fiber.New()
	app.Post("/audio/transcriptions", func(c *fiber.Ctx) error {
		return handler.handleAudioTranscription(c, models.AudioTranscriptionTaskTranscribe)
	})
	fields := map[string]string{"model": "gpt"}
	req := buildAudioMultipart(t, fields, "image", []byte("hello"))
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	assertStatusContains(t, resp, http.StatusBadRequest, "file is required")
}

func TestHandleAudioTranscriptionValidatesResponseFormat(t *testing.T) {
	handler := newAudioHandlerForTests()
	resp := executeAudioTranscriptionRequest(t, handler, models.AudioTranscriptionTaskTranscribe, map[string]string{
		"model":           "gpt",
		"response_format": "unknown",
	}, []byte("hello"))
	assertStatusContains(t, resp, http.StatusBadRequest, "unsupported response_format")
}

func TestHandleAudioTranslationRejectsDiarized(t *testing.T) {
	handler := newAudioHandlerForTests()
	resp := executeAudioTranscriptionRequest(t, handler, models.AudioTranscriptionTaskTranslate, map[string]string{
		"model":           "gpt",
		"response_format": "diarized_json",
	}, []byte("hello"))
	assertStatusContains(t, resp, http.StatusBadRequest, "diarized_json is not supported for translations")
}

func TestHandleAudioTranscriptionStreamingGuard(t *testing.T) {
	handler := newAudioHandlerForTests()
	resp := executeAudioTranscriptionRequest(t, handler, models.AudioTranscriptionTaskTranscribe, map[string]string{
		"model":           "gpt",
		"response_format": "json",
		"stream":          "true",
	}, []byte("hello"))
	assertStatusContains(t, resp, http.StatusBadRequest, "streaming requires response_format=diarized_json")
}

func TestHandleAudioTranscriptionTranslationStreamingRejected(t *testing.T) {
	handler := newAudioHandlerForTests()
	resp := executeAudioTranscriptionRequest(t, handler, models.AudioTranscriptionTaskTranslate, map[string]string{
		"model":           "gpt",
		"response_format": "diarized_json",
		"stream":          "true",
	}, []byte("hello"))
	assertStatusContains(t, resp, http.StatusBadRequest, "diarized_json is not supported for translations")
}

func TestHandleAudioTranscriptionGranularityRequiresVerbose(t *testing.T) {
	handler := newAudioHandlerForTests()
	resp := executeAudioTranscriptionRequest(t, handler, models.AudioTranscriptionTaskTranscribe, map[string]string{
		"model":                     "gpt",
		"response_format":           "json",
		"timestamp_granularities[]": "word",
	}, []byte("hello"))
	assertStatusContains(t, resp, http.StatusBadRequest, "timestamp_granularities require response_format=verbose_json")
}

func TestHandleAudioTranscriptionInvalidGranularityValue(t *testing.T) {
	handler := newAudioHandlerForTests()
	resp := executeAudioTranscriptionRequest(t, handler, models.AudioTranscriptionTaskTranscribe, map[string]string{
		"model":                     "gpt",
		"response_format":           "verbose_json",
		"timestamp_granularities[]": "bogus",
	}, []byte("hello"))
	assertStatusContains(t, resp, http.StatusBadRequest, "invalid timestamp_granularities value")
}

func TestHandleAudioSpeechRequiresModel(t *testing.T) {
	handler := newAudioHandlerForTests()
	app := fiber.New()
	app.Post("/audio/speech", handler.audioSpeech)
	req := httptest.NewRequest(http.MethodPost, "/audio/speech", strings.NewReader(`{"input":"hello"}`))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	assertStatusContains(t, resp, http.StatusBadRequest, "model is required")
}

func TestHandleAudioSpeechRequiresInput(t *testing.T) {
	handler := newAudioHandlerForTests()
	app := fiber.New()
	app.Post("/audio/speech", handler.audioSpeech)
	req := httptest.NewRequest(http.MethodPost, "/audio/speech", strings.NewReader(`{"model":"gpt"}`))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	assertStatusContains(t, resp, http.StatusBadRequest, "input is required")
}

func TestHandleAudioSpeechRejectsStreaming(t *testing.T) {
	handler := newAudioHandlerForTests()
	app := fiber.New()
	app.Post("/audio/speech", handler.audioSpeech)
	req := httptest.NewRequest(http.MethodPost, "/audio/speech", strings.NewReader(`{"model":"gpt","input":"hello","stream":true}`))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	assertStatusContains(t, resp, http.StatusNotImplemented, "stream=true is not supported for speech yet")
}

func TestInvokeAudioTranscriptionStreamHappyPath(t *testing.T) {
	streamer := &stubAudioStream{}
	container := basicStreamingContainer(streamer)
	app := fiber.New()
	app.Post("/audio", func(c *fiber.Ctx) error {
		ctx := requestctx.WithContext(context.Background(), &requestctx.Context{})
		c.SetUserContext(ctx)
		handler := &openAIHandler{
			container:     container,
			audioPipeline: newAudioPipeline(container),
		}
		return handler.invokeAudioTranscriptionStream(c, audioInvocation{
			Model:   "gpt",
			Task:    models.AudioTranscriptionTaskTranscribe,
			Payload: []byte("payload"),
			Format:  models.AudioResponseFormatDiarized,
		})
	})
	req := httptest.NewRequest(http.MethodPost, "/audio", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 got %d body=%s", resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "data: {\"chunk\":\"hello\"}") {
		t.Fatalf("expected stream chunk, got %s", string(body))
	}
	if !strings.Contains(string(body), "data: [DONE]") {
		t.Fatalf("expected [DONE] terminator")
	}
}

func buildAudioMultipart(t *testing.T, fields map[string]string, fileField string, payload []byte) *http.Request {
	return buildAudioMultipartPath(t, fields, fileField, payload, "/audio/transcriptions")
}

func buildAudioMultipartPath(t *testing.T, fields map[string]string, fileField string, payload []byte, path string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fileField, "test.wav")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write file: %v", err)
	}
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write field %s: %v", k, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set(fiber.HeaderContentType, writer.FormDataContentType())
	return req
}

func executeAudioTranscriptionRequest(t *testing.T, handler *openAIHandler, task models.AudioTranscriptionTask, fields map[string]string, payload []byte) *http.Response {
	t.Helper()
	app := fiber.New()
	path := "/audio/transcriptions"
	if task == models.AudioTranscriptionTaskTranslate {
		path = "/audio/translations"
	}
	app.Post(path, func(c *fiber.Ctx) error {
		return handler.handleAudioTranscription(c, task)
	})
	req := buildAudioMultipartPath(t, fields, "file", payload, path)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	return resp
}

func assertStatusContains(t *testing.T, resp *http.Response, status int, msg string) {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != status {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d got %d body=%s", status, resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), msg) {
		t.Fatalf("expected body to contain %q but got %s", msg, string(body))
	}
}

func newAudioHandlerForTests() *openAIHandler {
	container := basicTestContainer()
	return &openAIHandler{
		container:     container,
		audioPipeline: newAudioPipeline(container),
	}
}
func basicTestContainer() *app.Container {
	return &app.Container{
		Config: &config.Config{
			Audio: config.AudioConfig{MaxUploadMB: 10},
		},
		Engine: router.NewEngine(),
	}
}

type stubAudioStream struct{}

func (stubAudioStream) TranscribeStream(ctx context.Context, req models.AudioTranscriptionRequest) (<-chan models.AudioTranscriptionStreamChunk, func() error, error) {
	ch := make(chan models.AudioTranscriptionStreamChunk, 2)
	ch <- models.AudioTranscriptionStreamChunk{Payload: []byte(`{"chunk":"hello"}`)}
	ch <- models.AudioTranscriptionStreamChunk{Done: true, Usage: &models.Usage{TotalTokens: 10}}
	close(ch)
	return ch, func() error { return nil }, nil
}

func basicStreamingContainer(streamer providers.AudioTranscriptionStreaming) *app.Container {
	engine := router.NewEngine()
	engine.SetRoutes(map[string][]providers.Route{
		"gpt": {
			{
				Alias:                 "gpt",
				Provider:              "openai",
				Model:                 "whisper-1",
				Metadata:              map[string]string{"audio_streaming": "true", "audio_formats": "diarized_json"},
				AudioTranscribeStream: streamer,
			},
		},
	})
	return &app.Container{
		Config: &config.Config{
			Audio: config.AudioConfig{MaxUploadMB: 10},
		},
		Engine:      engine,
		UsageLogger: newFakeUsageLogger(),
		RateLimiter: limits.NewRateLimiter(nil, nil),
	}
}

type fakeUsageLogger struct {
	status usagepipeline.BudgetStatus
}

func newFakeUsageLogger() *fakeUsageLogger {
	return &fakeUsageLogger{
		status: usagepipeline.BudgetStatus{
			LimitCents: 100_00,
		},
	}
}

func (f *fakeUsageLogger) LoadCatalog(entries []config.ModelCatalogEntry) {}

func (f *fakeUsageLogger) SetConfig(cfg config.BudgetConfig) {}

func (f *fakeUsageLogger) CheckBudget(ctx context.Context, rc *requestctx.Context, now time.Time) (usagepipeline.BudgetStatus, error) {
	return f.status, nil
}

func (f *fakeUsageLogger) Record(ctx context.Context, rec usagepipeline.Record) (usagepipeline.BudgetStatus, error) {
	return f.status, errors.New("skip budget headers for tests")
}

func (f *fakeUsageLogger) Shutdown(ctx context.Context) error {
	return nil
}
