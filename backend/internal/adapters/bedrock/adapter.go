package bedrock

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers/streamutil"
)

const (
	ChatFormatAnthropicMessages = "anthropic_messages"

	EmbeddingFormatTitanText = "titan_text"
)

// Options controls how the Bedrock adapter is initialised.
type Options struct {
	Region          string
	Profile         string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string

	ModelID string

	ChatFormat       string
	EmbeddingFormat  string
	DefaultMaxTokens int32
	ImageTaskType    string

	AnthropicVersion string
	EmbedDimensions  int32
	EmbedNormalize   bool

	Metadata map[string]string
}

type stableDiffusionRequest struct {
	TextPrompts   []stableDiffusionPrompt `json:"text_prompts"`
	CfgScale      float64                 `json:"cfg_scale,omitempty"`
	Height        int                     `json:"height,omitempty"`
	Width         int                     `json:"width,omitempty"`
	Samples       int                     `json:"samples,omitempty"`
	Steps         int                     `json:"steps,omitempty"`
	Seed          *int32                  `json:"seed,omitempty"`
	InitImage     string                  `json:"init_image,omitempty"`
	InitImageMode string                  `json:"init_image_mode,omitempty"`
	ImageStrength float64                 `json:"image_strength,omitempty"`
	MaskImage     string                  `json:"mask_image,omitempty"`
	MaskSource    string                  `json:"mask_source,omitempty"`
}

type stableDiffusionPrompt struct {
	Text   string  `json:"text"`
	Weight float64 `json:"weight,omitempty"`
}

type stableDiffusionResponse struct {
	Artifacts []struct {
		Base64 string `json:"base64"`
	} `json:"artifacts"`
}

// Adapter implements the provider interfaces backed by Amazon Bedrock.
type Adapter struct {
	client    *bedrockruntime.Client
	stsClient *sts.Client
	awsCfg    aws.Config
	opts      Options
}

// New creates a Bedrock adapter using the provided credentials/region.
func New(ctx context.Context, opts Options) (*Adapter, error) {
	if opts.Region == "" {
		return nil, errors.New("bedrock region required")
	}
	if opts.ModelID == "" {
		return nil, errors.New("bedrock model id required")
	}

	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(opts.Region),
	}
	if opts.Profile != "" {
		loadOpts = append(loadOpts, config.WithSharedConfigProfile(opts.Profile))
	}
	if opts.AccessKeyID != "" && opts.SecretAccessKey != "" {
		staticProvider := credentials.NewStaticCredentialsProvider(opts.AccessKeyID, opts.SecretAccessKey, opts.SessionToken)
		loadOpts = append(loadOpts, config.WithCredentialsProvider(staticProvider))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	if awsCfg.Region == "" {
		awsCfg.Region = opts.Region
	}

	client := bedrockruntime.NewFromConfig(awsCfg)
	stsClient := sts.NewFromConfig(awsCfg)

	if opts.AnthropicVersion == "" {
		opts.AnthropicVersion = "bedrock-2023-05-31"
	}
	if opts.Metadata == nil {
		opts.Metadata = map[string]string{}
	}

	return &Adapter{
		client:    client,
		stsClient: stsClient,
		awsCfg:    awsCfg,
		opts:      opts,
	}, nil
}

// Chat executes a non-streaming chat request using the configured chat format.
func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	if a.opts.ChatFormat == "" {
		return models.ChatResponse{}, errors.New("chat not supported by this bedrock route")
	}

	switch a.opts.ChatFormat {
	case ChatFormatAnthropicMessages:
		return a.chatAnthropic(ctx, req)
	default:
		return models.ChatResponse{}, fmt.Errorf("chat format %q unsupported", a.opts.ChatFormat)
	}
}

