package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/pagination"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/packages/respjson"

	openaihelper "github.com/ncecere/open_model_gateway/backend/internal/adapters/openaihelper"
	"github.com/ncecere/open_model_gateway/backend/internal/apperror"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers/streamutil"
)

// Options configure the native OpenAI adapter.
type Options struct {
	APIKey       string
	BaseURL      string
	Organization string
	Extra        []option.RequestOption
	AllowNoKey   bool
}

// Adapter wraps the official OpenAI SDK for native + compatible deployments.
type Adapter struct {
	client     *openai.Client
	httpClient *http.Client
}

// New creates an OpenAI adapter using the provided API key and optional base URL/organization.
func New(opts Options) (*Adapter, error) {
	apiKey := strings.TrimSpace(opts.APIKey)
	if apiKey == "" && !opts.AllowNoKey {
		return nil, apperror.Validation("openai.New", "api key required")
	}

	requestOpts := make([]option.RequestOption, 0, 2)
	if apiKey != "" {
		requestOpts = append(requestOpts, option.WithAPIKey(apiKey))
	}
	if strings.TrimSpace(opts.BaseURL) != "" {
		requestOpts = append(requestOpts, option.WithBaseURL(strings.TrimRight(opts.BaseURL, "/")))
	}
	if strings.TrimSpace(opts.Organization) != "" {
		requestOpts = append(requestOpts, option.WithOrganization(strings.TrimSpace(opts.Organization)))
	}
	requestOpts = append(requestOpts, opts.Extra...)

	client := openai.NewClient(requestOpts...)
	httpClient := &http.Client{Timeout: 10 * time.Second}

	return &Adapter{client: &client, httpClient: httpClient}, nil
}

// Chat performs a non-streaming chat completion request.
func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	params, err := openaihelper.BuildChatParams(req)
	if err != nil {
		return models.ChatResponse{}, err
	}
	resp, err := a.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return models.ChatResponse{}, err
	}
	return convertChatResponse(*resp), nil
}

// ChatStream performs a streaming chat completion request.
func (a *Adapter) ChatStream(ctx context.Context, req models.ChatRequest) (<-chan models.ChatChunk, func() error, error) {
	params, err := openaihelper.BuildChatParams(req)
	if err != nil {
		return nil, nil, err
	}
	params.StreamOptions.IncludeUsage = param.NewOpt(true)
	stream := a.client.Chat.Completions.NewStreaming(ctx, params)
	if err := stream.Err(); err != nil {
		stream.Close()
		return nil, nil, err
	}

	forward := func(ctx context.Context, yield streamutil.YieldFunc) {
		for stream.Next() {
			chunk := stream.Current()
			if !yield(convertChatChunk(chunk)) {
				return
			}
		}
	}

	chunks, cancel := streamutil.Forward(ctx, stream.Close, forward)
	return chunks, cancel, nil
}

// Embed creates embeddings using the selected OpenAI model.
func (a *Adapter) Embed(ctx context.Context, req models.EmbeddingsRequest) (models.EmbeddingsResponse, error) {
	if len(req.Input) == 0 {
		return models.EmbeddingsResponse{}, apperror.Validation("openai.Embed", "embeddings input required")
	}
	params := openai.EmbeddingNewParams{Model: openai.EmbeddingModel(req.Model)}
	if len(req.Input) == 1 {
		params.Input.OfString = param.NewOpt(req.Input[0])
	} else {
		params.Input.OfArrayOfStrings = append(params.Input.OfArrayOfStrings, req.Input...)
	}
	resp, err := a.client.Embeddings.New(ctx, params)
	if err != nil {
		return models.EmbeddingsResponse{}, err
	}
	return convertEmbeddingsResponse(*resp), nil
}

// Moderate executes a moderation request using the Moderations API.
func (a *Adapter) Moderate(ctx context.Context, req models.ModerationRequest) (models.ModerationResponse, error) {
	if len(req.Input) == 0 {
		return models.ModerationResponse{}, apperror.Validation("openai.Moderate", "moderation input required")
	}
	params := openai.ModerationNewParams{
		Model: openai.ModerationModel(req.Model),
	}
	if len(req.Input) == 1 {
		params.Input.OfString = param.NewOpt(req.Input[0])
	} else {
		params.Input.OfStringArray = append(params.Input.OfStringArray, req.Input...)
	}
	resp, err := a.client.Moderations.New(ctx, params)
	if err != nil {
		return models.ModerationResponse{}, err
	}
	return convertModerationResponse(*resp), nil
}

