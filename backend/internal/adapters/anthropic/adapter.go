package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/apperror"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers/streamutil"
)

const (
	defaultBaseURL              = "https://api.anthropic.com"
	defaultVersion              = "2023-06-01"
	anthropicMaxInlineDataBytes = 20 * 1024 * 1024
)

// Options configures the native Anthropic adapter.
type Options struct {
	APIKey           string
	BaseURL          string
	Version          string
	DefaultMaxTokens int32
	HTTPClient       *http.Client
	AssetHTTPClient  *http.Client
}

type Adapter struct {
	client      *http.Client
	assetClient *http.Client
	baseURL     string
	opts        Options
}

func New(opts Options) (*Adapter, error) {
	if strings.TrimSpace(opts.APIKey) == "" {
		return nil, apperror.Validation("anthropic.New", "api key required")
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
	if opts.AssetHTTPClient == nil {
		opts.AssetHTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Adapter{
		client:      opts.HTTPClient,
		assetClient: opts.AssetHTTPClient,
		baseURL:     strings.TrimRight(opts.BaseURL, "/"),
		opts:        opts,
	}, nil
}

func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	payload, err := a.buildMessageRequest(ctx, req, a.opts.DefaultMaxTokens, false)
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
	payload, err := a.buildMessageRequest(ctx, req, a.opts.DefaultMaxTokens, true)
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

		// Track current tool call being built
		type toolCallInProgress struct {
			id        string
			name      string
			arguments strings.Builder
		}
		var currentToolCalls = make(map[int]*toolCallInProgress)

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
			case "content_block_start":
				// Start of a new content block (text or tool_use)
				if evt.ContentBlock != nil && evt.ContentBlock.Type == "tool_use" {
					currentToolCalls[evt.Index] = &toolCallInProgress{
						id:   evt.ContentBlock.ID,
						name: evt.ContentBlock.Name,
					}
					// Emit the start of the tool call
					idx := evt.Index
					chunk := models.ChatChunk{
						ID:      messageID,
						Model:   model,
						Created: created,
						Choices: []models.ChunkDelta{{
							Index: 0,
							Delta: models.ChatMessage{
								Role: "assistant",
								ToolCalls: []models.ToolCall{{
									ID:    evt.ContentBlock.ID,
									Type:  "function",
									Index: &idx,
									Function: models.ToolCallFunction{
										Name:      evt.ContentBlock.Name,
										Arguments: "",
									},
								}},
							},
						}},
					}
					if !yield(chunk) {
						return
					}
				}
			case "content_block_delta":
				if evt.Delta != nil && evt.Delta.Type == "input_json_delta" {
					// Accumulate tool call arguments
					if tc, ok := currentToolCalls[evt.Index]; ok {
						tc.arguments.WriteString(evt.Delta.PartialJSON)
						// Emit delta for arguments
						idx := evt.Index
						chunk := models.ChatChunk{
							ID:      messageID,
							Model:   model,
							Created: created,
							Choices: []models.ChunkDelta{{
								Index: 0,
								Delta: models.ChatMessage{
									Role: "assistant",
									ToolCalls: []models.ToolCall{{
										Index: &idx,
										Function: models.ToolCallFunction{
											Arguments: evt.Delta.PartialJSON,
										},
									}},
								},
							}},
						}
						if !yield(chunk) {
							return
						}
					}
				} else {
					// Text delta
					text := evt.DeltaText()
					if text == "" {
						continue
					}
					parts := []models.MessageContentPart{{
						Type: models.MessageContentPartTypeText,
						Text: text,
					}}
					chunk := models.ChatChunk{
						ID:      messageID,
						Model:   model,
						Created: created,
						Choices: []models.ChunkDelta{{
							Index: evt.Index,
							Delta: models.ChatMessage{Role: "assistant", Content: text, ContentParts: parts},
						}},
					}
					if !yield(chunk) {
						return
					}
				}
			case "content_block_stop":
				// Tool call complete - we've been streaming deltas so nothing more to emit
				delete(currentToolCalls, evt.Index)
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
	return nil, apperror.BadRequest("anthropic.Models", "model listing not implemented")
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

func (a *Adapter) buildMessageRequest(ctx context.Context, req models.ChatRequest, defaultMax int32, stream bool) (anthropicRequestBody, error) {
	var systemPrompts []string
	messages := make([]anthropicMessage, 0, len(req.Messages))

	for idx, msg := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		allowNonText := role != "system" && role != "developer"
		parts, err := a.convertMessageParts(ctx, idx, msg, allowNonText)
		if err != nil {
			return anthropicRequestBody{}, err
		}

		switch role {
		case "system", "developer":
			for _, part := range parts {
				if strings.TrimSpace(part.Text) != "" {
					systemPrompts = append(systemPrompts, part.Text)
				}
			}
		case "assistant":
			// For assistant messages with tool_calls, add tool_use content blocks
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					parts = append(parts, anthropicContent{
						Type:  "tool_use",
						ID:    tc.ID,
						Name:  tc.Function.Name,
						Input: json.RawMessage(tc.Function.Arguments),
					})
				}
			}
			if len(parts) > 0 {
				messages = append(messages, anthropicMessage{
					Role:    "assistant",
					Content: parts,
				})
			}
		case "tool":
			// Tool response messages become user messages with tool_result content
			toolResultContent := json.RawMessage(`"` + strings.ReplaceAll(msg.Text(), `"`, `\"`) + `"`)
			if json.Valid([]byte(msg.Text())) {
				toolResultContent = json.RawMessage(msg.Text())
			}
			parts = []anthropicContent{{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   toolResultContent,
			}}
			messages = append(messages, anthropicMessage{
				Role:    "user",
				Content: parts,
			})
		default:
			if len(parts) > 0 {
				messages = append(messages, anthropicMessage{
					Role:    "user",
					Content: parts,
				})
			}
		}
	}

	if len(messages) == 0 {
		return anthropicRequestBody{}, apperror.Validation("anthropic.buildMessageRequest", "no user/assistant messages provided")
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

	// Add tools if present
	if len(req.Tools) > 0 {
		body.Tools = convertToolsToAnthropic(req.Tools)
	}

	// Add tool_choice if present
	if len(req.ToolChoice) > 0 {
		toolChoice, err := convertToolChoiceToAnthropic(req.ToolChoice, req.ParallelToolCalls)
		if err != nil {
			return anthropicRequestBody{}, err
		}
		body.ToolChoice = toolChoice
	}

	return body, nil
}