func (a *Adapter) generateTitan(ctx context.Context, req models.ImageRequest) (models.ImageResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.ImageResponse{}, errors.New("prompt required")
	}
	if req.ResponseFormat != "" && req.ResponseFormat != "b64_json" {
		return models.ImageResponse{}, errors.New("bedrock image generation currently supports only base64 responses")
	}

	width, height := parseImageSize(req.Size)
	quality := strings.TrimSpace(req.Quality)
	if quality == "" {
		quality = strings.TrimSpace(a.opts.Metadata["bedrock_image_quality"])
	}
	if quality == "" {
		quality = "standard"
	}

	style := strings.TrimSpace(req.Style)
	if style == "" {
		style = strings.TrimSpace(a.opts.Metadata["bedrock_image_style"])
	}

	cfgScale := parseFloatMetadata(a.opts.Metadata, "bedrock_image_cfg_scale", 8)
	seed := rand.New(rand.NewSource(time.Now().UnixNano())).Int31()
	if v := a.opts.Metadata["bedrock_image_seed"]; v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 32); err == nil {
			seed = int32(parsed)
		}
	}

	numberOfImages := clampImageCount(req.N, 4)
	if numberOfImages == 0 {
		numberOfImages = 1
	}

	payload := titanImageRequest{
		TaskType: a.opts.ImageTaskType,
		TextToImageParams: titanTextParams{
			Text: prompt,
		},
		ImageGenerationConfig: titanImageConfig{
			NumberOfImages: numberOfImages,
			Quality:        quality,
			CfgScale:       cfgScale,
			Height:         height,
			Width:          width,
			Seed:           seed,
			Style:          style,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return models.ImageResponse{}, fmt.Errorf("encode titan image request: %w", err)
	}

	resp, err := a.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(a.opts.ModelID),
		Body:        body,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	})
	if err != nil {
		return models.ImageResponse{}, err
	}

	var parsed titanImageResponse
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return models.ImageResponse{}, fmt.Errorf("decode titan image response: %w", err)
	}
	if len(parsed.Images) == 0 {
		return models.ImageResponse{}, errors.New("titan image response missing images")
	}

	data := make([]models.ImageData, 0, len(parsed.Images))
	for _, img := range parsed.Images {
		data = append(data, models.ImageData{B64JSON: img})
	}

	return models.ImageResponse{
		Created: time.Now().UTC(),
		Data:    data,
	}, nil
}

func (a *Adapter) generateStableDiffusion(ctx context.Context, req models.ImageRequest) (models.ImageResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.ImageResponse{}, errors.New("prompt required")
	}
	if req.ResponseFormat != "" && req.ResponseFormat != "b64_json" {
		return models.ImageResponse{}, errors.New("bedrock image generation currently supports only base64 responses")
	}
	width, height := parseImageSize(req.Size)
	if width == 0 {
		width, height = 1024, 1024
	}
	samples := clampImageCount(req.N, 4)
	if samples == 0 {
		samples = 1
	}
	cfgScale := parseFloatMetadata(a.opts.Metadata, "bedrock_image_cfg_scale", 7.5)
	steps := parseIntMetadata(a.opts.Metadata, "bedrock_image_steps", 50)
	seed := parseIntMetadata(a.opts.Metadata, "bedrock_image_seed", int(rand.New(rand.NewSource(time.Now().UnixNano())).Int31()))
	seed32 := int32(seed)
	request := stableDiffusionRequest{
		TextPrompts: []stableDiffusionPrompt{{Text: prompt, Weight: 1}},
		CfgScale:    cfgScale,
		Height:      height,
		Width:       width,
		Samples:     samples,
		Steps:       steps,
		Seed:        &seed32,
	}
	if neg := strings.TrimSpace(a.opts.Metadata["bedrock_image_negative_prompt"]); neg != "" {
		request.TextPrompts = append(request.TextPrompts, stableDiffusionPrompt{Text: neg, Weight: -1})
	}
	return a.invokeStableDiffusion(ctx, request)
}

