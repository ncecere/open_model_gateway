package anthropic

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

const defaultBaseURL = "https://api.anthropic.com"
const defaultVersion = "2023-06-01"

// Options configures the native Anthropic adapter.
type Options struct {
	APIKey           string
	BaseURL          string
	Version          string
	DefaultMaxTokens int32
	HTTPClient       *http.Client
}

type Adapter struct {
	client  *http.Client
	baseURL string
	opts    Options
}

func New(opts Options) (*Adapter, error) {
	if strings.TrimSpace(opts.APIKey) == "" {
		return nil, errors.New("anthropic: api key required")
	}
	if strings.TrimSpace(opts.BaseURL) == "" {
		opts.BaseURL = defaultBaseURL
	}
	if strings.TrimSpace(opts.Version) == "" {
		opts.Version = defaultVersion
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Adapter{
		client:  opts.HTTPClient,
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		opts:    opts,
	}, nil
}

func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	payload, err := buildAnthropicMessageRequest(req, a.opts.DefaultMaxTokens, false)
	if err != nil {
		return models.ChatResponse{}, err
	}
	var anthropicResp anthropicResponse
	if err := a.postJSON(ctx, "/v1/messages", payload, &anthropicResp); err != nil {
		return models.ChatResponse{}, err
	}
	return convertAnthropicResponse(anthropicResp, req.Model), nil
}

func (a *Adapter) ChatStream(ctx context.Context, req models.ChatRequest) (<-chan models.ChatChunk, func() error, error) {
	payload, err := buildAnthropicMessageRequest(req, a.opts.DefaultMaxTokens, true)
	if err != nil {
		return nil, nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	endpoint := fmt.Sprintf("%s/v1/messages", a.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-api-key", a.opts.APIKey)
	httpReq.Header.Set("anthropic-version", a.opts.Version)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, nil, decodeAPIError(resp)
	}

	forward := func(ctx context.Context, yield streamutil.YieldFunc) {
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		created := time.Now().UTC()
		messageID := fmt.Sprintf("chatcmpl-anthropic-%d", created.UnixNano())
		model := req.Model
		finishReason := ""
		var usage anthropicUsage

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" || line == "event: ping" {
				continue
			}
			if line == "data: [DONE]" {
				chunk := models.ChatChunk{
					ID:      messageID,
					Model:   model,
					Created: created,
					Choices: []models.ChunkDelta{{
						Index:        0,
						FinishReason: mapAnthropicStopReason(finishReason),
					}},
				}
				if usage.InputTokens > 0 || usage.OutputTokens > 0 {
					chunk.Usage = &models.Usage{
						PromptTokens:     usage.InputTokens,
						CompletionTokens: usage.OutputTokens,
						TotalTokens:      usage.InputTokens + usage.OutputTokens,
					}
				}
				_ = yield(chunk)
				return
			}
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			var evt anthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &evt); err != nil {
				continue
			}
			switch evt.Type {
			case "message_start":
				if evt.Message != nil {
					if evt.Message.ID != "" {
						messageID = evt.Message.ID
					}
					if evt.Message.Model != "" {
						model = evt.Message.Model
					}
				}
			case "content_block_delta":
				text := evt.DeltaText()
				if text == "" {
					continue
				}
				chunk := models.ChatChunk{
					ID:      messageID,
					Model:   model,
					Created: created,
					Choices: []models.ChunkDelta{{
						Index: evt.Index,
						Delta: models.ChatMessage{Role: "assistant", Content: text},
					}},
				}
				if !yield(chunk) {
					return
				}
			case "message_delta":
				if reason := evt.StopReason(); reason != "" {
					finishReason = reason
				}
				if evt.Usage.InputTokens > 0 || evt.Usage.OutputTokens > 0 {
					usage = evt.Usage
				}
			case "message_stop":
				chunk := models.ChatChunk{
					ID:      messageID,
					Model:   model,
					Created: created,
					Choices: []models.ChunkDelta{{
						Index:        evt.Index,
						FinishReason: mapAnthropicStopReason(finishReason),
					}},
				}
				if usage.InputTokens > 0 || usage.OutputTokens > 0 {
					chunk.Usage = &models.Usage{
						PromptTokens:     usage.InputTokens,
						CompletionTokens: usage.OutputTokens,
						TotalTokens:      usage.InputTokens + usage.OutputTokens,
					}
				}
				_ = yield(chunk)
				return
			case "metadata":
				if evt.Usage.InputTokens > 0 || evt.Usage.OutputTokens > 0 {
					usage = evt.Usage
				}
			}
		}
	}

	cancel := func() error {
		resp.Body.Close()
		return nil
	}
	chunks, closeFn := streamutil.Forward(ctx, cancel, forward)
	return chunks, closeFn, nil
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/models", a.baseURL), nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", a.opts.APIKey)
	req.Header.Set("anthropic-version", a.opts.Version)
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("anthropic health status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (a *Adapter) Models(ctx context.Context) ([]models.Model, error) {
	return nil, errors.New("anthropic model listing not implemented")
}

func (a *Adapter) postJSON(ctx context.Context, path string, payload anthropicRequestBody, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s%s", a.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", a.opts.APIKey)
	req.Header.Set("anthropic-version", a.opts.Version)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return decodeAPIError(resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func buildAnthropicMessageRequest(req models.ChatRequest, defaultMax int32, stream bool) (anthropicRequestBody, error) {
	var systemPrompts []string
	messages := make([]anthropicMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		switch strings.ToLower(msg.Role) {
		case "system":
			systemPrompts = append(systemPrompts, msg.Content)
		case "assistant":
			messages = append(messages, anthropicMessage{
				Role:    "assistant",
				Content: []anthropicContent{{Type: "text", Text: msg.Content}},
			})
		default:
			messages = append(messages, anthropicMessage{
				Role:    "user",
				Content: []anthropicContent{{Type: "text", Text: msg.Content}},
			})
		}
	}

	maxTokens := int32(0)
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	} else if defaultMax > 0 {
		maxTokens = defaultMax
	}
	if maxTokens == 0 {
		maxTokens = 1024
	}

	body := anthropicRequestBody{
		Model:     req.Model,
		Messages:  messages,
		MaxTokens: maxTokens,
		Stream:    stream,
	}
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
	return body, nil
}

type anthropicRequestBody struct {
	Model         string             `json:"model"`
	System        string             `json:"system,omitempty"`
	Messages      []anthropicMessage `json:"messages"`
	MaxTokens     int32              `json:"max_tokens"`
	Temperature   float64            `json:"temperature,omitempty"`
	TopP          float64            `json:"top_p,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
}

// Reuse structures from bedrock adapter-style

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResponse struct {
	ID         string             `json:"id"`
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

type anthropicUsage struct {
	InputTokens  int32 `json:"input_tokens"`
	OutputTokens int32 `json:"output_tokens"`
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

func convertAnthropicResponse(resp anthropicResponse, model string) models.ChatResponse {
	message := models.ChatMessage{Role: "assistant", Content: resp.JoinText()}
	return models.ChatResponse{
		ID:      resp.ID,
		Model:   model,
		Created: time.Now().UTC(),
		Choices: []models.ChatChoice{{
			Index:        0,
			Message:      message,
			FinishReason: mapAnthropicStopReason(resp.StopReason),
		}},
		Usage: models.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

func mapAnthropicStopReason(reason string) string {
	switch reason {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	default:
		return reason
	}
}

func decodeAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("anthropic api error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
