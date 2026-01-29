package openrouter

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

	"github.com/ncecere/open_model_gateway/backend/internal/adapters/openaihelper"
	"github.com/ncecere/open_model_gateway/backend/internal/adapters/retryafter"
	"github.com/ncecere/open_model_gateway/backend/internal/apperror"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers/streamutil"
)

const (
	defaultBaseURL     = "https://openrouter.ai/api/v1"
	defaultHTTPTimeout = 60 * time.Second
)

// Options configure the OpenRouter adapter.
type Options struct {
	APIKey     string
	BaseURL    string
	Referer    string
	AppName    string
	HTTPClient *http.Client
}

// Adapter issues requests against the OpenRouter API surface.
type Adapter struct {
	client  *http.Client
	baseURL string
	opts    Options
}

// New builds an OpenRouter adapter using the supplied credentials.
func New(opts Options) (*Adapter, error) {
	if strings.TrimSpace(opts.APIKey) == "" {
		return nil, apperror.Validation("openrouter.New", "api key required")
	}
	if strings.TrimSpace(opts.BaseURL) == "" {
		opts.BaseURL = defaultBaseURL
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &Adapter{
		client:  opts.HTTPClient,
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		opts: Options{
			APIKey:  strings.TrimSpace(opts.APIKey),
			BaseURL: strings.TrimRight(opts.BaseURL, "/"),
			Referer: strings.TrimSpace(opts.Referer),
			AppName: strings.TrimSpace(opts.AppName),
		},
	}, nil
}

// Chat executes a non-streaming chat completion.
func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return models.ChatResponse{}, apperror.Validation("openrouter.Chat", "messages are required")
	}
	payload := buildChatRequest(req, false)
	var resp chatCompletionResponse
	if err := a.doJSON(ctx, http.MethodPost, "/chat/completions", payload, &resp); err != nil {
		return models.ChatResponse{}, err
	}
	return convertChatResponse(resp), nil
}

// ChatStream streams chat completions using SSE.
func (a *Adapter) ChatStream(ctx context.Context, req models.ChatRequest) (<-chan models.ChatChunk, func() error, error) {
	if len(req.Messages) == 0 {
		return nil, nil, apperror.Validation("openrouter.ChatStream", "messages are required")
	}
	payload := buildChatRequest(req, true)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	resp, err := a.send(ctx, http.MethodPost, "/chat/completions", bytes.NewReader(body), "text/event-stream", true)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		return nil, nil, decodeAPIError(resp)
	}

	forward := func(ctx context.Context, yield streamutil.YieldFunc) {
		reader := bufio.NewReader(resp.Body)
		var data strings.Builder

		flush := func() bool {
			defer data.Reset()
			payload := strings.TrimSpace(data.String())
			if payload == "" {
				return true
			}
			if payload == "[DONE]" {
				return false
			}
			var chunk chatCompletionChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				return true
			}
			if !yield(convertChatChunk(chunk)) {
				return false
			}
			return true
		}

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) {
					return
				}
				if !flush() {
					return
				}
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, ":") {
				continue
			}
			if strings.HasPrefix(line, "event:") {
				continue
			}
			if strings.HasPrefix(line, "data:") {
				if data.Len() > 0 {
					data.WriteByte('\n')
				}
				data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
				continue
			}
			if line == "" {
				if !flush() {
					return
				}
				continue
			}
		}
	}

	chunks, cancel := streamutil.Forward(ctx, resp.Body.Close, forward)
	return chunks, cancel, nil
}

// Embed invokes the embeddings API.
func (a *Adapter) Embed(ctx context.Context, req models.EmbeddingsRequest) (models.EmbeddingsResponse, error) {
	if len(req.Input) == 0 {
		return models.EmbeddingsResponse{}, apperror.Validation("openrouter.Embed", "embeddings input required")
	}
	payload := embeddingRequest{
		Model: req.Model,
		Input: req.Input,
	}
	var resp embeddingResponse
	if err := a.doJSON(ctx, http.MethodPost, "/embeddings", payload, &resp); err != nil {
		return models.EmbeddingsResponse{}, err
	}
	return convertEmbeddingsResponse(resp), nil
}