// convertToolsToAnthropic converts OpenAI-style tools to Anthropic format.
func convertToolsToAnthropic(tools []models.Tool) []anthropicTool {
	result := make([]anthropicTool, 0, len(tools))
	for _, tool := range tools {
		if tool.Type != "function" && tool.Type != "" {
			continue
		}
		at := anthropicTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
		}
		// Anthropic uses "input_schema" instead of "parameters"
		if len(tool.Function.Parameters) > 0 {
			at.InputSchema = tool.Function.Parameters
		} else {
			// Default to empty object schema
			at.InputSchema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		result = append(result, at)
	}
	return result
}

// convertToolChoiceToAnthropic converts OpenAI-style tool_choice to Anthropic format.
func convertToolChoiceToAnthropic(raw json.RawMessage, parallelToolCalls *bool) (*anthropicToolChoice, error) {
	// Try string first: "none", "auto", "required"
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		mode := strings.ToLower(strings.TrimSpace(str))
		switch mode {
		case "none":
			// Anthropic doesn't have "none" - we'll omit tool_choice (default behavior)
			return nil, nil
		case "auto":
			choice := &anthropicToolChoice{Type: "auto"}
			if parallelToolCalls != nil && !*parallelToolCalls {
				choice.DisableParallelToolUse = true
			}
			return choice, nil
		case "required":
			// Anthropic's "any" forces a tool call
			choice := &anthropicToolChoice{Type: "any"}
			if parallelToolCalls != nil && !*parallelToolCalls {
				choice.DisableParallelToolUse = true
			}
			return choice, nil
		default:
			return &anthropicToolChoice{Type: mode}, nil
		}
	}

	// Try object: { "type": "function", "function": { "name": "..." } }
	var obj struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("invalid tool_choice format: %w", err)
	}

	if obj.Function.Name != "" {
		choice := &anthropicToolChoice{
			Type: "tool",
			Name: obj.Function.Name,
		}
		if parallelToolCalls != nil && !*parallelToolCalls {
			choice.DisableParallelToolUse = true
		}
		return choice, nil
	}

	return nil, nil
}

type anthropicRequestBody struct {
	Model         string               `json:"model"`
	System        string               `json:"system,omitempty"`
	Messages      []anthropicMessage   `json:"messages"`
	MaxTokens     int32                `json:"max_tokens"`
	Temperature   float64              `json:"temperature,omitempty"`
	TopP          float64              `json:"top_p,omitempty"`
	StopSequences []string             `json:"stop_sequences,omitempty"`
	Stream        bool                 `json:"stream,omitempty"`
	Tools         []anthropicTool      `json:"tools,omitempty"`
	ToolChoice    *anthropicToolChoice `json:"tool_choice,omitempty"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicToolChoice struct {
	Type                   string `json:"type"`                                // "auto", "any", or "tool"
	Name                   string `json:"name,omitempty"`                      // Only for type "tool"
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"` // optional
}

