package vertex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers/streamutil"
)

const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

// Options configure the Vertex adapter.
type Options struct {
	ProjectID       string
	Location        string
	Publisher       string
	Model           string
	Endpoint        string
	CredentialsJSON []byte
	HTTPClient      *http.Client
	Metadata        map[string]string
}

// Adapter implements chat + embeddings via Vertex AI.
type Adapter struct {
	client    *http.Client
	model     string
	chatURL   string
	streamURL string
	embedURL  string
	imageURL  string
	baseURL   string
	metadata  map[string]string
}

// New creates a Vertex adapter using service-account credentials.
func New(ctx context.Context, opts Options) (*Adapter, error) {
	if opts.ProjectID == "" {
		return nil, errors.New("vertex: project id required")
	}
	if opts.Location == "" {
		return nil, errors.New("vertex: location required")
	}
	if opts.Model == "" {
		return nil, errors.New("vertex: model id required")
	}
	if len(opts.CredentialsJSON) == 0 {
		return nil, errors.New("vertex: credentials json required")
	}

	publisher := strings.TrimSpace(opts.Publisher)
	if publisher == "" {
		publisher = "google"
	}

	base := strings.TrimSpace(opts.Endpoint)
	if base == "" {
		base = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/%s/models/%s",
			opts.Location, opts.ProjectID, opts.Location, publisher, opts.Model)
	}
	base = strings.TrimSuffix(base, ":generateContent")
	base = strings.TrimSuffix(base, ":streamGenerateContent")
	base = strings.TrimSuffix(base, ":predict")
	base = strings.TrimSuffix(base, "/")

	httpClient := opts.HTTPClient
	if httpClient == nil {
		creds, err := google.CredentialsFromJSON(ctx, opts.CredentialsJSON, cloudPlatformScope)
		if err != nil {
			return nil, fmt.Errorf("vertex: load credentials: %w", err)
		}
		httpClient = oauth2.NewClient(ctx, creds.TokenSource)
	}
	if opts.Metadata == nil {
		opts.Metadata = map[string]string{}
	}

	return &Adapter{
		client:    httpClient,
		model:     opts.Model,
		baseURL:   base,
		chatURL:   base + ":generateContent",
		streamURL: base + ":streamGenerateContent",
		embedURL:  base + ":predict",
		imageURL:  base + ":predict",
		metadata:  opts.Metadata,
	}, nil
}

func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	if a.chatURL == "" {
		return models.ChatResponse{}, errors.New("vertex chat disabled for this model")
	}
	payload, err := buildGenerateContentRequest(req)
	if err != nil {
		return models.ChatResponse{}, err
	}
	var vertexResp vertexGenerateResponse
	if err := a.postJSON(ctx, a.chatURL, payload, &vertexResp); err != nil {
		return models.ChatResponse{}, err
	}
	resp, err := convertChatResponse(vertexResp, a.model)
	if err != nil {
		return models.ChatResponse{}, err
	}
	return resp, nil
}

func (a *Adapter) ChatStream(ctx context.Context, req models.ChatRequest) (<-chan models.ChatChunk, func() error, error) {
	if a.streamURL == "" {
		return nil, nil, errors.New("vertex streaming disabled for this model")
	}
	payload, err := buildGenerateContentRequest(req)
	if err != nil {
		return nil, nil, err
	}

	resp, err := a.post(ctx, a.streamURL, payload)
	if err != nil {
		return nil, nil, err
	}

	forward := func(ctx context.Context, yield streamutil.YieldFunc) {
		reader := bufio.NewReader(resp.Body)
		typeHint, err := peekNonWhitespace(reader)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				slog.Error("vertex stream peek", "error", err)
			}
			return
		}
		dec := json.NewDecoder(reader)
		if typeHint == '[' {
			// Array payload (non-streaming but chunked responses).
			if _, err := dec.Token(); err != nil {
				slog.Error("vertex stream array token", "error", err)
				return
			}
			for dec.More() {
				var chunk vertexGenerateResponse
				if err := dec.Decode(&chunk); err != nil {
					slog.Error("vertex stream array decode", "error", err)
					return
				}
				for _, part := range convertStreamChunk(chunk, a.model) {
					if !yield(part) {
						return
					}
				}
			}
			// consume closing bracket
			if _, err := dec.Token(); err != nil {
				slog.Error("vertex stream array closing", "error", err)
			}
			return
		}

		for {
			var chunk vertexGenerateResponse
			if err := dec.Decode(&chunk); err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				slog.Error("vertex stream decode", "error", err)
				return
			}
			for _, part := range convertStreamChunk(chunk, a.model) {
				if !yield(part) {
					return
				}
			}
		}
	}

	chunks, cancel := streamutil.Forward(ctx, resp.Body.Close, forward)
	return chunks, cancel, nil
}