// Models fetches the OpenRouter catalog and maps it into generic metadata.
func (a *Adapter) Models(ctx context.Context) ([]models.Model, error) {
	resp, err := a.fetchModelList(ctx)
	if err != nil {
		return nil, err
	}
	return convertModelList(resp), nil
}

// HealthCheck verifies the configured key by calling the /key endpoint.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	resp, err := a.send(ctx, http.MethodGet, "/key", nil, "application/json", false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return decodeAPIError(resp)
	}
	return nil
}

func (a *Adapter) fetchModelList(ctx context.Context) (modelListResponse, error) {
	var resp modelListResponse
	err := a.doJSON(ctx, http.MethodGet, "/models", nil, &resp)
	return resp, err
}

func (a *Adapter) doJSON(ctx context.Context, method, path string, payload interface{}, dest interface{}) error {
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
		return decodeAPIError(resp)
	}
	if dest == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("openrouter: decode response: %w", err)
	}
	return nil
}

func (a *Adapter) send(ctx context.Context, method, path string, body io.Reader, accept string, hasBody bool) (*http.Response, error) {
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

func (a *Adapter) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	endpoint := a.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.opts.APIKey)
	req.Header.Set("Accept", "application/json")
	if a.opts.Referer != "" {
		req.Header.Set("HTTP-Referer", a.opts.Referer)
	}
	if a.opts.AppName != "" {
		req.Header.Set("X-Title", a.opts.AppName)
	}
	return req, nil
}

