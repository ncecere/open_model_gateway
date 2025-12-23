package vllm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers/streamutil"
)

const defaultHTTPTimeout = 60 * time.Second

// Options configures the TGI adapter.
type Options struct {
	BaseURL    string
	APIKey     string
	AuthHeader string
	HTTPClient *http.Client
}

// TGIAdapter issues requests against Hugging Face Text Generation Inference.
type TGIAdapter struct {
	client     *http.Client
	baseURL    string
	authHeader string
	apiKey     string
}

// NewTGI creates a TGI adapter for /generate and /generate_stream endpoints.
func NewTGI(opts Options) (*TGIAdapter, error) {
	baseURL := strings.TrimSpace(opts.BaseURL)
	if baseURL == "" {
		return nil, errors.New("vllm: base_url required for tgi mode")
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	authHeader := strings.TrimSpace(opts.AuthHeader)
	apiKey := strings.TrimSpace(opts.APIKey)
	if authHeader == "" && apiKey != "" {
		authHeader = "Authorization"
	}
	if strings.EqualFold(authHeader, "authorization") && apiKey != "" {
		if !strings.HasPrefix(strings.ToLower(apiKey), "bearer ") {
			apiKey = "Bearer " + apiKey
		}
	}
	return &TGIAdapter{
		client:     opts.HTTPClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		authHeader: authHeader,
		apiKey:     apiKey,
	}, nil
}

type tgiRequest struct {
	Inputs     string        `json:"inputs"`
	Parameters tgiParameters `json:"parameters,omitempty"`
}

type tgiParameters struct {
	MaxNewTokens   *int32   `json:"max_new_tokens,omitempty"`
	Temperature    *float32 `json:"temperature,omitempty"`
	TopP           *float32 `json:"top_p,omitempty"`
	Stop           []string `json:"stop,omitempty"`
	ReturnFullText bool     `json:"return_full_text"`
	Details        bool     `json:"details,omitempty"`
	Seed           *int64   `json:"seed,omitempty"`
	BestOf         *int32   `json:"best_of,omitempty"`
	TopK           *int32   `json:"top_k,omitempty"`
	TypicalP       *float32 `json:"typical_p,omitempty"`
	DoSample       *bool    `json:"do_sample,omitempty"`
	Repetition     *float32 `json:"repetition_penalty,omitempty"`
	Watermark      *bool    `json:"watermark,omitempty"`
}

type tgiResponse struct {
	GeneratedText string      `json:"generated_text"`
	Details       *tgiDetails `json:"details,omitempty"`
}

type tgiStreamEvent struct {
	Token         *tgiToken   `json:"token,omitempty"`
	GeneratedText string      `json:"generated_text,omitempty"`
	Details       *tgiDetails `json:"details,omitempty"`
}

type tgiDetails struct {
	FinishReason    string     `json:"finish_reason"`
	GeneratedTokens int32      `json:"generated_tokens"`
	Prefill         []tgiToken `json:"prefill,omitempty"`
}

type tgiToken struct {
	ID      int32  `json:"id"`
	Text    string `json:"text"`
	Special bool   `json:"special"`
}

// Chat executes a non-streaming TGI generation.
func (a *TGIAdapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	prompt, err := buildPrompt(req.Messages)
	if err != nil {
		return models.ChatResponse{}, err
	}

	payload := tgiRequest{
		Inputs:     prompt,
		Parameters: buildParameters(req),
	}
	var resp tgiResponse
	if err := a.doJSON(ctx, http.MethodPost, "/generate", payload, &resp); err != nil {
		return models.ChatResponse{}, err
	}

	message := models.ChatMessage{
		Role:    "assistant",
		Content: strings.TrimSpace(resp.GeneratedText),
	}
	usage := usageFromDetails(resp.Details)

	return models.ChatResponse{
		ID:      fmt.Sprintf("tgi-%d", time.Now().UnixNano()),
		Created: time.Now().UTC(),
		Model:   req.Model,
		Choices: []models.ChatChoice{
			{Index: 0, Message: message, FinishReason: finishReasonFromDetails(resp.Details)},
		},
		Usage: usage,
	}, nil
}

// ChatStream streams generation from the TGI /generate_stream endpoint.
func (a *TGIAdapter) ChatStream(ctx context.Context, req models.ChatRequest) (<-chan models.ChatChunk, func() error, error) {
	prompt, err := buildPrompt(req.Messages)
	if err != nil {
		return nil, nil, err
	}
	payload := tgiRequest{
		Inputs:     prompt,
		Parameters: buildParameters(req),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	resp, err := a.send(ctx, http.MethodPost, "/generate_stream", bytes.NewReader(body), "text/event-stream", true)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		return nil, nil, decodeTGIError(resp)
	}

	created := time.Now().UTC()
	id := fmt.Sprintf("tgi-%d", time.Now().UnixNano())
	forward := func(ctx context.Context, yield streamutil.YieldFunc) {
		reader := bufio.NewReader(resp.Body)
		sentRole := false

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) {
					_ = yield(models.ChatChunk{
						ID:      id,
						Model:   req.Model,
						Created: created,
						Choices: []models.ChunkDelta{{Index: 0, FinishReason: "error"}},
					})
				}
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") {
				continue
			}
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "" || payload == "[DONE]" {
				return
			}
			var event tgiStreamEvent
			if err := json.Unmarshal([]byte(payload), &event); err != nil {
				continue
			}
			if event.Token != nil && event.Token.Text != "" && !event.Token.Special {
				delta := models.ChatMessage{Content: event.Token.Text}
				if !sentRole {
					delta.Role = "assistant"
					sentRole = true
				}
				if !yield(models.ChatChunk{
					ID:      id,
					Model:   req.Model,
					Created: created,
					Choices: []models.ChunkDelta{{Index: 0, Delta: delta}},
				}) {
					return
				}
			}
			if event.Details != nil || event.GeneratedText != "" {
				usage := usagePointerFromDetails(event.Details)
				finishReason := finishReasonFromDetails(event.Details)
				if finishReason == "" && event.GeneratedText != "" {
					finishReason = "stop"
				}
				chunk := models.ChatChunk{
					ID:      id,
					Model:   req.Model,
					Created: created,
					Choices: []models.ChunkDelta{{
						Index:        0,
						FinishReason: finishReason,
					}},
					Usage: usage,
				}
				_ = yield(chunk)
				return
			}
		}
	}

	chunks, cancel := streamutil.Forward(ctx, resp.Body.Close, forward)
	return chunks, cancel, nil
}