func (a *Adapter) Embed(ctx context.Context, req models.EmbeddingsRequest) (models.EmbeddingsResponse, error) {
	if len(req.Input) == 0 {
		return models.EmbeddingsResponse{}, errors.New("vertex embeddings input required")
	}
	if a.embedURL == "" {
		return models.EmbeddingsResponse{}, errors.New("vertex embeddings disabled for this model")
	}

	payload := vertexPredictRequest{Instances: make([]vertexPredictInstance, 0, len(req.Input))}
	for _, text := range req.Input {
		payload.Instances = append(payload.Instances, vertexPredictInstance{Content: text})
	}

	var vertexResp vertexPredictResponse
	if err := a.postJSON(ctx, a.embedURL, payload, &vertexResp); err != nil {
		return models.EmbeddingsResponse{}, err
	}
	return convertEmbeddingsResponse(vertexResp, a.model)
}

func (a *Adapter) Generate(ctx context.Context, req models.ImageRequest) (models.ImageResponse, error) {
	if a.imageURL == "" {
		return models.ImageResponse{}, models.ErrImageOperationUnsupported
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.ImageResponse{}, errors.New("vertex: prompt required")
	}
	if req.ResponseFormat != "" && req.ResponseFormat != "b64_json" {
		return models.ImageResponse{}, errors.New("vertex: only b64_json responses are supported")
	}
	payload := vertexImagePredictRequest{
		Instances: []vertexImageInstance{{Prompt: prompt}},
		Parameters: vertexImageParameters{
			SampleCount: clampVertexSampleCount(req.N),
			AspectRatio: vertexAspectRatio(req.Size),
			MimeType:    "image/png",
		},
	}
	var predict vertexImagePredictResponse
	if err := a.postJSON(ctx, a.imageURL, payload, &predict); err != nil {
		return models.ImageResponse{}, err
	}
	return convertVertexImageResponse(predict)
}

func (a *Adapter) Edit(ctx context.Context, req models.ImageEditRequest) (models.ImageResponse, error) {
	if a.imageURL == "" {
		return models.ImageResponse{}, models.ErrImageOperationUnsupported
	}
	if len(req.Images) == 0 {
		return models.ImageResponse{}, errors.New("vertex: base image required for edits")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.ImageResponse{}, errors.New("vertex: prompt required")
	}
	base64Image, err := encodeVertexImage(req.Images[0])
	if err != nil {
		return models.ImageResponse{}, err
	}
	refs := []vertexReferenceImage{{
		ReferenceType:  "REFERENCE_TYPE_RAW",
		ReferenceID:    1,
		ReferenceImage: vertexReferenceImageData{BytesBase64Encoded: base64Image},
	}}
	if req.Mask != nil {
		maskB64, err := encodeVertexImage(*req.Mask)
		if err != nil {
			return models.ImageResponse{}, err
		}
		refs = append(refs, vertexReferenceImage{
			ReferenceType: "REFERENCE_TYPE_MASK",
			ReferenceImage: vertexReferenceImageData{
				BytesBase64Encoded: maskB64,
			},
			MaskImageConfig: &vertexMaskImageConfig{
				MaskMode: vertexMaskMode(a.metadata),
				Dilation: vertexMaskDilation(a.metadata),
			},
		})
	}
	params := vertexImageParameters{
		SampleCount: clampVertexSampleCount(req.N),
		MimeType:    "image/png",
		EditConfig:  vertexEditConfigFromMetadata(a.metadata),
	}
	if params.SampleCount == 0 {
		params.SampleCount = 1
	}
	if req.Mask != nil {
		mode := strings.TrimSpace(a.metadata["vertex_edit_mode"])
		if mode == "" {
			mode = "EDIT_MODE_INPAINT_INSERTION"
		}
		params.EditMode = mode
	}
	if scale := parseVertexFloat(a.metadata, "vertex_guidance_scale", 0); scale > 0 {
		params.GuidanceScale = scale
	}
	if pg := strings.TrimSpace(a.metadata["vertex_person_generation"]); pg != "" {
		params.PersonGeneration = pg
	}
	payload := vertexImagePredictRequest{
		Instances: []vertexImageInstance{{
			Prompt:          prompt,
			ReferenceImages: refs,
		}},
		Parameters: params,
	}
	var predict vertexImagePredictResponse
	if err := a.postJSON(ctx, a.imageURL, payload, &predict); err != nil {
		return models.ImageResponse{}, err
	}
	return convertVertexImageResponse(predict)
}

