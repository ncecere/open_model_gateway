package azureopenai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/azure"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"

	openaihelper "github.com/ncecere/open_model_gateway/backend/internal/adapters/openaihelper"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers/streamutil"
)

// Adapter wraps the official OpenAI Go SDK configured for Azure endpoints.
type Adapter struct {
	client     *openai.Client
	httpClient *http.Client
	endpoint   string
	apiKey     string
	apiVersion string
}

type Options struct {
	Endpoint   string
	APIKey     string
	APIVersion string
	Extra      []option.RequestOption
}

// New creates a new Azure adapter using the provided endpoint, api key, and api version.
func New(opts Options) (*Adapter, error) {
	if opts.Endpoint == "" {
		return nil, errors.New("azure openai endpoint required")
	}
	if opts.APIKey == "" {
		return nil, errors.New("azure openai api key required")
	}
	if opts.APIVersion == "" {
		opts.APIVersion = "2024-07-01-preview"
	}

	endpoint := strings.TrimSuffix(opts.Endpoint, "/")

	options := []option.RequestOption{
		azure.WithEndpoint(endpoint, opts.APIVersion),
		azure.WithAPIKey(opts.APIKey),
	}
	options = append(options, opts.Extra...)

	client := openai.NewClient(options...)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	return &Adapter{
		client:     &client,
		httpClient: httpClient,
		endpoint:   endpoint,
		apiKey:     opts.APIKey,
		apiVersion: opts.APIVersion,
	}, nil
}

// Chat performs a non-streaming chat completion request against Azure OpenAI.
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
			if len(chunk.Choices) == 0 {
				slog.Info("azure stream usage chunk",
					"prompt_tokens", chunk.Usage.PromptTokens,
					"completion_tokens", chunk.Usage.CompletionTokens,
					"total_tokens", chunk.Usage.TotalTokens,
					"usage_field_present", chunk.JSON.Usage.Valid())
			}
			if !yield(convertChatChunk(chunk)) {
				return
			}
		}
	}

	chunks, cancel := streamutil.Forward(ctx, stream.Close, forward)
	return chunks, cancel, nil
}

// Embed creates embeddings using an Azure OpenAI deployment.
func (a *Adapter) Embed(ctx context.Context, req models.EmbeddingsRequest) (models.EmbeddingsResponse, error) {
	if len(req.Input) == 0 {
		return models.EmbeddingsResponse{}, errors.New("embedding input required")
	}

	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(req.Model),
	}

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

// Moderate forwards a moderation request to Azure OpenAI.
func (a *Adapter) Moderate(ctx context.Context, req models.ModerationRequest) (models.ModerationResponse, error) {
	if len(req.Input) == 0 {
		return models.ModerationResponse{}, errors.New("moderation input required")
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
	// Azure currently exposes moderations via the shared OpenAI client
	if err != nil {
		return models.ModerationResponse{}, err
	}
	return convertModerationResponse(*resp), nil
}

// Models list is not yet supported via Azure OpenAI REST.
func (a *Adapter) Models(ctx context.Context) ([]models.Model, error) {
	return nil, errors.New("azure models listing not implemented")
}

// HealthCheck makes a lightweight GET request against the Azure deployments endpoint.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/openai/deployments?api-version=%s", a.endpoint, url.QueryEscape(a.apiVersion))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("api-key", a.apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("azure health check status %d", resp.StatusCode)
	}
	return nil
}

// Generate produces images via the Azure OpenAI deployments.
func (a *Adapter) Generate(ctx context.Context, req models.ImageRequest) (models.ImageResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.ImageResponse{}, errors.New("prompt required")
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
	if req.Quality != "" {
		params.Quality = openai.ImageGenerateParamsQuality(req.Quality)
	}
	if req.User != "" {
		params.User = param.NewOpt(req.User)
	}
	if req.Background != "" {
		params.Background = openai.ImageGenerateParamsBackground(req.Background)
	}
	if req.Style != "" {
		params.Style = openai.ImageGenerateParamsStyle(req.Style)
	}

	resp, err := a.client.Images.Generate(ctx, params)
	if err != nil {
		return models.ImageResponse{}, err
	}

	return convertImageResponse(*resp), nil
}

// Edit is not supported for Azure OpenAI image routes (only generation is
// available today).
func (a *Adapter) Edit(ctx context.Context, req models.ImageEditRequest) (models.ImageResponse, error) {
	return models.ImageResponse{}, models.ErrImageOperationUnsupported
}

// Variation is not supported for Azure OpenAI image routes.
func (a *Adapter) Variation(ctx context.Context, req models.ImageVariationRequest) (models.ImageResponse, error) {
	return models.ImageResponse{}, models.ErrImageOperationUnsupported
}

func (a *Adapter) Transcribe(ctx context.Context, req models.AudioTranscriptionRequest) (models.AudioTranscriptionResponse, error) {
	if req.Input.Reader == nil {
		return models.AudioTranscriptionResponse{}, errors.New("azure openai: audio input required")
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

func (a *Adapter) Translate(ctx context.Context, req models.AudioTranscriptionRequest) (models.AudioTranscriptionResponse, error) {
	if req.Input.Reader == nil {
		return models.AudioTranscriptionResponse{}, errors.New("azure openai: audio input required")
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
		return models.AudioTranscriptionResponse{}, errors.New("azure openai: response format not supported for translations")
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

func convertChatResponse(resp openai.ChatCompletion) models.ChatResponse {
	choices := make([]models.ChatChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		content, parts := openaihelper.ExtractMessageContent(choice.Message.RawJSON(), choice.Message.Content)
		message := models.ChatMessage{
			Role:         string(choice.Message.Role),
			Content:      content,
			ContentParts: parts,
		}

		choices = append(choices, models.ChatChoice{
			Index:        int(choice.Index),
			Message:      message,
			FinishReason: choice.FinishReason,
		})
	}

	usage := models.Usage{}
	usage.PromptTokens = int32(resp.Usage.PromptTokens)
	usage.CompletionTokens = int32(resp.Usage.CompletionTokens)
	usage.ReasoningTokens = int32(resp.Usage.CompletionTokensDetails.ReasoningTokens)
	usage.TotalTokens = int32(resp.Usage.TotalTokens)

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
		content, parts := openaihelper.ExtractMessageContent(choice.Delta.RawJSON(), choice.Delta.Content)
		msg := models.ChatMessage{
			Role:         choice.Delta.Role,
			Content:      content,
			ContentParts: parts,
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

	created := time.Unix(resp.Created, 0)

	return models.ImageResponse{
		Created: created,
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
		embeddings = append(embeddings, models.Embedding{
			Index:  int(item.Index),
			Vector: vec,
		})
	}

	usage := models.Usage{
		PromptTokens: int32(resp.Usage.PromptTokens),
		TotalTokens:  int32(resp.Usage.TotalTokens),
	}

	return models.EmbeddingsResponse{
		Model:      resp.Model,
		Embeddings: embeddings,
		Usage:      usage,
	}
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

func convertAudioUsage(usage openai.AudioTranscriptionNewResponseUnionUsage) models.Usage {
	if usage.Type != "tokens" && usage.InputTokens == 0 && usage.OutputTokens == 0 && usage.TotalTokens == 0 {
		return models.Usage{}
	}
	return models.Usage{
		PromptTokens:     int32(usage.InputTokens),
		CompletionTokens: int32(usage.OutputTokens),
		TotalTokens:      int32(usage.TotalTokens),
	}
}