func (a *Adapter) invokeStableDiffusion(ctx context.Context, request stableDiffusionRequest) (models.ImageResponse, error) {
	if request.Samples <= 0 {
		request.Samples = 1
	}
	body, err := json.Marshal(request)
	if err != nil {
		return models.ImageResponse{}, fmt.Errorf("encode stable diffusion request: %w", err)
	}
	resp, err := a.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(a.opts.ModelID),
		Body:        body,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	})
	if err != nil {
		return models.ImageResponse{}, err
	}
	var parsed stableDiffusionResponse
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return models.ImageResponse{}, fmt.Errorf("decode stable diffusion response: %w", err)
	}
	if len(parsed.Artifacts) == 0 {
		return models.ImageResponse{}, errors.New("bedrock diffusion response missing images")
	}
	data := make([]models.ImageData, 0, len(parsed.Artifacts))
	for _, artifact := range parsed.Artifacts {
		if strings.TrimSpace(artifact.Base64) == "" {
			continue
		}
		data = append(data, models.ImageData{B64JSON: artifact.Base64})
	}
	if len(data) == 0 {
		return models.ImageResponse{}, errors.New("bedrock diffusion response empty")
	}
	return models.ImageResponse{
		Created: time.Now().UTC(),
		Data:    data,
	}, nil
}

func (a *Adapter) imageEditStableDiffusion(ctx context.Context, req models.ImageEditRequest) (models.ImageResponse, error) {
	if len(req.Images) == 0 {
		return models.ImageResponse{}, errors.New("bedrock: at least one image is required for edits")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.ImageResponse{}, errors.New("prompt required")
	}
	base64Image, err := encodeImageInput(req.Images[0])
	if err != nil {
		return models.ImageResponse{}, err
	}
	maskMode := strings.TrimSpace(a.opts.Metadata["bedrock_image_mask_source"])
	if maskMode == "" {
		maskMode = "MASK_IMAGE_WHITE"
	}
	seed := parseIntMetadata(a.opts.Metadata, "bedrock_image_seed", int(rand.New(rand.NewSource(time.Now().UnixNano())).Int31()))
	seed32 := int32(seed)
	request := stableDiffusionRequest{
		TextPrompts: []stableDiffusionPrompt{{Text: prompt, Weight: 1}},
		CfgScale:    parseFloatMetadata(a.opts.Metadata, "bedrock_image_cfg_scale", 7.5),
		Steps:       parseIntMetadata(a.opts.Metadata, "bedrock_image_steps", 50),
		Samples:     clampImageCount(req.N, 4),
		Seed:        &seed32,
		InitImage:   base64Image,
		InitImageMode: func() string {
			mode := strings.TrimSpace(a.opts.Metadata["bedrock_image_init_mode"])
			if mode == "" {
				mode = "IMAGE_STRENGTH"
			}
			return mode
		}(),
		ImageStrength: parseFloatMetadata(a.opts.Metadata, "bedrock_image_strength", 0.35),
	}
	if request.Samples == 0 {
		request.Samples = 1
	}
	if width, height := parseImageSize(req.Size); width > 0 && height > 0 {
		request.Width = width
		request.Height = height
	}
	if req.Mask != nil {
		maskB64, err := encodeImageInput(*req.Mask)
		if err != nil {
			return models.ImageResponse{}, err
		}
		request.MaskImage = maskB64
		request.MaskSource = maskMode
	}
	if neg := strings.TrimSpace(a.opts.Metadata["bedrock_image_negative_prompt"]); neg != "" {
		request.TextPrompts = append(request.TextPrompts, stableDiffusionPrompt{Text: neg, Weight: -1})
	}
	return a.invokeStableDiffusion(ctx, request)
}

