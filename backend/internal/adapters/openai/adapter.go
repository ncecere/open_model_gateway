package openai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/pagination"
	"github.com/openai/openai-go/v3/packages/param"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers/streamutil"
)

// Options configure the native OpenAI adapter.
type Options struct {
	APIKey       string
	BaseURL      string
	Organization string
	Extra        []option.RequestOption
}

// Adapter wraps the official OpenAI SDK for native + compatible deployments.
type Adapter struct {
	client     *openai.Client
	httpClient *http.Client
}

// New creates an OpenAI adapter using the provided API key and optional base URL/organization.
func New(opts Options) (*Adapter, error) {
	if strings.TrimSpace(opts.APIKey) == "" {
		return nil, errors.New("openai: api key required")
	}

	requestOpts := []option.RequestOption{option.WithAPIKey(opts.APIKey)}
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
	params := buildChatParams(req)
	resp, err := a.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return models.ChatResponse{}, err
	}
	return convertChatResponse(*resp), nil
}

// ChatStream performs a streaming chat completion request.
func (a *Adapter) ChatStream(ctx context.Context, req models.ChatRequest) (<-chan models.ChatChunk, func() error, error) {
	params := buildChatParams(req)
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
		return models.EmbeddingsResponse{}, errors.New("openai: embeddings input required")
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

// Generate produces images with the Images API.
func (a *Adapter) Generate(ctx context.Context, req models.ImageRequest) (models.ImageResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.ImageResponse{}, errors.New("openai: prompt required")
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
		return models.ImageResponse{}, errors.New("openai: at least one image is required for edits")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.ImageResponse{}, errors.New("openai: prompt required for image edits")
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
		return models.ImageResponse{}, errors.New("openai: image input required for variations")
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
		return models.AudioTranscriptionResponse{}, errors.New("openai: audio input required")
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
	resp, err := a.client.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return models.AudioTranscriptionResponse{}, err
	}
	return models.AudioTranscriptionResponse{
		Text:  resp.Text,
		Usage: convertAudioUsage(resp.Usage),
	}, nil
}

// Translate performs speech translation using the OpenAI Audio Translations API.
func (a *Adapter) Translate(ctx context.Context, req models.AudioTranscriptionRequest) (models.AudioTranscriptionResponse, error) {
	if req.Input.Reader == nil {
		return models.AudioTranscriptionResponse{}, errors.New("openai: audio input required")
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
	resp, err := a.client.Audio.Translations.New(ctx, params)
	if err != nil {
		return models.AudioTranscriptionResponse{}, err
	}
	return models.AudioTranscriptionResponse{Text: resp.Text}, nil
}

func (a *Adapter) Synthesize(ctx context.Context, req models.AudioSpeechRequest) (models.AudioSpeechResponse, error) {
	input := strings.TrimSpace(req.Input)
	if input == "" {
		return models.AudioSpeechResponse{}, errors.New("openai: input is required for speech synthesis")
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
	return nil, nil, errors.New("openai: streaming speech not implemented")
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

func buildChatParams(req models.ChatRequest) openai.ChatCompletionNewParams {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch strings.ToLower(msg.Role) {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "assistant":
			choice := openai.ChatCompletionMessageParamOfAssistant(msg.Content)
			messages = append(messages, choice)
		case "tool":
			fallthrough
		default:
			union := openai.UserMessage(msg.Content)
			if name := strings.TrimSpace(msg.Name); name != "" && union.OfUser != nil {
				union.OfUser.Name = param.NewOpt(name)
			}
			messages = append(messages, union)
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(req.Model),
		Messages: messages,
	}
	if req.Temperature != nil {
		params.Temperature = param.NewOpt(float64(*req.Temperature))
	}
	if req.TopP != nil {
		params.TopP = param.NewOpt(float64(*req.TopP))
	}
	if req.MaxTokens != nil {
		params.MaxTokens = param.NewOpt(int64(*req.MaxTokens))
	}
	if len(req.Stop) == 1 {
		params.Stop.OfString = param.NewOpt(req.Stop[0])
	} else if len(req.Stop) > 1 {
		params.Stop.OfStringArray = append(params.Stop.OfStringArray, req.Stop...)
	}
	return params
}

func convertChatResponse(resp openai.ChatCompletion) models.ChatResponse {
	choices := make([]models.ChatChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		message := models.ChatMessage{
			Role:    string(choice.Message.Role),
			Content: choice.Message.Content,
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
		msg := models.ChatMessage{
			Role:    choice.Delta.Role,
			Content: choice.Delta.Content,
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
		TotalTokens:      int32(u.TotalTokens),
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return &usage
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