// Generate produces images with the Images API.
func (a *Adapter) Generate(ctx context.Context, req models.ImageRequest) (models.ImageResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.ImageResponse{}, apperror.Validation("openai.Generate", "prompt required")
	}
	params := openai.ImageGenerateParams{
		Model:  openai.ImageModel(req.Model),
		Prompt: prompt,
	}
	if req.N > 0 {
		params.N = param.NewOpt(int64(req.N))
	}
	if req.Size != "" {
		params.Size = openai.ImageGenerateParamsSize(req.Size)
	}
	if req.ResponseFormat != "" {
		params.ResponseFormat = openai.ImageGenerateParamsResponseFormat(req.ResponseFormat)
	}
	if req.User != "" {
		params.User = param.NewOpt(req.User)
	}
	resp, err := a.client.Images.Generate(ctx, params)
	if err != nil {
		return models.ImageResponse{}, err
	}
	return convertImageResponse(*resp), nil
}

// Edit performs an image edit/extension request via the Images API.
func (a *Adapter) Edit(ctx context.Context, req models.ImageEditRequest) (models.ImageResponse, error) {
	if len(req.Images) == 0 {
		return models.ImageResponse{}, apperror.Validation("openai.Edit", "at least one image is required for edits")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.ImageResponse{}, apperror.Validation("openai.Edit", "prompt required for image edits")
	}
	params := openai.ImageEditParams{
		Model:  openai.ImageModel(req.Model),
		Prompt: prompt,
	}
	if req.N > 0 {
		params.N = param.NewOpt(int64(req.N))
	}
	if req.Size != "" {
		params.Size = openai.ImageEditParamsSize(req.Size)
	}
	if req.ResponseFormat != "" {
		params.ResponseFormat = openai.ImageEditParamsResponseFormat(req.ResponseFormat)
	}
	if req.Quality != "" {
		params.Quality = openai.ImageEditParamsQuality(req.Quality)
	}
	if req.Background != "" {
		params.Background = openai.ImageEditParamsBackground(req.Background)
	}
	if req.User != "" {
		params.User = param.NewOpt(req.User)
	}
	readers := make([]io.ReadCloser, 0, len(req.Images))
	cleanup := func() {
		for _, r := range readers {
			_ = r.Close()
		}
	}
	defer cleanup()
	if len(req.Images) == 1 {
		reader := req.Images[0].Reader()
		readers = append(readers, reader)
		params.Image.OfFile = reader
	} else {
		params.Image.OfFileArray = make([]io.Reader, 0, len(req.Images))
		for _, img := range req.Images {
			reader := img.Reader()
			readers = append(readers, reader)
			params.Image.OfFileArray = append(params.Image.OfFileArray, reader)
		}
	}
	if req.Mask != nil {
		mask := req.Mask.Reader()
		readers = append(readers, mask)
		params.Mask = mask
	}
	resp, err := a.client.Images.Edit(ctx, params)
	if err != nil {
		return models.ImageResponse{}, err
	}
	return convertImageResponse(*resp), nil
}

// Variation creates image variations via the Images API.
func (a *Adapter) Variation(ctx context.Context, req models.ImageVariationRequest) (models.ImageResponse, error) {
	reader := req.Image.Reader()
	defer reader.Close()
	if reader == nil {
		return models.ImageResponse{}, apperror.Validation("openai.Variation", "image input required for variations")
	}
	params := openai.ImageNewVariationParams{
		Image: reader,
		Model: openai.ImageModel(req.Model),
	}
	if req.N > 0 {
		params.N = param.NewOpt(int64(req.N))
	}
	if req.Size != "" {
		params.Size = openai.ImageNewVariationParamsSize(req.Size)
	}
	if req.ResponseFormat != "" {
		params.ResponseFormat = openai.ImageNewVariationParamsResponseFormat(req.ResponseFormat)
	}
	if req.User != "" {
		params.User = param.NewOpt(req.User)
	}
	resp, err := a.client.Images.NewVariation(ctx, params)
	if err != nil {
		return models.ImageResponse{}, err
	}
	return convertImageResponse(*resp), nil
}