// Reuse structures from bedrock adapter-style

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type   string                `json:"type"`
	Text   string                `json:"text,omitempty"`
	Source *anthropicImageSource `json:"source,omitempty"`
	// Tool use fields (for responses)
	ID    string          `json:"id,omitempty"`    // tool_use ID
	Name  string          `json:"name,omitempty"`  // tool name
	Input json.RawMessage `json:"input,omitempty"` // tool arguments as JSON
	// Tool result fields (for requests)
	ToolUseID string          `json:"tool_use_id,omitempty"` // ID of the tool_use being responded to
	Content   json.RawMessage `json:"content,omitempty"`     // Result content (can be string or array)
}

type anthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
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
	Type         string                       `json:"type"`
	Index        int                          `json:"index"`
	Message      *anthropicStreamMessage      `json:"message"`
	Delta        *anthropicStreamDelta        `json:"delta"`
	ContentBlock *anthropicStreamContentBlock `json:"content_block"`
	Usage        anthropicUsage               `json:"usage"`
}

type anthropicStreamMessage struct {
	ID    string `json:"id"`
	Model string `json:"model"`
}

type anthropicStreamDelta struct {
	Type         string `json:"type"`
	Text         string `json:"text"`
	PartialJSON  string `json:"partial_json"` // For input_json_delta in tool calls
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
}

type anthropicStreamContentBlock struct {
	Type string `json:"type"` // "text" or "tool_use"
	ID   string `json:"id"`   // For tool_use blocks
	Name string `json:"name"` // For tool_use blocks
	Text string `json:"text"` // For text blocks
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

func (a *Adapter) convertMessageParts(ctx context.Context, msgIndex int, msg models.ChatMessage, allowNonText bool) ([]anthropicContent, error) {
	parts := msg.ContentParts
	if len(parts) == 0 {
		text := strings.TrimSpace(msg.Text())
		if text == "" {
			return nil, nil
		}
		parts = []models.MessageContentPart{{Type: models.MessageContentPartTypeText, Text: text}}
	}

	converted := make([]anthropicContent, 0, len(parts))
	for partIdx, part := range parts {
		if part.IsTextual() {
			text := strings.TrimSpace(part.Text)
			if text != "" {
				converted = append(converted, anthropicContent{Type: "text", Text: text})
			}
			continue
		}

		if !allowNonText {
			partType := strings.TrimSpace(part.Type)
			if partType == "" {
				partType = "non-text"
			}
			return nil, fmt.Errorf("anthropic: message %d does not support %s content", msgIndex, partType)
		}

		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case models.MessageContentPartTypeImageURL:
			content, err := a.imageContentFromURL(ctx, part.ImageURL)
			if err != nil {
				return nil, fmt.Errorf("anthropic: message %d image_url: %w", msgIndex, err)
			}
			converted = append(converted, content)
		case models.MessageContentPartTypeImageFile:
			return nil, fmt.Errorf("anthropic: message %d image_file content is not supported yet", msgIndex)
		case models.MessageContentPartTypeInputAudio:
			return nil, fmt.Errorf("anthropic: message %d input_audio content is not supported", msgIndex)
		case "input_image":
			content, err := imageContentFromInline(part.InputImage)
			if err != nil {
				return nil, fmt.Errorf("anthropic: message %d input_image: %w", msgIndex, err)
			}
			converted = append(converted, content)
		default:
			return nil, fmt.Errorf("anthropic: message %d unsupported content part %q (index %d)", msgIndex, part.Type, partIdx)
		}
	}

	return converted, nil
}

func (a *Adapter) imageContentFromURL(ctx context.Context, image *models.MessageContentImageURL) (anthropicContent, error) {
	if image == nil || strings.TrimSpace(image.URL) == "" {
		return anthropicContent{}, apperror.Validation("anthropic.imageContentFromURL", "image_url missing url")
	}
	urlValue := strings.TrimSpace(image.URL)
	var mime string
	var data []byte
	var err error
	if strings.HasPrefix(strings.ToLower(urlValue), "data:") {
		mime, data, err = decodeDataURL(urlValue)
	} else {
		mime, data, err = a.fetchExternalAsset(ctx, urlValue)
	}
	if err != nil {
		return anthropicContent{}, err
	}
	return inlineImageContent(mime, data)
}

func imageContentFromInline(image *models.MessageContentImageObject) (anthropicContent, error) {
	if image == nil {
		return anthropicContent{}, apperror.Validation("anthropic.imageContentFromInline", "image payload missing")
	}
	encoded := strings.TrimSpace(image.Data)
	if encoded == "" {
		return anthropicContent{}, apperror.Validation("anthropic.imageContentFromInline", "image data missing")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return anthropicContent{}, fmt.Errorf("decode inline image: %w", err)
	}
	return inlineImageContent(image.Format, data)
}