func (a *Adapter) imageVariationStableDiffusion(ctx context.Context, req models.ImageVariationRequest) (models.ImageResponse, error) {
	if len(req.Image.Data) == 0 {
		return models.ImageResponse{}, errors.New("bedrock: variation image input required")
	}
	base64Image, err := encodeImageInput(req.Image)
	if err != nil {
		return models.ImageResponse{}, err
	}
	prompt := strings.TrimSpace(a.opts.Metadata["bedrock_image_variation_prompt"])
	if prompt == "" {
		prompt = "variation of the provided image"
	}
	seed := parseIntMetadata(a.opts.Metadata, "bedrock_image_seed", int(rand.New(rand.NewSource(time.Now().UnixNano())).Int31()))
	seed32 := int32(seed)
	request := stableDiffusionRequest{
		TextPrompts: []stableDiffusionPrompt{{Text: prompt, Weight: 1}},
		CfgScale:    parseFloatMetadata(a.opts.Metadata, "bedrock_image_cfg_scale", 7.5),
		Steps:       parseIntMetadata(a.opts.Metadata, "bedrock_image_steps", 50),
		Samples:     clampImageCount(req.N, 4),
		Seed:        &seed32,
		InitImage:   base64Image,
		InitImageMode: func() string {
			mode := strings.TrimSpace(a.opts.Metadata["bedrock_image_init_mode"])
			if mode == "" {
				mode = "IMAGE_STRENGTH"
			}
			return mode
		}(),
		ImageStrength: parseFloatMetadata(a.opts.Metadata, "bedrock_image_strength", 0.35),
	}
	if request.Samples == 0 {
		request.Samples = 1
	}
	if neg := strings.TrimSpace(a.opts.Metadata["bedrock_image_negative_prompt"]); neg != "" {
		request.TextPrompts = append(request.TextPrompts, stableDiffusionPrompt{Text: neg, Weight: -1})
	}
	return a.invokeStableDiffusion(ctx, request)
}

func encodeImageInput(input models.ImageInput) (string, error) {
	if len(input.Data) == 0 {
		return "", errors.New("empty image payload")
	}
	return base64.StdEncoding.EncodeToString(input.Data), nil
}

func (a *Adapter) ChatStream(ctx context.Context, req models.ChatRequest) (<-chan models.ChatChunk, func() error, error) {
	if a.opts.ChatFormat != ChatFormatAnthropicMessages {
		return nil, nil, errors.New("streaming not supported for this bedrock route")
	}

	body, err := a.buildAnthropicBody(req)
	if err != nil {
		return nil, nil, err
	}

	input := &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(a.opts.ModelID),
		Body:        body,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	}

	resp, err := a.client.InvokeModelWithResponseStream(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	stream := resp.GetStream()
	if stream == nil {
		return nil, nil, errors.New("bedrock stream missing")
	}

	forward := func(ctx context.Context, yield streamutil.YieldFunc) {
		created := time.Now().UTC()
		messageID := fmt.Sprintf("chatcmpl-bedrock-%d", created.UnixNano())
		modelName := req.Model
		finishSent := false

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-stream.Events():
				if !ok {
					return
				}
				chunk, ok := evt.(*types.ResponseStreamMemberChunk)
				if !ok || chunk == nil {
					continue
				}

				var payload anthropicStreamEvent
				if err := json.Unmarshal(chunk.Value.Bytes, &payload); err != nil {
					continue
				}

				switch payload.Type {
				case "message_start":
					if payload.Message != nil {
						if payload.Message.ID != "" {
							messageID = payload.Message.ID
						}
						if payload.Message.Model != "" {
							modelName = payload.Message.Model
						}
					}
				case "content_block_delta":
					text := payload.DeltaText()
					if text == "" {
						continue
					}
					chunk := models.ChatChunk{
						ID:      messageID,
						Model:   modelName,
						Created: created,
						Choices: []models.ChunkDelta{{
							Index: payload.Index,
							Delta: models.ChatMessage{
								Role:    "assistant",
								Content: text,
							},
						}},
					}
					if !yield(chunk) {
						return
					}
				case "message_delta":
					if finishSent {
						continue
					}
					finish := strings.TrimSpace(payload.StopReason())
					if finish == "" {
						continue
					}
					finish = mapAnthropicStopReason(finish)
					chunk := models.ChatChunk{
						ID:      messageID,
						Model:   modelName,
						Created: created,
						Choices: []models.ChunkDelta{{
							Index:        payload.Index,
							FinishReason: finish,
						}},
					}
					if !yield(chunk) {
						return
					}
					finishSent = true
				case "message_stop":
					if finishSent {
						return
					}
					chunk := models.ChatChunk{
						ID:      messageID,
						Model:   modelName,
						Created: created,
						Choices: []models.ChunkDelta{{
							Index:        payload.Index,
							FinishReason: "stop",
						}},
					}
					_ = yield(chunk)
					return
				}

				if payload.Usage.InputTokens > 0 || payload.Usage.OutputTokens > 0 {
					usage := models.Usage{
						PromptTokens:     payload.Usage.InputTokens,
						CompletionTokens: payload.Usage.OutputTokens,
						TotalTokens:      payload.Usage.InputTokens + payload.Usage.OutputTokens,
					}
					chunk := models.ChatChunk{
						ID:      messageID,
						Model:   modelName,
						Created: created,
						Usage:   &usage,
					}
					if !yield(chunk) {
						return
					}
				}
			}
		}
	}

	chunks, cancel := streamutil.Forward(ctx, stream.Close, forward)
	return chunks, cancel, nil
}