// HealthCheck probes /health for readiness.
func (a *TGIAdapter) HealthCheck(ctx context.Context) error {
	resp, err := a.send(ctx, http.MethodGet, "/health", nil, "application/json", false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return decodeTGIError(resp)
	}
	return nil
}

func buildPrompt(messages []models.ChatMessage) (string, error) {
	if len(messages) == 0 {
		return "", errors.New("vllm: messages required")
	}
	var sb strings.Builder
	for _, msg := range messages {
		if msg.HasNonTextContent() {
			return "", errors.New("vllm: non-text content unsupported by tgi mode")
		}
		content := strings.TrimSpace(msg.Text())
		if content == "" {
			continue
		}
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			role = "user"
		}
		sb.WriteString(role)
		sb.WriteString(": ")
		sb.WriteString(content)
		sb.WriteString("\n")
	}
	prompt := strings.TrimSpace(sb.String())
	if prompt == "" {
		return "", errors.New("vllm: prompt required")
	}
	if !strings.HasSuffix(strings.ToLower(prompt), "\nassistant:") {
		prompt += "\nassistant:"
	}
	return prompt, nil
}

func buildParameters(req models.ChatRequest) tgiParameters {
	params := tgiParameters{
		ReturnFullText: false,
		Details:        true,
	}
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		params.MaxNewTokens = int32Ptr(*req.MaxTokens)
	}
	if req.Temperature != nil {
		params.Temperature = req.Temperature
	}
	if req.TopP != nil {
		params.TopP = req.TopP
	}
	if len(req.Stop) > 0 {
		params.Stop = append(params.Stop, req.Stop...)
	}
	return params
}

func usageFromDetails(details *tgiDetails) models.Usage {
	if details == nil {
		return models.Usage{}
	}
	var promptTokens int32
	if len(details.Prefill) > 0 {
		promptTokens = int32(len(details.Prefill))
	}
	usage := models.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: details.GeneratedTokens,
		TotalTokens:      promptTokens + details.GeneratedTokens,
	}
	return usage
}

func usagePointerFromDetails(details *tgiDetails) *models.Usage {
	usage := usageFromDetails(details)
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 && usage.TotalTokens == 0 {
		return nil
	}
	return &usage
}

func finishReasonFromDetails(details *tgiDetails) string {
	if details == nil {
		return ""
	}
	return strings.TrimSpace(details.FinishReason)
}

func (a *TGIAdapter) doJSON(ctx context.Context, method, path string, payload interface{}, dest interface{}) error {
	var reader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(body)
	}
	resp, err := a.send(ctx, method, path, reader, "application/json", payload != nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return decodeTGIError(resp)
	}
	if dest == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("vllm: decode response: %w", err)
	}
	return nil
}

func (a *TGIAdapter) send(ctx context.Context, method, path string, body io.Reader, accept string, hasBody bool) (*http.Response, error) {
	req, err := a.newRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	return a.client.Do(req)
}

func (a *TGIAdapter) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := a.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if a.authHeader != "" && a.apiKey != "" {
		req.Header.Set(a.authHeader, a.apiKey)
	}
	return req, nil
}

func decodeTGIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	return fmt.Errorf("vllm: %s", msg)
}

func int32Ptr(value int32) *int32 {
	if value == 0 {
		return nil
	}
	return &value
}