func inlineImageContent(format string, data []byte) (anthropicContent, error) {
	if len(data) > anthropicMaxInlineDataBytes {
		return anthropicContent{}, fmt.Errorf("asset exceeds %d bytes", anthropicMaxInlineDataBytes)
	}
	mime := normalizeAnthropicMime(format)
	return anthropicContent{
		Type: "image",
		Source: &anthropicImageSource{
			Type:      "base64",
			MediaType: mime,
			Data:      base64.StdEncoding.EncodeToString(data),
		},
	}, nil
}

func normalizeAnthropicMime(format string) string {
	format = strings.TrimSpace(format)
	if format == "" {
		return "image/png"
	}
	if strings.Contains(format, "/") {
		return format
	}
	if strings.HasPrefix(format, "image") {
		return format
	}
	return "image/" + format
}

func (a *Adapter) fetchExternalAsset(ctx context.Context, raw string) (string, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", nil, err
	}
	resp, err := a.assetClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", nil, fmt.Errorf("remote asset status %d", resp.StatusCode)
	}
	reader := io.LimitReader(resp.Body, anthropicMaxInlineDataBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, fmt.Errorf("read remote asset: %w", err)
	}
	if len(data) > anthropicMaxInlineDataBytes {
		return "", nil, fmt.Errorf("remote asset exceeds %d bytes", anthropicMaxInlineDataBytes)
	}
	mime := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if mime == "" {
		mime = http.DetectContentType(data)
	}
	return mime, data, nil
}

func decodeDataURL(raw string) (string, []byte, error) {
	payload := strings.TrimPrefix(raw, "data:")
	parts := strings.SplitN(payload, ",", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid data url")
	}
	meta := parts[0]
	dataPart := parts[1]
	segments := strings.Split(meta, ";")
	mime := strings.TrimSpace(segments[0])
	base64Encoded := false
	for _, seg := range segments[1:] {
		if strings.EqualFold(strings.TrimSpace(seg), "base64") {
			base64Encoded = true
		}
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	var decoded []byte
	var err error
	if base64Encoded {
		decoded, err = base64.StdEncoding.DecodeString(dataPart)
	} else {
		decodedString, decodeErr := url.QueryUnescape(dataPart)
		if decodeErr != nil {
			err = decodeErr
		} else {
			decoded = []byte(decodedString)
		}
	}
	if err != nil {
		return "", nil, fmt.Errorf("decode data url: %w", err)
	}
	if len(decoded) > anthropicMaxInlineDataBytes {
		return "", nil, fmt.Errorf("data url exceeds %d bytes", anthropicMaxInlineDataBytes)
	}
	return mime, decoded, nil
}

func anthropicContentsToMessage(role string, content []anthropicContent) models.ChatMessage {
	parts := make([]models.MessageContentPart, 0, len(content))
	var toolCalls []models.ToolCall

	for _, block := range content {
		switch strings.ToLower(block.Type) {
		case "text":
			if strings.TrimSpace(block.Text) != "" {
				parts = append(parts, models.MessageContentPart{
					Type: models.MessageContentPartTypeText,
					Text: block.Text,
				})
			}
		case "image":
			if block.Source != nil && strings.TrimSpace(block.Source.Data) != "" {
				media := block.Source.MediaType
				if strings.TrimSpace(media) == "" {
					media = "image/png"
				}
				parts = append(parts, models.MessageContentPart{
					Type: models.MessageContentPartTypeImageURL,
					ImageURL: &models.MessageContentImageURL{
						URL: fmt.Sprintf("data:%s;base64,%s", media, block.Source.Data),
					},
				})
			}
		case "tool_use":
			// Convert Anthropic tool_use to OpenAI-compatible tool_call
			arguments := ""
			if len(block.Input) > 0 {
				arguments = string(block.Input)
			}
			toolCalls = append(toolCalls, models.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: models.ToolCallFunction{
					Name:      block.Name,
					Arguments: arguments,
				},
			})
		}
	}
	text := models.TextFromContentParts(parts)
	if strings.TrimSpace(role) == "" {
		role = "assistant"
	}
	return models.ChatMessage{
		Role:         role,
		Content:      text,
		ContentParts: parts,
		ToolCalls:    toolCalls,
	}
}

func convertAnthropicResponse(resp anthropicResponse, model string) models.ChatResponse {
	message := anthropicContentsToMessage(resp.Role, resp.Content)
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
	case "tool_use":
		return "tool_calls"
	default:
		return reason
	}
}

func decodeAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("anthropic api error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