// Models lists available models from the OpenAI API.
func (a *Adapter) Models(ctx context.Context) ([]models.Model, error) {
	page, err := a.client.Models.List(ctx)
	if err != nil {
		return nil, err
	}
	return convertModelPage(page), nil
}

// HealthCheck uses the Models API as a lightweight readiness probe.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	_, err := a.client.Models.List(ctx)
	return err
}

// Transcribe performs speech-to-text via the OpenAI Audio Transcriptions API.
func (a *Adapter) Transcribe(ctx context.Context, req models.AudioTranscriptionRequest) (models.AudioTranscriptionResponse, error) {
	if req.Input.Reader == nil {
		return models.AudioTranscriptionResponse{}, apperror.Validation("openai.Transcribe", "audio input required")
	}
	params := openai.AudioTranscriptionNewParams{
		File:  req.Input.Reader,
		Model: openai.AudioModel(req.Model),
	}
	if lang := strings.TrimSpace(req.Language); lang != "" {
		params.Language = openai.String(lang)
	}
	if prompt := strings.TrimSpace(req.Prompt); prompt != "" {
		params.Prompt = openai.String(prompt)
	}
	if req.Temperature != nil {
		params.Temperature = openai.Float(float64(*req.Temperature))
	}
	if len(req.TimestampGranularities) > 0 {
		params.TimestampGranularities = make([]string, 0, len(req.TimestampGranularities))
		for _, gran := range req.TimestampGranularities {
			params.TimestampGranularities = append(params.TimestampGranularities, string(gran))
		}
	}
	format := req.ResponseFormat
	if format == "" {
		format = models.AudioResponseFormatJSON
	}
	if format != "" {
		params.ResponseFormat = openai.AudioResponseFormat(format)
	}
	if !format.IsJSONFormat() {
		return a.invokeAudioRaw(ctx, "audio/transcriptions", params, format)
	}
	resp, err := a.client.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return models.AudioTranscriptionResponse{}, err
	}
	payload := []byte(resp.RawJSON())
	return models.AudioTranscriptionResponse{
		Format:      format,
		ContentType: format.ContentType(),
		Payload:     payload,
		Text:        resp.Text,
		Usage:       convertAudioUsage(resp.Usage),
	}, nil
}

func (a *Adapter) TranscribeStream(ctx context.Context, req models.AudioTranscriptionRequest) (<-chan models.AudioTranscriptionStreamChunk, func() error, error) {
	if req.Input.Reader == nil {
		return nil, nil, apperror.Validation("openai.TranscribeStream", "audio input required")
	}
	params := openai.AudioTranscriptionNewParams{
		File:  req.Input.Reader,
		Model: openai.AudioModel(req.Model),
	}
	if lang := strings.TrimSpace(req.Language); lang != "" {
		params.Language = openai.String(lang)
	}
	if prompt := strings.TrimSpace(req.Prompt); prompt != "" {
		params.Prompt = openai.String(prompt)
	}
	if req.Temperature != nil {
		params.Temperature = openai.Float(float64(*req.Temperature))
	}
	if len(req.TimestampGranularities) > 0 {
		params.TimestampGranularities = make([]string, 0, len(req.TimestampGranularities))
		for _, gran := range req.TimestampGranularities {
			params.TimestampGranularities = append(params.TimestampGranularities, string(gran))
		}
	}
	format := req.ResponseFormat
	if format == "" {
		format = models.AudioResponseFormatJSON
	}
	params.ResponseFormat = openai.AudioResponseFormat(format)

	stream := a.client.Audio.Transcriptions.NewStreaming(ctx, params)
	if err := stream.Err(); err != nil {
		stream.Close()
		return nil, nil, err
	}

	chunks := make(chan models.AudioTranscriptionStreamChunk)
	cancel := func() error {
		return stream.Close()
	}

	go func() {
		defer close(chunks)
		defer stream.Close()
		for stream.Next() {
			event := stream.Current()
			payload := event.RawJSON()
			chunk := models.AudioTranscriptionStreamChunk{
				Payload: []byte(payload),
			}
			switch event.Type {
			case "transcript.text.done":
				done := event.AsTranscriptTextDone()
				usage := convertStreamUsage(done.Usage)
				chunk.Usage = &usage
				chunk.Done = true
			}
			select {
			case <-ctx.Done():
				return
			case chunks <- chunk:
			}
		}
		if err := stream.Err(); err != nil {
			select {
			case <-ctx.Done():
				return
			case chunks <- models.AudioTranscriptionStreamChunk{Err: err}:
			}
		}
	}()

	return chunks, cancel, nil
}

