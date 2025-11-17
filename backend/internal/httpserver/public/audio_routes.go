package public

import (
	"io"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
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
	field := "file"
	if len(fileHeaders) == 0 {
		fileHeaders = form.File["audio"]
		field = "audio"
	}
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
	maxUpload := int64(h.container.Config.Audio.MaxUploadMB) * 1024 * 1024
	if maxUpload > 0 && int64(len(data)) > maxUpload {
		return httputil.WriteError(c, fiber.StatusRequestEntityTooLarge, "audio file exceeds maximum allowed size")
	}

	prompt := c.FormValue("prompt")
	language := c.FormValue("language")
	user := strings.TrimSpace(c.FormValue("user"))
	rawFormat := c.FormValue("response_format")
	format, ok := models.ParseAudioResponseFormat(rawFormat)
	if !ok {
		return httputil.WriteError(c, fiber.StatusBadRequest, "unsupported response_format")
	}
	if task == models.AudioTranscriptionTaskTranslate && format == models.AudioResponseFormatDiarized {
		return httputil.WriteError(c, fiber.StatusBadRequest, "diarized_json is not supported for translations")
	}
	stream := false
	if val := strings.TrimSpace(c.FormValue("stream")); val != "" {
		stream = strings.EqualFold(val, "true") || val == "1"
	}
	if stream {
		if task != models.AudioTranscriptionTaskTranscribe {
			return httputil.WriteError(c, fiber.StatusBadRequest, "streaming is only supported for transcriptions")
		}
		if format != models.AudioResponseFormatDiarized {
			return httputil.WriteError(c, fiber.StatusBadRequest, "streaming requires response_format=diarized_json")
		}
	}
	rawGran := form.Value["timestamp_granularities[]"]
	if len(rawGran) == 0 {
		rawGran = form.Value["timestamp_granularities"]
	}
	granularities := make([]models.AudioTimestampGranularity, 0, len(rawGran))
	if len(rawGran) > 0 {
		seen := make(map[models.AudioTimestampGranularity]struct{}, len(rawGran))
		for _, val := range rawGran {
			gran, ok := models.ParseAudioGranularity(val)
			if !ok {
				return httputil.WriteError(c, fiber.StatusBadRequest, "invalid timestamp_granularities value")
			}
			if _, exists := seen[gran]; exists {
				continue
			}
			seen[gran] = struct{}{}
			granularities = append(granularities, gran)
		}
		if format != models.AudioResponseFormatVerboseJSON {
			return httputil.WriteError(c, fiber.StatusBadRequest, "timestamp_granularities require response_format=verbose_json")
		}
	}
	var temperature *float32
	if val := strings.TrimSpace(c.FormValue("temperature")); val != "" {
		if parsed, err := strconv.ParseFloat(val, 32); err == nil {
			tmp := float32(parsed)
			temperature = &tmp
		}
	}

	invocation := audioInvocation{
		Model:    modelID,
		Task:     task,
		Payload:  data,
		Filename: fh.Filename,
		Mime:     fh.Header.Get("Content-Type"),
		Prompt:   prompt,
		Language: language,
		User:     user,
		Format:   format,
		Granular: granularities,
		Field:    field,
		Stream:   stream,
		Temp:     temperature,
	}
	if stream {
		return h.invokeAudioTranscriptionStream(c, invocation)
	}
	return h.invokeAudioTranscription(c, invocation)
}

type audioInvocation struct {
	Model    string
	Task     models.AudioTranscriptionTask
	Payload  []byte
	Filename string
	Mime     string
	Prompt   string
	Language string
	User     string
	Format   models.AudioResponseFormat
	Granular []models.AudioTimestampGranularity
	Field    string
	Temp     *float32
	Stream   bool
}

func (h *openAIHandler) invokeAudioTranscription(c *fiber.Ctx, inv audioInvocation) error {
	return h.audioPipeline.Transcribe(c, inv)
}

func (h *openAIHandler) invokeAudioTranscriptionStream(c *fiber.Ctx, inv audioInvocation) error {
	return h.audioPipeline.TranscribeStream(c, inv)
}

func writeAudioTranscriptionResponse(c *fiber.Ctx, resp models.AudioTranscriptionResponse) error {
	payload := resp.Payload
	if len(payload) == 0 {
		if resp.Format.IsJSONFormat() || resp.Format == "" {
			return c.JSON(fiber.Map{"text": resp.Text})
		}
		if resp.Text == "" {
			return c.JSON(fiber.Map{"text": ""})
		}
		payload = []byte(resp.Text)
	}
	contentType := strings.TrimSpace(resp.ContentType)
	if contentType == "" && resp.Format != "" {
		contentType = resp.Format.ContentType()
	}
	if contentType != "" {
		c.Set(fiber.HeaderContentType, contentType)
	}
	c.Set(fiber.HeaderContentLength, strconv.Itoa(len(payload)))
	return c.Send(payload)
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
	return h.audioPipeline.Speech(c, req)
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

func routeSupportsAudioFormat(metadata map[string]string, format models.AudioResponseFormat) bool {
	if format == "" || format == models.AudioResponseFormatJSON {
		return true
	}
	values := parseCSVMetadata(metadata["audio_formats"])
	if len(values) == 0 {
		return true
	}
	for _, val := range values {
		if strings.EqualFold(val, string(format)) {
			return true
		}
	}
	return false
}

func routeSupportsGranularities(metadata map[string]string, requested []models.AudioTimestampGranularity) bool {
	if len(requested) == 0 {
		return true
	}
	values := parseCSVMetadata(metadata["audio_timestamp_granularities"])
	if len(values) == 0 {
		return true
	}
	for _, gran := range requested {
		match := false
		for _, val := range values {
			if strings.EqualFold(val, string(gran)) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

func routeSupportsAudioStream(metadata map[string]string) bool {
	if metadata == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(metadata["audio_streaming"])) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

func parseCSVMetadata(val string) []string {
	if strings.TrimSpace(val) == "" {
		return nil
	}
	fields := strings.Split(val, ",")
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