// Models is not supported (parity with Azure adapter behaviour).
func (a *Adapter) Models(ctx context.Context) ([]models.Model, error) {
	return nil, errors.New("bedrock models listing not implemented")
}

// Embed generates embeddings using the configured embedding format.
func (a *Adapter) Embed(ctx context.Context, req models.EmbeddingsRequest) (models.EmbeddingsResponse, error) {
	if a.opts.EmbeddingFormat == "" {
		return models.EmbeddingsResponse{}, errors.New("embeddings not supported by this bedrock route")
	}

	switch a.opts.EmbeddingFormat {
	case EmbeddingFormatTitanText:
		return a.embedTitan(ctx, req)
	default:
		return models.EmbeddingsResponse{}, fmt.Errorf("embedding format %q unsupported", a.opts.EmbeddingFormat)
	}
}

func (a *Adapter) Generate(ctx context.Context, req models.ImageRequest) (models.ImageResponse, error) {
	task := strings.ToLower(strings.TrimSpace(a.opts.ImageTaskType))
	switch {
	case task == "":
		return models.ImageResponse{}, errors.New("image generation not supported for this bedrock route")
	case strings.Contains(task, "stability"), strings.Contains(task, "diffusion"):
		return a.generateStableDiffusion(ctx, req)
	default:
		return a.generateTitan(ctx, req)
	}
}

func (a *Adapter) Edit(ctx context.Context, req models.ImageEditRequest) (models.ImageResponse, error) {
	task := strings.ToLower(strings.TrimSpace(a.opts.ImageTaskType))
	switch {
	case strings.Contains(task, "stability"), strings.Contains(task, "diffusion"):
		return a.imageEditStableDiffusion(ctx, req)
	default:
		return models.ImageResponse{}, models.ErrImageOperationUnsupported
	}
}

func (a *Adapter) Variation(ctx context.Context, req models.ImageVariationRequest) (models.ImageResponse, error) {
	task := strings.ToLower(strings.TrimSpace(a.opts.ImageTaskType))
	switch {
	case strings.Contains(task, "stability"), strings.Contains(task, "diffusion"):
		return a.imageVariationStableDiffusion(ctx, req)
	default:
		return models.ImageResponse{}, models.ErrImageOperationUnsupported
	}
}

// HealthCheck simply verifies the AWS client can be used (no-op to avoid extra inference costs).
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if a.stsClient == nil {
		return errors.New("bedrock sts client not initialised")
	}
	_, err := a.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	return err
}

func (a *Adapter) chatAnthropic(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return models.ChatResponse{}, errors.New("at least one message is required")
	}

	body, err := a.buildAnthropicBody(req)
	if err != nil {
		return models.ChatResponse{}, err
	}

	input := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(a.opts.ModelID),
		Body:        body,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	}

	out, err := a.client.InvokeModel(ctx, input)
	if err != nil {
		return models.ChatResponse{}, err
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(out.Body, &parsed); err != nil {
		return models.ChatResponse{}, fmt.Errorf("decode bedrock response: %w", err)
	}

	content := parsed.JoinText()
	resp := models.ChatResponse{
		ID:      parsed.ID,
		Created: time.Now().UTC(),
		Model:   req.Model,
		Choices: []models.ChatChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: mapAnthropicStopReason(parsed.StopReason),
			},
		},
		Usage: models.Usage{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
			TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}

	return resp, nil
}