func (a *Adapter) Variation(ctx context.Context, req models.ImageVariationRequest) (models.ImageResponse, error) {
	if a.imageURL == "" {
		return models.ImageResponse{}, models.ErrImageOperationUnsupported
	}
	if len(req.Image.Data) == 0 {
		return models.ImageResponse{}, errors.New("vertex: variation image input required")
	}
	base64Image, err := encodeVertexImage(req.Image)
	if err != nil {
		return models.ImageResponse{}, err
	}
	refs := []vertexReferenceImage{{
		ReferenceType:  "REFERENCE_TYPE_RAW",
		ReferenceID:    1,
		ReferenceImage: vertexReferenceImageData{BytesBase64Encoded: base64Image},
	}}
	params := vertexImageParameters{
		SampleCount: clampVertexSampleCount(req.N),
		MimeType:    "image/png",
	}
	if params.SampleCount == 0 {
		params.SampleCount = 1
	}
	payload := vertexImagePredictRequest{
		Instances: []vertexImageInstance{{
			Prompt:          vertexVariationPrompt(a.metadata),
			ReferenceImages: refs,
		}},
		Parameters: params,
	}
	var predict vertexImagePredictResponse
	if err := a.postJSON(ctx, a.imageURL, payload, &predict); err != nil {
		return models.ImageResponse{}, err
	}
	return convertVertexImageResponse(predict)
}

func (a *Adapter) Models(ctx context.Context) ([]models.Model, error) {
	return nil, errors.New("vertex model listing not implemented")
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL, nil)
	if err != nil {
		return err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("vertex health check status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (a *Adapter) postJSON(ctx context.Context, url string, payload any, out any) error {
	resp, err := a.post(ctx, url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("vertex decode response: %w", err)
	}
	return nil
}

func (a *Adapter) post(ctx context.Context, url string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("vertex encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, decodeAPIError(resp)
	}
	return resp, nil
}

func convertChatResponse(v vertexGenerateResponse, model string) (models.ChatResponse, error) {
	candidate := v.FirstCandidate()
	if candidate == nil {
		return models.ChatResponse{}, errors.New("vertex response missing candidates")
	}
	message := models.ChatMessage{Role: "assistant", Content: candidate.Content.Text()}
	resp := models.ChatResponse{
		ID:      uuid.NewString(),
		Created: time.Now().UTC(),
		Model:   model,
		Choices: []models.ChatChoice{{
			Index:        0,
			Message:      message,
			FinishReason: strings.ToLower(candidate.FinishReason),
		}},
	}
	if usage := v.Usage(); usage != nil {
		resp.Usage = convertUsageMetadata(*usage)
	}
	return resp, nil
}

func convertStreamChunk(v vertexGenerateResponse, model string) []models.ChatChunk {
	chunks := make([]models.ChatChunk, 0, 2)
	if candidate := v.FirstCandidate(); candidate != nil {
		text := candidate.Content.Text()
		if text != "" {
			chunks = append(chunks, models.ChatChunk{
				ID:      uuid.NewString(),
				Model:   model,
				Created: time.Now().UTC(),
				Choices: []models.ChunkDelta{{
					Index:        0,
					Delta:        models.ChatMessage{Role: "assistant", Content: text},
					FinishReason: strings.ToLower(candidate.FinishReason),
				}},
			})
		}
	}
	if usage := v.Usage(); usage != nil && (usage.PromptTokens > 0 || usage.CandidatesTokens > 0 || usage.TotalTokens > 0) {
		usageCopy := convertUsageMetadata(*usage)
		chunks = append(chunks, models.ChatChunk{
			ID:      uuid.NewString(),
			Model:   model,
			Created: time.Now().UTC(),
			Usage:   &usageCopy,
		})
	}
	return chunks
}

func convertEmbeddingsResponse(v vertexPredictResponse, model string) (models.EmbeddingsResponse, error) {
	if len(v.Predictions) == 0 {
		return models.EmbeddingsResponse{}, errors.New("vertex embeddings response empty")
	}
	data := make([]models.Embedding, 0, len(v.Predictions))
	for idx, pred := range v.Predictions {
		vector := make([]float32, len(pred.Values))
		for i, val := range pred.Values {
			vector[i] = float32(val)
		}
		data = append(data, models.Embedding{Index: idx, Vector: vector})
	}
	resp := models.EmbeddingsResponse{Model: model, Embeddings: data}
	if v.Metadata != nil {
		resp.Usage = convertUsageMetadata(*v.Metadata)
	}
	return resp, nil
}

func convertUsageMetadata(meta vertexUsageMetadata) models.Usage {
	return models.Usage{
		PromptTokens:     meta.PromptTokens,
		CompletionTokens: meta.CandidatesTokens,
		TotalTokens:      meta.TotalTokens,
	}
}

type vertexImagePredictRequest struct {
	Instances  []vertexImageInstance `json:"instances"`
	Parameters vertexImageParameters `json:"parameters"`
}

type vertexImageInstance struct {
	Prompt          string                 `json:"prompt,omitempty"`
	ReferenceImages []vertexReferenceImage `json:"referenceImages,omitempty"`
}

type vertexImageParameters struct {
	SampleCount      int               `json:"sampleCount,omitempty"`
	AspectRatio      string            `json:"aspectRatio,omitempty"`
	MimeType         string            `json:"mimeType,omitempty"`
	EditMode         string            `json:"editMode,omitempty"`
	EditConfig       *vertexEditConfig `json:"editConfig,omitempty"`
	GuidanceScale    float64           `json:"guidanceScale,omitempty"`
	PersonGeneration string            `json:"personGeneration,omitempty"`
}

type vertexReferenceImage struct {
	ReferenceType   string                   `json:"referenceType"`
	ReferenceID     int                      `json:"referenceId,omitempty"`
	ReferenceImage  vertexReferenceImageData `json:"referenceImage"`
	MaskImageConfig *vertexMaskImageConfig   `json:"maskImageConfig,omitempty"`
}

type vertexReferenceImageData struct {
	BytesBase64Encoded string `json:"bytesBase64Encoded"`
}

type vertexMaskImageConfig struct {
	MaskMode string  `json:"maskMode"`
	Dilation float64 `json:"dilation,omitempty"`
}

type vertexEditConfig struct {
	BaseSteps int `json:"baseSteps,omitempty"`
}

type vertexImagePredictResponse struct {
	Predictions []vertexImagePrediction `json:"predictions"`
}

type vertexImagePrediction struct {
	BytesBase64 string `json:"bytesBase64Encoded"`
	MimeType    string `json:"mimeType"`
}

func convertVertexImageResponse(resp vertexImagePredictResponse) (models.ImageResponse, error) {
	if len(resp.Predictions) == 0 {
		return models.ImageResponse{}, errors.New("vertex image response missing predictions")
	}
	data := make([]models.ImageData, 0, len(resp.Predictions))
	for _, pred := range resp.Predictions {
		encoded := strings.TrimSpace(pred.BytesBase64)
		if encoded == "" {
			continue
		}
		data = append(data, models.ImageData{B64JSON: encoded})
	}
	if len(data) == 0 {
		return models.ImageResponse{}, errors.New("vertex image response empty")
	}
	return models.ImageResponse{
		Created: time.Now().UTC(),
		Data:    data,
	}, nil
}

func peekNonWhitespace(r *bufio.Reader) (byte, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b > ' ' {
			r.UnreadByte()
			return b, nil
		}
	}
}