// Translate performs speech translation using the OpenAI Audio Translations API.
func (a *Adapter) Translate(ctx context.Context, req models.AudioTranscriptionRequest) (models.AudioTranscriptionResponse, error) {
	if req.Input.Reader == nil {
		return models.AudioTranscriptionResponse{}, apperror.Validation("openai.Translate", "audio input required")
	}
	params := openai.AudioTranslationNewParams{
		File:  req.Input.Reader,
		Model: openai.AudioModel(req.Model),
	}
	if prompt := strings.TrimSpace(req.Prompt); prompt != "" {
		params.Prompt = openai.String(prompt)
	}
	if req.Temperature != nil {
		params.Temperature = openai.Float(float64(*req.Temperature))
	}
	format := req.ResponseFormat
	if format == "" {
		format = models.AudioResponseFormatJSON
	}
	switch format {
	case models.AudioResponseFormatJSON, models.AudioResponseFormatVerboseJSON, models.AudioResponseFormatText, models.AudioResponseFormatSRT, models.AudioResponseFormatVTT:
	default:
		return models.AudioTranscriptionResponse{}, apperror.Validation("openai.Translate", "response format not supported for translations")
	}
	if format != "" {
		params.ResponseFormat = openai.AudioTranslationNewParamsResponseFormat(format)
	}
	if !format.IsJSONFormat() {
		return a.invokeAudioRaw(ctx, "audio/translations", params, format)
	}
	resp, err := a.client.Audio.Translations.New(ctx, params)
	if err != nil {
		return models.AudioTranscriptionResponse{}, err
	}
	payload := []byte(resp.RawJSON())
	return models.AudioTranscriptionResponse{
		Format:      format,
		ContentType: format.ContentType(),
		Payload:     payload,
		Text:        resp.Text,
	}, nil
}

func (a *Adapter) Synthesize(ctx context.Context, req models.AudioSpeechRequest) (models.AudioSpeechResponse, error) {
	input := strings.TrimSpace(req.Input)
	if input == "" {
		return models.AudioSpeechResponse{}, apperror.Validation("openai.Synthesize", "input is required for speech synthesis")
	}
	voice := strings.TrimSpace(req.Voice)
	if voice == "" {
		voice = "alloy"
	}
	format := strings.TrimSpace(req.Format)
	if format == "" {
		format = "mp3"
	}
	params := openai.AudioSpeechNewParams{
		Model: openai.SpeechModel(req.Model),
		Input: input,
		Voice: openai.AudioSpeechNewParamsVoice(voice),
		StreamFormat: func() openai.AudioSpeechNewParamsStreamFormat {
			if strings.EqualFold(req.StreamFormat, "audio") {
				return openai.AudioSpeechNewParamsStreamFormatAudio
			}
			if strings.EqualFold(req.StreamFormat, "sse") {
				return openai.AudioSpeechNewParamsStreamFormatSSE
			}
			return ""
		}(),
	}
	params.ResponseFormat = openai.AudioSpeechNewParamsResponseFormat(format)
	resp, err := a.client.Audio.Speech.New(ctx, params)
	if err != nil {
		return models.AudioSpeechResponse{}, err
	}
	defer resp.Body.Close()
	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.AudioSpeechResponse{}, err
	}
	return models.AudioSpeechResponse{Audio: audioBytes}, nil
}