func (a *Adapter) embedTitan(ctx context.Context, req models.EmbeddingsRequest) (models.EmbeddingsResponse, error) {
	if len(req.Input) == 0 {
		return models.EmbeddingsResponse{}, errors.New("embedding input required")
	}

	embeddings := make([]models.Embedding, 0, len(req.Input))
	var totalTokens int32

	for idx, text := range req.Input {
		body := titanEmbedRequest{
			InputText: strings.TrimSpace(text),
		}
		if body.InputText == "" {
			return models.EmbeddingsResponse{}, fmt.Errorf("input %d is empty", idx)
		}
		if a.opts.EmbedDimensions > 0 {
			body.Dimensions = a.opts.EmbedDimensions
		}
		if a.opts.EmbedNormalize {
			body.Normalize = aws.Bool(true)
		}

		raw, err := json.Marshal(body)
		if err != nil {
			return models.EmbeddingsResponse{}, fmt.Errorf("encode titan request: %w", err)
		}

		invoke := &bedrockruntime.InvokeModelInput{
			ModelId:     aws.String(a.opts.ModelID),
			Body:        raw,
			ContentType: aws.String("application/json"),
			Accept:      aws.String("application/json"),
		}

		out, err := a.client.InvokeModel(ctx, invoke)
		if err != nil {
			return models.EmbeddingsResponse{}, err
		}

		vector, tokens, err := parseTitanEmbedding(out.Body)
		if err != nil {
			return models.EmbeddingsResponse{}, err
		}

		embeddings = append(embeddings, models.Embedding{
			Index:  idx,
			Vector: vector,
		})
		totalTokens += tokens
	}

	return models.EmbeddingsResponse{
		Model:      req.Model,
		Embeddings: embeddings,
		Usage: models.Usage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}, nil
}

func (a *Adapter) buildAnthropicBody(req models.ChatRequest) ([]byte, error) {
	var systemPrompts []string
	messages := make([]anthropicMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		switch strings.ToLower(msg.Role) {
		case "system":
			systemPrompts = append(systemPrompts, msg.Content)
		case "assistant":
			messages = append(messages, anthropicMessage{
				Role: "assistant",
				Content: []anthropicContent{
					{Type: "text", Text: msg.Content},
				},
			})
		default:
			messages = append(messages, anthropicMessage{
				Role: "user",
				Content: []anthropicContent{
					{Type: "text", Text: msg.Content},
				},
			})
		}
	}

	body := anthropicRequest{
		AnthropicVersion: a.opts.AnthropicVersion,
		Messages:         messages,
	}

	maxTokens := int32(0)
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	} else if a.opts.DefaultMaxTokens > 0 {
		maxTokens = a.opts.DefaultMaxTokens
	}
	if maxTokens == 0 {
		maxTokens = 1024
	}
	body.MaxTokens = maxTokens

	if len(systemPrompts) > 0 {
		body.System = strings.Join(systemPrompts, "\n")
	}
	if req.Temperature != nil {
		body.Temperature = float64(*req.Temperature)
	}
	if req.TopP != nil {
		body.TopP = float64(*req.TopP)
	}
	if len(req.Stop) > 0 {
		body.StopSequences = append(body.StopSequences, req.Stop...)
	}

	return json.Marshal(body)
}

// anthropicRequest models the payload expected by Claude 3 on Bedrock.
type anthropicRequest struct {
	AnthropicVersion string             `json:"anthropic_version"`
	System           string             `json:"system,omitempty"`
	Messages         []anthropicMessage `json:"messages"`
	MaxTokens        int32              `json:"max_tokens"`
	Temperature      float64            `json:"temperature,omitempty"`
	TopP             float64            `json:"top_p,omitempty"`
	StopSequences    []string           `json:"stop_sequences,omitempty"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int32 `json:"input_tokens"`
	OutputTokens int32 `json:"output_tokens"`
}

type anthropicResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []anthropicContent `json:"content"`
	StopReason string             `json:"stop_reason"`
	Usage      anthropicUsage     `json:"usage"`
}

func (a anthropicResponse) JoinText() string {
	var b strings.Builder
	for _, c := range a.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}

type anthropicStreamEvent struct {
	Type    string                  `json:"type"`
	Index   int                     `json:"index"`
	Message *anthropicStreamMessage `json:"message"`
	Delta   *anthropicStreamDelta   `json:"delta"`
	Usage   anthropicUsage          `json:"usage"`
}

type anthropicStreamMessage struct {
	ID    string `json:"id"`
	Model string `json:"model"`
}

type anthropicStreamDelta struct {
	Type         string `json:"type"`
	Text         string `json:"text"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
}