func buildChatRequest(req models.ChatRequest, streaming bool) chatCompletionRequest {
	messages := make([]chatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		cm := chatMessage{
			Role:             msg.Role,
			Content:          models.MarshalMessageContent(msg),
			Name:             msg.Name,
			Reasoning:        msg.Reasoning,
			ReasoningContent: msg.ReasoningContent,
			ToolCallID:       msg.ToolCallID,
		}
		// Convert tool_calls for assistant messages
		if len(msg.ToolCalls) > 0 {
			cm.ToolCalls = make([]chatToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				cm.ToolCalls = append(cm.ToolCalls, chatToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: chatToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
		messages = append(messages, cm)
	}
	payload := chatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   streaming,
	}
	if req.Temperature != nil {
		payload.Temperature = req.Temperature
	}
	if req.TopP != nil {
		payload.TopP = req.TopP
	}
	if req.MaxTokens != nil {
		payload.MaxTokens = req.MaxTokens
	}
	if len(req.Stop) > 0 {
		payload.Stop = req.Stop
	}
	if streaming {
		payload.StreamOptions = &streamOptions{IncludeUsage: true}
	}

	// Best-effort pass-through for penalties and response_format
	if req.PresencePenalty != nil {
		payload.PresencePenalty = req.PresencePenalty
	}
	if req.FrequencyPenalty != nil {
		payload.FrequencyPenalty = req.FrequencyPenalty
	}
	if len(req.ChatResponseFormat) > 0 {
		payload.ResponseFormat = req.ChatResponseFormat
	}

	// Add tools
	if len(req.Tools) > 0 {
		payload.Tools = make([]chatTool, 0, len(req.Tools))
		for _, tool := range req.Tools {
			payload.Tools = append(payload.Tools, chatTool{
				Type: tool.Type,
				Function: chatToolFunction{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
					Strict:      tool.Function.Strict,
				},
			})
		}
	}

	// Add tool_choice
	if len(req.ToolChoice) > 0 {
		payload.ToolChoice = req.ToolChoice
	}

	// Add parallel_tool_calls
	if req.ParallelToolCalls != nil {
		payload.ParallelToolCalls = req.ParallelToolCalls
	}

	return payload
}

func convertChatResponse(resp chatCompletionResponse) models.ChatResponse {
	choices := make([]models.ChatChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		content, parts := openaihelper.ExtractMessageContent(string(choice.Message.Content), choice.Message.ReasoningContent)
		if strings.TrimSpace(content) == "" && strings.TrimSpace(choice.Message.ReasoningContent) != "" {
			content = choice.Message.ReasoningContent
		}
		msg := models.ChatMessage{
			Role:             choice.Message.Role,
			Content:          content,
			ContentParts:     parts,
			Reasoning:        choice.Message.Reasoning,
			ReasoningContent: choice.Message.ReasoningContent,
			ToolCallID:       choice.Message.ToolCallID,
		}
		// Extract tool_calls
		if len(choice.Message.ToolCalls) > 0 {
			msg.ToolCalls = make([]models.ToolCall, 0, len(choice.Message.ToolCalls))
			for _, tc := range choice.Message.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, models.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: models.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
		choices = append(choices, models.ChatChoice{
			Index:        choice.Index,
			Message:      msg,
			FinishReason: choice.FinishReason,
		})
	}
	return models.ChatResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Created: epochTime(resp.Created),
		Choices: choices,
		Usage:   convertUsage(resp.Usage),
	}
}

func convertChatChunk(chunk chatCompletionChunk) models.ChatChunk {
	choices := make([]models.ChunkDelta, 0, len(chunk.Choices))
	for _, choice := range chunk.Choices {
		content, parts := openaihelper.ExtractMessageContent(string(choice.Delta.Content), choice.Delta.ReasoningContent)
		if strings.TrimSpace(content) == "" && strings.TrimSpace(choice.Delta.ReasoningContent) != "" {
			content = choice.Delta.ReasoningContent
		}
		msg := models.ChatMessage{
			Role:             choice.Delta.Role,
			Content:          content,
			ContentParts:     parts,
			Reasoning:        choice.Delta.Reasoning,
			ReasoningContent: choice.Delta.ReasoningContent,
		}
		// Extract streaming tool_calls
		if len(choice.Delta.ToolCalls) > 0 {
			msg.ToolCalls = make([]models.ToolCall, 0, len(choice.Delta.ToolCalls))
			for _, tc := range choice.Delta.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, models.ToolCall{
					ID:    tc.ID,
					Type:  tc.Type,
					Index: tc.Index,
					Function: models.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
		choices = append(choices, models.ChunkDelta{
			Index:        choice.Index,
			Delta:        msg,
			FinishReason: choice.FinishReason,
		})
	}
	return models.ChatChunk{
		ID:      chunk.ID,
		Model:   chunk.Model,
		Created: epochTime(chunk.Created),
		Choices: choices,
		Usage:   usagePointer(chunk.Usage),
	}
}

func convertEmbeddingsResponse(resp embeddingResponse) models.EmbeddingsResponse {
	vectors := make([]models.Embedding, 0, len(resp.Data))
	for _, item := range resp.Data {
		vec := make([]float32, 0, len(item.Embedding))
		for _, val := range item.Embedding {
			vec = append(vec, float32(val))
		}
		vectors = append(vectors, models.Embedding{
			Index:  item.Index,
			Vector: vec,
		})
	}
	return models.EmbeddingsResponse{
		Model:      resp.Model,
		Embeddings: vectors,
		Usage:      convertUsage(resp.Usage),
	}
}

func convertModelList(resp modelListResponse) []models.Model {
	if len(resp.Data) == 0 {
		return nil
	}
	out := make([]models.Model, 0, len(resp.Data))
	for _, item := range resp.Data {
		m := models.Model{
			Alias:           item.ID,
			Provider:        "openrouter",
			ProviderModel:   item.ID,
			ContextWindow:   item.ContextLength,
			MaxOutputTokens: item.TopProvider.MaxCompletionTokens,
			SupportsTools:   hasTools(item.SupportedParameters),
		}
		modalities := item.Architecture.InputModalities
		if len(modalities) == 0 && item.Architecture.Modality != "" {
			modalities = []string{item.Architecture.Modality}
		}
		if len(modalities) > 0 {
			m.Modalities = copyStrings(modalities)
		}
		out = append(out, m)
	}
	return out
}

func copyStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func hasTools(params []string) bool {
	for _, p := range params {
		if strings.EqualFold(p, "tools") {
			return true
		}
	}
	return false
}

func convertUsage(u usagePayload) models.Usage {
	usage := models.Usage{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		ReasoningTokens:  u.ReasoningTokens,
		TotalTokens:      u.TotalTokens,
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return usage
}

func usagePointer(u usagePayload) *models.Usage {
	if u.PromptTokens == 0 && u.CompletionTokens == 0 && u.TotalTokens == 0 {
		return nil
	}
	usage := convertUsage(u)
	return &usage
}

func epochTime(ts int64) time.Time {
	if ts <= 0 {
		return time.Now().UTC()
	}
	return time.Unix(ts, 0).UTC()
}

func decodeAPIError(resp *http.Response) error {
	op := "openrouter"
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		resp.Body.Close()
		return retryafter.RateLimitError(op, resp)
	case http.StatusServiceUnavailable:
		resp.Body.Close()
		return retryafter.OverloadedError(op, resp)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if len(body) == 0 {
		return fmt.Errorf("openrouter: http %d", resp.StatusCode)
	}
	var apiErr apiErrorResponse
	if err := json.Unmarshal(body, &apiErr); err != nil || apiErr.Error.Message == "" {
		return fmt.Errorf("openrouter: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if resp.StatusCode >= 500 {
		return apperror.ServiceUnavailable(op, apiErr.Error.Message)
	}
	return fmt.Errorf("openrouter: %s", apiErr.Error.Message)
}

type chatCompletionRequest struct {
	Model             string          `json:"model"`
	Messages          []chatMessage   `json:"messages"`
	Temperature       *float32        `json:"temperature,omitempty"`
	TopP              *float32        `json:"top_p,omitempty"`
	MaxTokens         *int32          `json:"max_tokens,omitempty"`
	Stop              []string        `json:"stop,omitempty"`
	Stream            bool            `json:"stream,omitempty"`
	StreamOptions     *streamOptions  `json:"stream_options,omitempty"`
	PresencePenalty   *float32        `json:"presence_penalty,omitempty"`
	FrequencyPenalty  *float32        `json:"frequency_penalty,omitempty"`
	ResponseFormat    json.RawMessage `json:"response_format,omitempty"`
	Tools             []chatTool      `json:"tools,omitempty"`
	ToolChoice        json.RawMessage `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool           `json:"parallel_tool_calls,omitempty"`
}

type chatTool struct {
	Type     string           `json:"type"` // "function"
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type chatToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"` // "function"
	Function chatToolCallFunction `json:"function"`
	Index    *int                 `json:"index,omitempty"`
}

type chatToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatMessage struct {
	Role             string          `json:"role"`
	Content          json.RawMessage `json:"content"`
	Name             string          `json:"name,omitempty"`
	Reasoning        string          `json:"reasoning,omitempty"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
	ToolCalls        []chatToolCall  `json:"tool_calls,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatCompletionResponse struct {
	ID      string           `json:"id"`
	Model   string           `json:"model"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Choices []chatChoice     `json:"choices"`
	Usage   usagePayload     `json:"usage"`
	Error   *completionError `json:"error,omitempty"`
}

type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatCompletionChunk struct {
	ID      string            `json:"id"`
	Model   string            `json:"model"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Choices []chatChoiceDelta `json:"choices"`
	Usage   usagePayload      `json:"usage"`
	Error   *completionError  `json:"error,omitempty"`
}

type chatChoiceDelta struct {
	Index        int         `json:"index"`
	Delta        chatMessage `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type completionError struct {
	Code    interface{} `json:"code"`
	Message string      `json:"message"`
}

type usagePayload struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	ReasoningTokens  int32 `json:"reasoning_tokens,omitempty"`
	TotalTokens      int32 `json:"total_tokens"`
}

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingResponse struct {
	Model string            `json:"model"`
	Data  []embeddingVector `json:"data"`
	Usage usagePayload      `json:"usage"`
}

type embeddingVector struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

type modelListResponse struct {
	Data []modelMetadata `json:"data"`
}

type modelMetadata struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Description         string            `json:"description"`
	ContextLength       int32             `json:"context_length"`
	Architecture        modelArchitecture `json:"architecture"`
	SupportedParameters []string          `json:"supported_parameters"`
	TopProvider         providerMetadata  `json:"top_provider"`
	Pricing             modelPricing      `json:"pricing"`
}

type modelArchitecture struct {
	Modality         string   `json:"modality"`
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
}

type providerMetadata struct {
	MaxCompletionTokens int32 `json:"max_completion_tokens"`
	ContextLength       int32 `json:"context_length"`
}

type apiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type modelPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	Request    string `json:"request"`
	Image      string `json:"image"`
	WebSearch  string `json:"web_search"`
}