func (a *Adapter) SynthesizeStream(ctx context.Context, req models.AudioSpeechRequest) (<-chan models.AudioSpeechChunk, func() error, error) {
	return nil, nil, apperror.Internal("openai.SynthesizeStream", "streaming speech not implemented")
}

func (a *Adapter) invokeAudioRaw(ctx context.Context, path string, body any, format models.AudioResponseFormat) (models.AudioTranscriptionResponse, error) {
	if format == "" {
		format = models.AudioResponseFormatJSON
	}
	var rawResp *http.Response
	var payload []byte
	opts := []option.RequestOption{
		option.WithHeader("Accept", "*/*"),
		option.WithResponseInto(&rawResp),
	}
	if err := a.client.Execute(ctx, http.MethodPost, path, body, &payload, opts...); err != nil {
		return models.AudioTranscriptionResponse{}, err
	}
	contentType := format.ContentType()
	if rawResp != nil && rawResp.Header != nil {
		if header := strings.TrimSpace(rawResp.Header.Get("Content-Type")); header != "" {
			contentType = header
		}
	}
	text := ""
	if format == models.AudioResponseFormatText {
		text = string(payload)
	}
	return models.AudioTranscriptionResponse{
		Format:      format,
		ContentType: contentType,
		Payload:     payload,
		Text:        text,
	}, nil
}

func convertModelPage(page *pagination.Page[openai.Model]) []models.Model {
	if page == nil {
		return nil
	}
	out := make([]models.Model, 0, len(page.Data))
	for _, item := range page.Data {
		out = append(out, models.Model{
			Alias:         item.ID,
			Provider:      "openai",
			ProviderModel: item.ID,
			Modalities:    nil,
			SupportsTools: true,
		})
	}
	return out
}

func convertAudioUsage(usage openai.AudioTranscriptionNewResponseUnionUsage) models.Usage {
	if usage.Type != "tokens" && usage.Type != "duration" && usage.InputTokens == 0 && usage.OutputTokens == 0 && usage.TotalTokens == 0 {
		return models.Usage{}
	}
	return models.Usage{
		PromptTokens:     int32(usage.InputTokens),
		CompletionTokens: int32(usage.OutputTokens),
		TotalTokens:      int32(usage.TotalTokens),
	}
}

func convertStreamUsage(usage openai.TranscriptionTextDoneEventUsage) models.Usage {
	return models.Usage{
		PromptTokens:     int32(usage.InputTokens),
		CompletionTokens: int32(usage.OutputTokens),
		TotalTokens:      int32(usage.TotalTokens),
	}
}

