package groq

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
	"github.com/ncecere/open_model_gateway/backend/internal/apperror"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers/streamutil"
)

const (
	defaultBaseURL     = "https://api.groq.com/openai/v1"
	defaultHTTPTimeout = 60 * time.Second
)

// Options configure the Groq adapter.
type Options struct {
	APIKey     string
	BaseURL    string
	Region     string
	HTTPClient *http.Client
}

// Adapter issues requests to Groq's OpenAI-compatible API surface.
type Adapter struct {
	client  *http.Client
	baseURL string
	opts    Options
}

// New creates a Groq adapter using the provided credentials.
func New(opts Options) (*Adapter, error) {
	if strings.TrimSpace(opts.APIKey) == "" {
		return nil, apperror.Validation("groq.New", "api key required")
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
			Region:  strings.TrimSpace(opts.Region),
		},
	}, nil
}

// Chat executes a non-streaming chat completion.
func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return models.ChatResponse{}, apperror.Validation("groq.Chat", "messages are required")
	}
	payload := buildChatRequest(req, false)
	var resp chatCompletionResponse
	if err := a.doJSON(ctx, http.MethodPost, "/chat/completions", payload, &resp); err != nil {
		return models.ChatResponse{}, err
	}
	return convertChatResponse(resp), nil
}

// ChatStream performs a streaming chat completion using SSE.
func (a *Adapter) ChatStream(ctx context.Context, req models.ChatRequest) (<-chan models.ChatChunk, func() error, error) {
	if len(req.Messages) == 0 {
		return nil, nil, apperror.Validation("groq.ChatStream", "messages are required")
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
		var buffer strings.Builder

		flush := func() bool {
			defer buffer.Reset()
			payload := strings.TrimSpace(buffer.String())
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
			switch {
			case line == "":
				if !flush() {
					return
				}
			case strings.HasPrefix(line, ":"):
				continue
			case strings.HasPrefix(line, "event:"):
				continue
			case strings.HasPrefix(line, "data:"):
				if buffer.Len() > 0 {
					buffer.WriteByte('\n')
				}
				buffer.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
	}

	chunks, cancel := streamutil.Forward(ctx, resp.Body.Close, forward)
	return chunks, cancel, nil
}

// HealthCheck verifies credentials by calling /models.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	resp, err := a.send(ctx, http.MethodGet, "/models", nil, "application/json", false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return decodeAPIError(resp)
	}
	return nil
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
		return fmt.Errorf("groq: decode response: %w", err)
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
	if a.opts.Region != "" {
		req.Header.Set("X-Groq-Region", a.opts.Region)
	}
	return req, nil
}

func buildChatRequest(req models.ChatRequest, streaming bool) chatCompletionRequest {
	messages := make([]chatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		cm := chatMessage{
			Role:       msg.Role,
			Content:    models.MarshalMessageContent(msg),
			ToolCallID: msg.ToolCallID,
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
		payload.MaxCompletionTokens = req.MaxTokens
		payload.MaxTokens = req.MaxTokens
	}
	if len(req.Stop) > 0 {
		payload.Stop = req.Stop
	}
	if streaming {
		payload.StreamOptions = &streamOptions{IncludeUsage: true}
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
		content, parts := openaihelper.ExtractMessageContent(string(choice.Message.Content), "")
		msg := models.ChatMessage{
			Role:         choice.Message.Role,
			Content:      content,
			ContentParts: parts,
			ToolCallID:   choice.Message.ToolCallID,
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
		content, parts := openaihelper.ExtractMessageContent(string(choice.Delta.Content), "")
		msg := models.ChatMessage{
			Role:         choice.Delta.Role,
			Content:      content,
			ContentParts: parts,
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

func convertUsage(usage usagePayload) models.Usage {
	return models.Usage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func usagePointer(usage *usagePayload) *models.Usage {
	if usage == nil {
		return nil
	}
	result := convertUsage(*usage)
	return &result
}

func epochTime(sec int64) time.Time {
	if sec <= 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}

func decodeAPIError(resp *http.Response) error {
	var parsed apiError
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("groq: http %d", resp.StatusCode)
	}
	if parsed.Error.Message == "" {
		return fmt.Errorf("groq: http %d", resp.StatusCode)
	}
	return fmt.Errorf("groq: %s", parsed.Error.Message)
}

type chatCompletionRequest struct {
	Model               string          `json:"model"`
	Messages            []chatMessage   `json:"messages"`
	Temperature         *float32        `json:"temperature,omitempty"`
	TopP                *float32        `json:"top_p,omitempty"`
	MaxTokens           *int32          `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int32          `json:"max_completion_tokens,omitempty"`
	Stop                []string        `json:"stop,omitempty"`
	Stream              bool            `json:"stream,omitempty"`
	StreamOptions       *streamOptions  `json:"stream_options,omitempty"`
	Tools               []chatTool      `json:"tools,omitempty"`
	ToolChoice          json.RawMessage `json:"tool_choice,omitempty"`
	ParallelToolCalls   *bool           `json:"parallel_tool_calls,omitempty"`
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
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCalls  []chatToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatCompletionResponse struct {
	ID       string          `json:"id"`
	Object   string          `json:"object"`
	Created  int64           `json:"created"`
	Model    string          `json:"model"`
	Choices  []chatChoice    `json:"choices"`
	Usage    usagePayload    `json:"usage"`
	Metadata json.RawMessage `json:"x_groq"`
}

type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type usagePayload struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

type chatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Model   string        `json:"model"`
	Created int64         `json:"created"`
	Choices []chunkChoice `json:"choices"`
	Usage   *usagePayload `json:"usage"`
}

type chunkChoice struct {
	Index        int         `json:"index"`
	FinishReason string      `json:"finish_reason"`
	Delta        chatMessage `json:"delta"`
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
		Param   string `json:"param"`
	} `json:"error"`
}
