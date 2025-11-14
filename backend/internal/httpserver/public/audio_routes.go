package public

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

func (h *openAIHandler) audioTranscriptions(c *fiber.Ctx) error {
	return h.handleAudioTranscription(c, models.AudioTranscriptionTaskTranscribe)
}

func (h *openAIHandler) audioTranslations(c *fiber.Ctx) error {
	return h.handleAudioTranscription(c, models.AudioTranscriptionTaskTranslate)
}

func (h *openAIHandler) handleAudioTranscription(c *fiber.Ctx, task models.AudioTranscriptionTask) error {
	form, err := c.MultipartForm()
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "multipart form required")
	}
	modelID := strings.TrimSpace(c.FormValue("model"))
	if modelID == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	fileHeaders := form.File["file"]
	if len(fileHeaders) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "file is required")
	}
	fh := fileHeaders[0]
	src, err := fh.Open()
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "failed to open file")
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "failed to read file")
	}

	prompt := c.FormValue("prompt")
	language := c.FormValue("language")
	var temperature *float32
	if val := strings.TrimSpace(c.FormValue("temperature")); val != "" {
		if parsed, err := strconv.ParseFloat(val, 32); err == nil {
			tmp := float32(parsed)
			temperature = &tmp
		}
	}

	return h.invokeAudioTranscription(c, audioInvocation{
		Model:    modelID,
		Task:     task,
		Payload:  data,
		Filename: fh.Filename,
		Mime:     fh.Header.Get("Content-Type"),
		Prompt:   prompt,
		Language: language,
		Temp:     temperature,
	})
}

type audioInvocation struct {
	Model    string
	Task     models.AudioTranscriptionTask
	Payload  []byte
	Filename string
	Mime     string
	Prompt   string
	Language string
	Temp     *float32
}

func (h *openAIHandler) invokeAudioTranscription(c *fiber.Ctx, inv audioInvocation) error {
	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	if !h.container.IsModelAllowed(rc.TenantID, inv.Model) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}
	routes := h.container.Engine.SelectRoutes(inv.Model)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}

	traceID := traceIDFromContext(c)
	alias := inv.Model

	budget, err := h.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if budget.Exceeded {
		setBudgetHeaders(c, budget)
		return httputil.WriteError(c, fiber.StatusForbidden, "tenant budget exceeded")
	}
	setBudgetHeaders(c, budget)

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := h.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return httputil.WriteError(c, fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer release()

	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration

	for _, route := range routes {
		var (
			resp models.AudioTranscriptionResponse
			err  error
		)
		lastRoute = route
		req := models.AudioTranscriptionRequest{
			Model: route.ResolveDeployment(),
			Task:  inv.Task,
			Input: models.AudioInput{
				Reader:      bytes.NewReader(inv.Payload),
				Filename:    inv.Filename,
				ContentType: inv.Mime,
				Bytes:       int64(len(inv.Payload)),
			},
			Prompt:      inv.Prompt,
			Temperature: inv.Temp,
			Language:    inv.Language,
		}
		start := time.Now()
		if inv.Task == models.AudioTranscriptionTaskTranslate && route.AudioTranslate != nil {
			resp, err = route.AudioTranslate.Translate(ctx, req)
		} else if route.AudioTranscribe != nil {
			resp, err = route.AudioTranscribe.Transcribe(ctx, req)
		} else {
			continue
		}
		if err != nil {
			h.container.Engine.ReportFailure(alias, route)
			lastLatency = time.Since(start)
			lastErr = err
			continue
		}
		h.container.Engine.ReportSuccess(alias, route)
		if tokens := int(resp.Usage.TotalTokens); tokens > 0 {
			if err := h.container.RateLimiter.TokenAllowance(ctx, keyKey, tokens, keyCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, "token accounting failed")
			}
			if err := h.container.RateLimiter.TokenAllowance(ctx, tenantKey, tokens, tenantCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, "token accounting failed")
			}
		}
		elapsed := time.Since(start)
		record := usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  route.Provider,
			Usage:     resp.Usage,
			Latency:   elapsed,
			Status:    fiber.StatusOK,
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   true,
		}
		if status, err := h.container.UsageLogger.Record(ctx, record); err == nil {
			setBudgetHeaders(c, status)
		}
		return c.JSON(fiber.Map{"text": resp.Text})
	}

	if lastErr == nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support audio tasks")
	}
	if lastRoute.Provider != "" {
		_, _ = h.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  lastRoute.Provider,
			Latency:   lastLatency,
			Status:    fiber.StatusBadGateway,
			ErrorCode: lastErr.Error(),
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
	}
	return httputil.WriteError(c, fiber.StatusBadGateway, lastErr.Error())
}

func (h *openAIHandler) audioSpeech(c *fiber.Ctx) error {
	var payload audioSpeechRequest
	if err := c.BodyParser(&payload); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	model := strings.TrimSpace(payload.Model)
	if model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	input := strings.TrimSpace(payload.Input)
	if input == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "input is required")
	}
	if payload.Stream {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "stream=true is not supported for speech yet")
	}
	format := strings.TrimSpace(payload.Format)
	if format == "" {
		format = strings.TrimSpace(payload.ResponseFormat)
	}
	if format == "" {
		format = "mp3"
	}
	req := models.AudioSpeechRequest{
		Model:        model,
		Input:        input,
		Voice:        strings.TrimSpace(payload.Voice),
		Format:       format,
		Stream:       payload.Stream,
		StreamFormat: strings.TrimSpace(payload.StreamFormat),
	}
	return h.invokeAudioSpeech(c, req)
}

type audioSpeechRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice"`
	Format         string `json:"format"`
	ResponseFormat string `json:"response_format"`
	Stream         bool   `json:"stream"`
	StreamFormat   string `json:"stream_format"`
}

func (h *openAIHandler) invokeAudioSpeech(c *fiber.Ctx, req models.AudioSpeechRequest) error {
	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	alias := req.Model
	if !h.container.IsModelAllowed(rc.TenantID, alias) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}
	routes := h.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}
	traceID := traceIDFromContext(c)
	budget, err := h.container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
	}
	if budget.Exceeded {
		setBudgetHeaders(c, budget)
		return httputil.WriteError(c, fiber.StatusForbidden, "tenant budget exceeded")
	}
	setBudgetHeaders(c, budget)

	keyKey, keyCfg, tenantKey, tenantCfg, release, err := h.container.AcquireRateLimits(ctx, alias)
	if err != nil {
		if errors.Is(err, limits.ErrLimitExceeded) {
			return httputil.WriteError(c, fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer release()

	var lastErr error
	var lastRoute providers.Route
	var lastLatency time.Duration

	for _, route := range routes {
		providerReq := req
		providerReq.Model = route.ResolveDeployment()
		providerReq.Voice = resolveSpeechVoice(route.Metadata, providerReq.Voice)
		providerReq.Format = resolveSpeechFormat(route.Metadata, providerReq.Format)
		var resp models.AudioSpeechResponse
		var synthErr error
		start := time.Now()
		lastRoute = route
		if providerReq.Stream {
			if route.TextToSpeechStream == nil {
				continue
			}
			lastLatency = time.Since(start)
			return httputil.WriteError(c, fiber.StatusNotImplemented, "streaming speech is not supported")
		}
		if route.TextToSpeech == nil {
			continue
		}
		resp, synthErr = route.TextToSpeech.Synthesize(ctx, providerReq)
		if synthErr != nil {
			lastErr = synthErr
			lastLatency = time.Since(start)
			h.container.Engine.ReportFailure(alias, route)
			continue
		}
		h.container.Engine.ReportSuccess(alias, route)
		if tokens := int(resp.Usage.TotalTokens); tokens > 0 {
			if err := h.container.RateLimiter.TokenAllowance(ctx, keyKey, tokens, keyCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, "token accounting failed")
			}
			if err := h.container.RateLimiter.TokenAllowance(ctx, tenantKey, tokens, tenantCfg); err != nil {
				if errors.Is(err, limits.ErrLimitExceeded) {
					return httputil.WriteError(c, fiber.StatusTooManyRequests, "token limit exceeded")
				}
				return httputil.WriteError(c, fiber.StatusInternalServerError, "token accounting failed")
			}
		}
		elapsed := time.Since(start)
		record := usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  route.Provider,
			Usage:     resp.Usage,
			Latency:   elapsed,
			Status:    fiber.StatusOK,
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   true,
		}
		if status, err := h.container.UsageLogger.Record(ctx, record); err == nil {
			setBudgetHeaders(c, status)
		}
		if err := writeAudioSpeechResponse(c, providerReq, resp); err != nil {
			return err
		}
		return nil
	}

	if lastErr == nil {
		if req.Stream {
			return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support streaming speech")
		}
		return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support speech synthesis")
	}
	if lastRoute.Provider != "" {
		_, _ = h.container.UsageLogger.Record(ctx, usagepipeline.Record{
			Context:   rc,
			Alias:     alias,
			Provider:  lastRoute.Provider,
			Latency:   lastLatency,
			Status:    fiber.StatusBadGateway,
			ErrorCode: lastErr.Error(),
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
			Success:   false,
		})
	}
	return httputil.WriteError(c, fiber.StatusBadGateway, lastErr.Error())
}

func writeAudioSpeechResponse(c *fiber.Ctx, req models.AudioSpeechRequest, resp models.AudioSpeechResponse) error {
	contentType := audioContentType(req.Format)
	if contentType == "application/json" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "unsupported audio format")
	}
	c.Set(fiber.HeaderContentType, contentType)
	c.Set(fiber.HeaderContentLength, strconv.Itoa(len(resp.Audio)))
	return c.Send(resp.Audio)
}

func resolveSpeechVoice(metadata map[string]string, requested string) string {
	if v := strings.TrimSpace(requested); v != "" {
		return v
	}
	if metadata != nil {
		if v := strings.TrimSpace(metadata["audio_voice"]); v != "" {
			return v
		}
		if v := strings.TrimSpace(metadata["audio_default_voice"]); v != "" {
			return v
		}
	}
	return "alloy"
}

func resolveSpeechFormat(metadata map[string]string, requested string) string {
	format := strings.ToLower(strings.TrimSpace(requested))
	if format == "" && metadata != nil {
		format = strings.ToLower(strings.TrimSpace(metadata["audio_format"]))
	}
	if format == "" {
		format = "mp3"
	}
	return format
}

func audioContentType(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "mp3":
		return "audio/mpeg"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	case "opus":
		return "audio/opus"
	case "wav":
		return "audio/wav"
	case "pcm":
		return "audio/L16"
	default:
		return "audio/mpeg"
	}
}