func convertChatResponse(resp openai.ChatCompletion) models.ChatResponse {
	choices := make([]models.ChatChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		reasoning, reasoningContent := extractReasoning(choice.Message.JSON.ExtraFields, choice.Message.RawJSON())
		content, parts := openaihelper.ExtractMessageContent(choice.Message.RawJSON(), choice.Message.Content)
		message := models.ChatMessage{
			Role:             string(choice.Message.Role),
			Content:          content,
			ContentParts:     parts,
			Reasoning:        reasoning,
			ReasoningContent: reasoningContent,
		}

		// Extract tool_calls if present
		if len(choice.Message.ToolCalls) > 0 {
			message.ToolCalls = make([]models.ToolCall, 0, len(choice.Message.ToolCalls))
			for _, tc := range choice.Message.ToolCalls {
				message.ToolCalls = append(message.ToolCalls, models.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: models.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}

		if strings.TrimSpace(message.Content) == "" && message.ReasoningContent != "" {
			message.Content = message.ReasoningContent
		}
		choices = append(choices, models.ChatChoice{
			Index:        int(choice.Index),
			Message:      message,
			FinishReason: choice.FinishReason,
		})
	}

	usage := models.Usage{
		PromptTokens:     int32(resp.Usage.PromptTokens),
		CompletionTokens: int32(resp.Usage.CompletionTokens),
		ReasoningTokens:  int32(resp.Usage.CompletionTokensDetails.ReasoningTokens),
		TotalTokens:      int32(resp.Usage.TotalTokens),
	}

	return models.ChatResponse{
		ID:      resp.ID,
		Created: time.Unix(resp.Created, 0),
		Model:   resp.Model,
		Choices: choices,
		Usage:   usage,
	}
}

func convertChatChunk(chunk openai.ChatCompletionChunk) models.ChatChunk {
	choices := make([]models.ChunkDelta, 0, len(chunk.Choices))
	for _, choice := range chunk.Choices {
		reasoning, reasoningContent := extractReasoning(choice.Delta.JSON.ExtraFields, choice.Delta.RawJSON())
		content, parts := openaihelper.ExtractMessageContent(choice.Delta.RawJSON(), choice.Delta.Content)
		msg := models.ChatMessage{
			Role:             choice.Delta.Role,
			Content:          content,
			ContentParts:     parts,
			Reasoning:        reasoning,
			ReasoningContent: reasoningContent,
		}

		// Extract streaming tool_calls if present
		if len(choice.Delta.ToolCalls) > 0 {
			msg.ToolCalls = make([]models.ToolCall, 0, len(choice.Delta.ToolCalls))
			for _, tc := range choice.Delta.ToolCalls {
				idx := int(tc.Index)
				msg.ToolCalls = append(msg.ToolCalls, models.ToolCall{
					ID:    tc.ID,
					Type:  tc.Type,
					Index: &idx,
					Function: models.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}

		if strings.TrimSpace(msg.Content) == "" && msg.ReasoningContent != "" {
			msg.Content = msg.ReasoningContent
		}
		choices = append(choices, models.ChunkDelta{
			Index:        int(choice.Index),
			Delta:        msg,
			FinishReason: choice.FinishReason,
		})
	}

	return models.ChatChunk{
		ID:      chunk.ID,
		Model:   chunk.Model,
		Created: time.Unix(chunk.Created, 0),
		Choices: choices,
		Usage:   convertUsagePointer(chunk.Usage),
	}
}

func convertUsagePointer(u openai.CompletionUsage) *models.Usage {
	if u.PromptTokens == 0 && u.CompletionTokens == 0 && u.TotalTokens == 0 {
		return nil
	}
	usage := models.Usage{
		PromptTokens:     int32(u.PromptTokens),
		CompletionTokens: int32(u.CompletionTokens),
		ReasoningTokens:  int32(u.CompletionTokensDetails.ReasoningTokens),
		TotalTokens:      int32(u.TotalTokens),
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return &usage
}

func extractReasoning(extra map[string]respjson.Field, raw string) (string, string) {
	decodeField := func(field respjson.Field) string {
		if !field.Valid() {
			return ""
		}
		return decodeRawValue([]byte(field.Raw()))
	}
	reasoning := decodeField(extra["reasoning"])
	reasoningContent := decodeField(extra["reasoning_content"])
	if (reasoning == "" || reasoningContent == "") && strings.TrimSpace(raw) != "" {
		var payload map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &payload); err == nil {
			if reasoning == "" {
				reasoning = decodeRawValue(payload["reasoning"])
			}
			if reasoningContent == "" {
				reasoningContent = decodeRawValue(payload["reasoning_content"])
			}
		}
	}
	return reasoning, reasoningContent
}

func decodeRawValue(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if string(data) == "null" {
		return ""
	}
	var out string
	if err := json.Unmarshal(data, &out); err == nil {
		return out
	}
	return strings.Trim(string(data), "\"")
}

func convertImageResponse(resp openai.ImagesResponse) models.ImageResponse {
	data := make([]models.ImageData, 0, len(resp.Data))
	for _, item := range resp.Data {
		data = append(data, models.ImageData{
			B64JSON:       item.B64JSON,
			URL:           item.URL,
			RevisedPrompt: item.RevisedPrompt,
		})
	}

	usage := models.Usage{}
	if resp.Usage.JSON.InputTokens.Valid() {
		usage.PromptTokens = int32(resp.Usage.InputTokens)
	}
	if resp.Usage.JSON.OutputTokens.Valid() {
		usage.CompletionTokens = int32(resp.Usage.OutputTokens)
	}
	if resp.Usage.JSON.TotalTokens.Valid() {
		usage.TotalTokens = int32(resp.Usage.TotalTokens)
	} else {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if len(data) > 0 {
		usage.ImageCount = int32(len(data))
	}

	return models.ImageResponse{
		Created: time.Unix(resp.Created, 0),
		Data:    data,
		Usage:   usage,
	}
}

func convertEmbeddingsResponse(resp openai.CreateEmbeddingResponse) models.EmbeddingsResponse {
	embeddings := make([]models.Embedding, 0, len(resp.Data))
	for _, item := range resp.Data {
		vec := make([]float32, len(item.Embedding))
		for i, v := range item.Embedding {
			vec[i] = float32(v)
		}
		embeddings = append(embeddings, models.Embedding{Index: int(item.Index), Vector: vec})
	}
	usage := models.Usage{
		PromptTokens: int32(resp.Usage.PromptTokens),
		TotalTokens:  int32(resp.Usage.TotalTokens),
	}
	return models.EmbeddingsResponse{Model: resp.Model, Embeddings: embeddings, Usage: usage}
}

func convertModerationResponse(resp openai.ModerationNewResponse) models.ModerationResponse {
	results := make([]models.ModerationResult, 0, len(resp.Results))
	for _, item := range resp.Results {
		results = append(results, models.ModerationResult{
			Categories:                convertModerationCategories(item.Categories),
			CategoryAppliedInputTypes: convertModerationAppliedInputTypes(item.CategoryAppliedInputTypes),
			CategoryScores:            convertModerationScores(item.CategoryScores),
			Flagged:                   item.Flagged,
		})
	}
	return models.ModerationResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Results: results,
		Usage:   models.Usage{},
	}
}

func convertModerationCategories(src openai.ModerationCategories) models.ModerationCategories {
	return models.ModerationCategories{
		Harassment:            src.Harassment,
		HarassmentThreatening: src.HarassmentThreatening,
		Hate:                  src.Hate,
		HateThreatening:       src.HateThreatening,
		Illicit:               src.Illicit,
		IllicitViolent:        src.IllicitViolent,
		SelfHarm:              src.SelfHarm,
		SelfHarmInstructions:  src.SelfHarmInstructions,
		SelfHarmIntent:        src.SelfHarmIntent,
		Sexual:                src.Sexual,
		SexualMinors:          src.SexualMinors,
		Violence:              src.Violence,
		ViolenceGraphic:       src.ViolenceGraphic,
	}
}

func convertModerationScores(src openai.ModerationCategoryScores) models.ModerationCategoryScores {
	return models.ModerationCategoryScores{
		Harassment:            src.Harassment,
		HarassmentThreatening: src.HarassmentThreatening,
		Hate:                  src.Hate,
		HateThreatening:       src.HateThreatening,
		Illicit:               src.Illicit,
		IllicitViolent:        src.IllicitViolent,
		SelfHarm:              src.SelfHarm,
		SelfHarmInstructions:  src.SelfHarmInstructions,
		SelfHarmIntent:        src.SelfHarmIntent,
		Sexual:                src.Sexual,
		SexualMinors:          src.SexualMinors,
		Violence:              src.Violence,
		ViolenceGraphic:       src.ViolenceGraphic,
	}
}

func convertModerationAppliedInputTypes(src openai.ModerationCategoryAppliedInputTypes) models.ModerationCategoryAppliedInputTypes {
	return models.ModerationCategoryAppliedInputTypes{
		Harassment:            append([]string(nil), src.Harassment...),
		HarassmentThreatening: append([]string(nil), src.HarassmentThreatening...),
		Hate:                  append([]string(nil), src.Hate...),
		HateThreatening:       append([]string(nil), src.HateThreatening...),
		Illicit:               append([]string(nil), src.Illicit...),
		IllicitViolent:        append([]string(nil), src.IllicitViolent...),
		SelfHarm:              append([]string(nil), src.SelfHarm...),
		SelfHarmInstructions:  append([]string(nil), src.SelfHarmInstructions...),
		SelfHarmIntent:        append([]string(nil), src.SelfHarmIntent...),
		Sexual:                append([]string(nil), src.Sexual...),
		SexualMinors:          append([]string(nil), src.SexualMinors...),
		Violence:              append([]string(nil), src.Violence...),
		ViolenceGraphic:       append([]string(nil), src.ViolenceGraphic...),
	}
}