func (e anthropicStreamEvent) DeltaText() string {
	if e.Delta == nil {
		return ""
	}
	return e.Delta.Text
}

func (e anthropicStreamEvent) StopReason() string {
	if e.Delta == nil {
		return ""
	}
	return e.Delta.StopReason
}

func mapAnthropicStopReason(reason string) string {
	switch reason {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	default:
		if reason == "" {
			return "stop"
		}
		return reason
	}
}

type titanEmbedRequest struct {
	InputText  string `json:"inputText"`
	Dimensions int32  `json:"dimensions,omitempty"`
	Normalize  *bool  `json:"normalize,omitempty"`
}

type titanEmbedResponse struct {
	Embedding struct {
		Embedding []float64 `json:"embedding"`
	} `json:"embedding"`
	InputTextTokenCount int32 `json:"inputTextTokenCount"`
}

func clampImageCount(n, max int) int {
	if n <= 0 {
		return 1
	}
	if n > max {
		return max
	}
	return n
}

func parseFloatMetadata(md map[string]string, key string, def float64) float64 {
	if md == nil {
		return def
	}
	if v := md[key]; strings.TrimSpace(v) != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return def
}

func parseIntMetadata(md map[string]string, key string, def int) int {
	if md == nil {
		return def
	}
	if v := md[key]; strings.TrimSpace(v) != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return def
}

func int32Ptr(v int32) *int32 {
	return &v
}

type titanEmbedResponseAlt struct {
	Embeddings []struct {
		Values []float64 `json:"values"`
	} `json:"embeddings"`
	InputTokenCount int32 `json:"inputTextTokenCount"`
}

func parseTitanEmbedding(payload []byte) ([]float32, int32, error) {
	var primary titanEmbedResponse
	if err := json.Unmarshal(payload, &primary); err == nil && len(primary.Embedding.Embedding) > 0 {
		return float64To32(primary.Embedding.Embedding), primary.InputTextTokenCount, nil
	}

	var alt titanEmbedResponseAlt
	if err := json.Unmarshal(payload, &alt); err == nil && len(alt.Embeddings) > 0 {
		return float64To32(alt.Embeddings[0].Values), alt.InputTokenCount, nil
	}

	return nil, 0, errors.New("unexpected titan embedding response")
}

func float64To32(values []float64) []float32 {
	result := make([]float32, len(values))
	for i, v := range values {
		result[i] = float32(v)
	}
	return result
}

type titanImageRequest struct {
	TaskType              string           `json:"taskType"`
	TextToImageParams     titanTextParams  `json:"textToImageParams"`
	ImageGenerationConfig titanImageConfig `json:"imageGenerationConfig"`
}

type titanTextParams struct {
	Text string `json:"text"`
}

type titanImageConfig struct {
	NumberOfImages int     `json:"numberOfImages"`
	Quality        string  `json:"quality"`
	CfgScale       float64 `json:"cfgScale"`
	Height         int     `json:"height"`
	Width          int     `json:"width"`
	Seed           int32   `json:"seed"`
	Style          string  `json:"style,omitempty"`
}

type titanImageResponse struct {
	Images []string `json:"images"`
}

func parseImageSize(size string) (int, int) {
	width, height := 1024, 1024
	if size != "" {
		parts := strings.Split(size, "x")
		if len(parts) == 2 {
			if w, err := strconv.Atoi(parts[0]); err == nil {
				width = w
			}
			if h, err := strconv.Atoi(parts[1]); err == nil {
				height = h
			}
		}
	}
	return clampImageDimension(width), clampImageDimension(height)
}

func clampImageDimension(value int) int {
	if value < 256 {
		value = 256
	}
	if value > 1024 {
		value = 1024
	}
	if rem := value % 64; rem != 0 {
		value -= rem
	}
	if value < 256 {
		value = 256
	}
	return value
}