func encodeVertexImage(input models.ImageInput) (string, error) {
	if len(input.Data) == 0 {
		return "", errors.New("vertex: empty image payload")
	}
	return base64.StdEncoding.EncodeToString(input.Data), nil
}

func parseVertexFloat(meta map[string]string, key string, def float64) float64 {
	if meta == nil {
		return def
	}
	if raw := strings.TrimSpace(meta[key]); raw != "" {
		if val, err := strconv.ParseFloat(raw, 64); err == nil {
			return val
		}
	}
	return def
}

func parseVertexInt(meta map[string]string, key string, def int) int {
	if meta == nil {
		return def
	}
	if raw := strings.TrimSpace(meta[key]); raw != "" {
		if val, err := strconv.Atoi(raw); err == nil {
			return val
		}
	}
	return def
}

func vertexMaskMode(meta map[string]string) string {
	mode := strings.TrimSpace(meta["vertex_mask_mode"])
	if mode == "" {
		return "MASK_MODE_USER_PROVIDED"
	}
	return mode
}

func vertexMaskDilation(meta map[string]string) float64 {
	return parseVertexFloat(meta, "vertex_mask_dilation", 0.01)
}

func vertexVariationPrompt(meta map[string]string) string {
	return strings.TrimSpace(meta["vertex_variation_prompt"])
}

func vertexEditConfigFromMetadata(meta map[string]string) *vertexEditConfig {
	steps := parseVertexInt(meta, "vertex_base_steps", 0)
	if steps <= 0 {
		return nil
	}
	return &vertexEditConfig{BaseSteps: steps}
}

func clampVertexSampleCount(n int) int {
	if n <= 0 {
		return 1
	}
	if n > 4 {
		return 4
	}
	return n
}

func vertexAspectRatio(size string) string {
	size = strings.TrimSpace(size)
	switch size {
	case "1536x1024":
		return "3:2"
	case "1024x1536":
		return "2:3"
	case "1152x768":
		return "3:2"
	case "768x1152":
		return "2:3"
	default:
		return "1:1"
	}
}
