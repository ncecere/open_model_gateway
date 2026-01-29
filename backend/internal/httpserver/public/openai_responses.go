package public

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/logging"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	"github.com/ncecere/open_model_gateway/backend/internal/services/responsestore"
)

// ---------------------------------------------------------------------------
// Request types (Open Responses spec)
// ---------------------------------------------------------------------------

type openAIResponsesRequest struct {
	Model              string            `json:"model"`
	Input              json.RawMessage   `json:"input"`
	Instructions       string            `json:"instructions"`
	PreviousResponseID string            `json:"previous_response_id"`
	Temperature        *float64          `json:"temperature,omitempty"`
	TopP               *float64          `json:"top_p,omitempty"`
	PresencePenalty    *float64          `json:"presence_penalty,omitempty"`
	FrequencyPenalty   *float64          `json:"frequency_penalty,omitempty"`
	MaxOutputTokens    *int32            `json:"max_output_tokens,omitempty"`
	MaxToolCalls       *int32            `json:"max_tool_calls,omitempty"`
	TopLogprobs        *int32            `json:"top_logprobs,omitempty"`
	Metadata           map[string]string `json:"metadata"`
	Stream             bool              `json:"stream,omitempty"`
	StreamOptions      *streamOptions    `json:"stream_options,omitempty"`
	Tools              json.RawMessage   `json:"tools"`
	ToolChoice         json.RawMessage   `json:"tool_choice"`
	Text               *textParam        `json:"text,omitempty"`
	Truncation         string            `json:"truncation,omitempty"`
	Include            []string          `json:"include,omitempty"`
	Reasoning          *reasoningParam   `json:"reasoning,omitempty"`
	Store              *bool             `json:"store,omitempty"`
	Background         *bool             `json:"background,omitempty"`
	ServiceTier        string            `json:"service_tier,omitempty"`
	SafetyIdentifier   string            `json:"safety_identifier,omitempty"`
	PromptCacheKey     string            `json:"prompt_cache_key,omitempty"`
	ParallelToolCalls  *bool             `json:"parallel_tool_calls,omitempty"`
}

type streamOptions struct {
	IncludeObfuscation *bool `json:"include_obfuscation,omitempty"`
}

type textParam struct {
	Format    json.RawMessage `json:"format,omitempty"`
	Verbosity string          `json:"verbosity,omitempty"`
}

type reasoningParam struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// ---------------------------------------------------------------------------
// Response types (Open Responses spec — all required fields)
// ---------------------------------------------------------------------------

type responsesPayload struct {
	ID                 string                `json:"id"`
	Object             string                `json:"object"`
	CreatedAt          int64                 `json:"created_at"`
	CompletedAt        *int64                `json:"completed_at"`
	Status             string                `json:"status"`
	IncompleteDetails  *incompleteDetails    `json:"incomplete_details"`
	Model              string                `json:"model"`
	PreviousResponseID *string               `json:"previous_response_id"`
	Instructions       *string               `json:"instructions"`
	Output             []responsesOutputItem `json:"output"`
	Error              *responsesError       `json:"error"`
	Tools              []responsesToolDef    `json:"tools"`
	ToolChoice         json.RawMessage       `json:"tool_choice"`
	Truncation         string                `json:"truncation"`
	ParallelToolCalls  bool                  `json:"parallel_tool_calls"`
	Text               *responsesTextField   `json:"text"`
	TopP               *float64              `json:"top_p"`
	PresencePenalty    *float64              `json:"presence_penalty"`
	FrequencyPenalty   *float64              `json:"frequency_penalty"`
	TopLogprobs        *int32                `json:"top_logprobs"`
	Temperature        *float64              `json:"temperature"`
	Reasoning          *responsesReasoning   `json:"reasoning"`
	Usage              responsesUsage        `json:"usage"`
	MaxOutputTokens    *int32                `json:"max_output_tokens"`
	MaxToolCalls       *int32                `json:"max_tool_calls"`
	Store              bool                  `json:"store"`
	Background         bool                  `json:"background"`
	ServiceTier        string                `json:"service_tier"`
	Metadata           map[string]string     `json:"metadata"`
	SafetyIdentifier   *string               `json:"safety_identifier"`
	PromptCacheKey     *string               `json:"prompt_cache_key"`
}

type incompleteDetails struct {
	Reason string `json:"reason"`
}

type responsesError struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Type    string  `json:"type,omitempty"`
	Param   *string `json:"param,omitempty"`
}

// responsesOutputItem is a union: message | function_call | function_call_output | reasoning.
// We use a single struct with all possible fields and discriminate on Type.
type responsesOutputItem struct {
	Type string `json:"type"`
	ID   string `json:"id"`

	// message fields
	Status  string                   `json:"status"`
	Role    string                   `json:"role,omitempty"`
	Content []responsesOutputContent `json:"content,omitempty"`

	// function_call fields
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`

	// reasoning fields
	Summary []responsesOutputContent `json:"summary,omitempty"`
}

type responsesOutputContent struct {
	Type        string        `json:"type"`
	Text        string        `json:"text,omitempty"`
	Annotations []interface{} `json:"annotations"`
}

type responsesToolDef struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      *bool           `json:"strict"`
}

type responsesTextField struct {
	Format    json.RawMessage `json:"format,omitempty"`
	Verbosity string          `json:"verbosity,omitempty"`
}

type responsesReasoning struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary"`
}

type responsesUsage struct {
	InputTokens         int32                `json:"input_tokens"`
	OutputTokens        int32                `json:"output_tokens"`
	TotalTokens         int32                `json:"total_tokens"`
	InputTokensDetails  responsesInputUsage  `json:"input_tokens_details"`
	OutputTokensDetails responsesOutputUsage `json:"output_tokens_details"`
}

type responsesInputUsage struct {
	CachedTokens int32 `json:"cached_tokens"`
}

type responsesOutputUsage struct {
	ReasoningTokens int32 `json:"reasoning_tokens"`
}

// openAIResponseOptions carries request-level context needed when building the response payload.
type openAIResponseOptions struct {
	Instructions       string
	Metadata           map[string]string
	ParallelToolCalls  bool
	Tools              []responsesToolDef
	ToolChoice         json.RawMessage
	Truncation         string
	Temperature        *float64
	TopP               *float64
	PresencePenalty    *float64
	FrequencyPenalty   *float64
	TopLogprobs        *int32
	MaxOutputTokens    *int32
	MaxToolCalls       *int32
	Text               *responsesTextField
	Reasoning          *responsesReasoning
	Store              bool
	Background         bool
	ServiceTier        string
	SafetyIdentifier   string
	PromptCacheKey     string
	PreviousResponseID string
}

// ---------------------------------------------------------------------------
// Input item types (Open Responses spec — polymorphic)
// ---------------------------------------------------------------------------

type responsesInputItem struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`

	// message fields
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	Status  string          `json:"status,omitempty"`

	// function_call fields
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`

	// function_call_output fields
	Output json.RawMessage `json:"output,omitempty"`

	// item_reference fields (type=item_reference, just has id)

	// reasoning fields
	Summary          json.RawMessage `json:"summary,omitempty"`
	EncryptedContent string          `json:"encrypted_content,omitempty"`
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

func (h *openAIHandler) responses(c *fiber.Ctx) error {
	var req openAIResponsesRequest
	if err := c.BodyParser(&req); err != nil {
		return writeResponsesError(c, fiber.StatusBadRequest, "invalid_request", "invalid_request_body", "invalid request body", nil)
	}
	alias := strings.TrimSpace(req.Model)
	if alias == "" {
		return writeResponsesError(c, fiber.StatusBadRequest, "invalid_request", "model_required", "model is required", strPtr("model"))
	}
	if len(req.Input) == 0 {
		return writeResponsesError(c, fiber.StatusBadRequest, "invalid_request", "input_required", "input is required", strPtr("input"))
	}
	if err := validateResponseMetadata(req.Metadata); err != nil {
		return writeResponsesError(c, fiber.StatusBadRequest, "invalid_request", "invalid_metadata", err.Error(), strPtr("metadata"))
	}

	// Reconstruct conversation from previous_response_id if provided
	var previousMessages []models.ChatMessage
	if req.PreviousResponseID != "" {
		if h.container.ResponseStore == nil {
			return writeResponsesError(c, fiber.StatusBadRequest, "invalid_request", "unsupported_parameter", "response storage is not available", strPtr("previous_response_id"))
		}
		stored, err := h.container.ResponseStore.Get(c.UserContext(), req.PreviousResponseID)
		if err != nil {
			return writeResponsesError(c, fiber.StatusNotFound, "invalid_request", "response_not_found", "previous response not found or expired", strPtr("previous_response_id"))
		}
		// Reconstruct messages from stored input + output
		prevInput, err := parseResponseInputItems(stored.Input)
		if err == nil {
			previousMessages = append(previousMessages, prevInput...)
		}
		prevOutput, err := outputItemsToMessages(stored.Output)
		if err == nil {
			previousMessages = append(previousMessages, prevOutput...)
		}
	}

	messages, err := buildResponseMessages(req.Instructions, req.Input)
	if err != nil {
		return writeResponsesError(c, fiber.StatusBadRequest, "invalid_request", "invalid_input", err.Error(), strPtr("input"))
	}
	// Prepend previous conversation context
	if len(previousMessages) > 0 {
		full := make([]models.ChatMessage, 0, len(previousMessages)+len(messages))
		// Keep developer/system instruction at the front if present
		if len(messages) > 0 && (messages[0].Role == "developer" || messages[0].Role == "system") {
			full = append(full, messages[0])
			full = append(full, previousMessages...)
			full = append(full, messages[1:]...)
		} else {
			full = append(full, previousMessages...)
			full = append(full, messages...)
		}
		messages = full
	}

	// Apply truncation if configured
	truncationMode := strings.TrimSpace(req.Truncation)
	if truncationMode == "" {
		truncationMode = "disabled"
	}
	if truncationMode == "auto" {
		model, lookupErr := h.container.Queries.GetModelByAlias(c.UserContext(), alias)
		if lookupErr == nil && model.ContextWindow > 0 {
			messages = truncateMessages(messages, int(model.ContextWindow))
		}
	}

	// Parse tools
	var tools []models.Tool
	var toolDefs []responsesToolDef
	if len(req.Tools) > 0 {
		var parsedTools []openAITool
		if err := json.Unmarshal(req.Tools, &parsedTools); err != nil {
			return writeResponsesError(c, fiber.StatusBadRequest, "invalid_request", "invalid_tools", "invalid tools format", strPtr("tools"))
		}
		tools = make([]models.Tool, 0, len(parsedTools))
		toolDefs = make([]responsesToolDef, 0, len(parsedTools))
		for _, t := range parsedTools {
			tools = append(tools, models.Tool{
				Type: t.Type,
				Function: models.ToolFunction{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
					Strict:      t.Function.Strict,
				},
			})
			toolDefs = append(toolDefs, responsesToolDef{
				Type:        "function",
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
				Strict:      t.Function.Strict,
			})
		}
	}

	// Handle allowed_tools in tool_choice: filter tools list and convert to "required"
	providerToolChoice := req.ToolChoice
	if len(req.ToolChoice) > 0 && len(tools) > 0 {
		tools, toolDefs, providerToolChoice = applyAllowedTools(req.ToolChoice, tools, toolDefs)
	}

	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return writeResponsesError(c, fiber.StatusInternalServerError, "server_error", "context_missing", "request context missing", nil)
	}
	if !h.container.IsModelAllowed(rc.TenantID, alias) {
		return writeResponsesError(c, fiber.StatusForbidden, "invalid_request", "model_not_allowed", "model not enabled for tenant", strPtr("model"))
	}

	// Validate that the model supports tools if tools are provided
	if len(tools) > 0 {
		model, err := h.container.Queries.GetModelByAlias(ctx, alias)
		if err == nil && !model.SupportsTools {
			return writeResponsesError(c, fiber.StatusBadRequest, "invalid_request", "tools_not_supported", "model does not support tool calling", strPtr("tools"))
		}
	}

	traceID := traceIDFromContext(c)
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))

	// Map temperature from float64 to float32 for internal ChatRequest
	var tempF32 *float32
	if req.Temperature != nil {
		v := float32(*req.Temperature)
		tempF32 = &v
	}
	var topPF32 *float32
	if req.TopP != nil {
		v := float32(*req.TopP)
		topPF32 = &v
	}

	var presF32 *float32
	if req.PresencePenalty != nil {
		v := float32(*req.PresencePenalty)
		presF32 = &v
	}
	var freqF32 *float32
	if req.FrequencyPenalty != nil {
		v := float32(*req.FrequencyPenalty)
		freqF32 = &v
	}

	// Validate and convert text.format to ChatRequest response_format if present
	var chatResponseFormat json.RawMessage
	if req.Text != nil && len(req.Text.Format) > 0 {
		if err := validateTextFormat(req.Text.Format); err != nil {
			return writeResponsesError(c, fiber.StatusBadRequest, "invalid_request", "invalid_text_format", err.Error(), strPtr("text.format"))
		}
		chatResponseFormat = req.Text.Format
	}

	modelReq := models.ChatRequest{
		Messages:           messages,
		Temperature:        tempF32,
		TopP:               topPF32,
		MaxTokens:          req.MaxOutputTokens,
		PresencePenalty:    presF32,
		FrequencyPenalty:   freqF32,
		ChatResponseFormat: chatResponseFormat,
		Tools:              tools,
		ToolChoice:         providerToolChoice,
		ParallelToolCalls:  req.ParallelToolCalls,
	}

	parallel := true
	if req.ParallelToolCalls != nil {
		parallel = *req.ParallelToolCalls
	}

	// Resolve truncation default
	truncation := strings.TrimSpace(req.Truncation)
	if truncation == "" {
		truncation = "disabled"
	}

	// Resolve tool_choice for echo-back
	toolChoiceEcho := req.ToolChoice
	if len(toolChoiceEcho) == 0 {
		toolChoiceEcho = json.RawMessage(`"auto"`)
	}

	// Build text field echo
	var textField *responsesTextField
	if req.Text != nil {
		textField = &responsesTextField{
			Format:    req.Text.Format,
			Verbosity: req.Text.Verbosity,
		}
	}

	// Build reasoning echo
	var reasoningField *responsesReasoning
	if req.Reasoning != nil {
		reasoningField = &responsesReasoning{
			Effort:  req.Reasoning.Effort,
			Summary: req.Reasoning.Summary,
		}
	}

	storeBool := false
	if req.Store != nil {
		storeBool = *req.Store
	}
	bgBool := false
	if req.Background != nil {
		bgBool = *req.Background
	}

	options := openAIResponseOptions{
		Instructions:       strings.TrimSpace(req.Instructions),
		Metadata:           req.Metadata,
		ParallelToolCalls:  parallel,
		Tools:              toolDefs,
		ToolChoice:         toolChoiceEcho,
		Truncation:         truncation,
		Temperature:        req.Temperature,
		TopP:               req.TopP,
		PresencePenalty:    req.PresencePenalty,
		FrequencyPenalty:   req.FrequencyPenalty,
		TopLogprobs:        req.TopLogprobs,
		MaxOutputTokens:    req.MaxOutputTokens,
		MaxToolCalls:       req.MaxToolCalls,
		Text:               textField,
		Reasoning:          reasoningField,
		Store:              storeBool,
		Background:         bgBool,
		ServiceTier:        req.ServiceTier,
		SafetyIdentifier:   req.SafetyIdentifier,
		PromptCacheKey:     req.PromptCacheKey,
		PreviousResponseID: req.PreviousResponseID,
	}

	// Enrich wide event with Responses API metadata
	if event, ok := logging.WideEventFromContext(c.UserContext()); ok {
		event.ResponsesAPI = true
		event.Truncation = truncation
		event.PreviousResponseID = req.PreviousResponseID
		event.ToolCallCount = len(tools)
	}

	// Record Prometheus counter for Responses API
	if obs := h.container.Observability; obs != nil {
		obs.RecordResponsesRequest(alias, "ok", truncation, len(tools) > 0, reasoningField != nil)
	}

	// Capture input for response storage
	rawInput := req.Input
	responseStore := h.container.ResponseStore
	tenantID := rc.TenantID

	if req.Stream {
		return h.chatStreamPipeline.StreamResponses(c, rc, alias, traceID, idempotencyKey, modelReq, options)
	}

	return h.chatPipeline.ExecuteWithConverter(c, rc, alias, traceID, idempotencyKey, modelReq, func(resp models.ChatResponse, alias string) (interface{}, error) {
		payload := convertResponsesResponse(resp, alias, options)

		// Store response for previous_response_id support
		if responseStore != nil && storeBool {
			outputJSON, _ := json.Marshal(payload.Output)
			_ = responseStore.Store(c.UserContext(), responsestore.StoredResponse{
				ID:           payload.ID,
				TenantID:     tenantID,
				Model:        alias,
				Input:        rawInput,
				Output:       outputJSON,
				Instructions: strings.TrimSpace(req.Instructions),
				Metadata:     req.Metadata,
			})
		}

		return payload, nil
	})
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func validateTextFormat(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var obj struct {
		Type       string          `json:"type"`
		JSONSchema json.RawMessage `json:"json_schema"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return fmt.Errorf("text.format must be an object with a \"type\" field")
	}
	switch obj.Type {
	case "text", "":
		return nil
	case "json_object":
		return nil
	case "json_schema":
		if len(obj.JSONSchema) == 0 {
			return fmt.Errorf("text.format type \"json_schema\" requires a \"json_schema\" field")
		}
		// Validate that json_schema has at minimum a "name" field
		var schema struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(obj.JSONSchema, &schema); err != nil || schema.Name == "" {
			return fmt.Errorf("text.format json_schema requires a \"name\" field")
		}
		return nil
	default:
		return fmt.Errorf("unsupported text.format type %q; must be \"text\", \"json_object\", or \"json_schema\"", obj.Type)
	}
}

func validateResponseMetadata(md map[string]string) error {
	if len(md) == 0 {
		return nil
	}
	if len(md) > 16 {
		return fmt.Errorf("metadata supports at most 16 key/value pairs")
	}
	for key, value := range md {
		if len(key) > 64 {
			return fmt.Errorf("metadata key %q exceeds 64 characters", key)
		}
		if len(value) > 512 {
			return fmt.Errorf("metadata value for %q exceeds 512 characters", key)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Input parsing — polymorphic items per Open Responses spec
// ---------------------------------------------------------------------------

func buildResponseMessages(instructions string, input json.RawMessage) ([]models.ChatMessage, error) {
	messages := make([]models.ChatMessage, 0, 1)
	if instr := strings.TrimSpace(instructions); instr != "" {
		messages = append(messages, models.ChatMessage{Role: "developer", Content: instr})
	}
	inputMessages, err := parseResponseInputItems(input)
	if err != nil {
		return nil, err
	}
	messages = append(messages, inputMessages...)
	return messages, nil
}

func parseResponseInputItems(raw json.RawMessage) ([]models.ChatMessage, error) {
	if len(raw) == 0 {
		return nil, errors.New("input is required")
	}
	// Try string shorthand first
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []models.ChatMessage{{Role: "user", Content: text}}, nil
	}
	// Try as polymorphic item array
	var items []responsesInputItem
	if err := json.Unmarshal(raw, &items); err == nil {
		if len(items) == 0 {
			return nil, errors.New("input array must contain at least one item")
		}
		return convertResponsesInputItems(items)
	}
	return nil, errors.New("input must be a string or array of input items")
}

func convertResponsesInputItems(items []responsesInputItem) ([]models.ChatMessage, error) {
	msgs := make([]models.ChatMessage, 0, len(items))
	for i, item := range items {
		switch item.Type {
		case "message":
			msg, err := convertResponsesMessageItem(item)
			if err != nil {
				return nil, fmt.Errorf("input[%d]: %w", i, err)
			}
			msgs = append(msgs, msg)

		case "function_call":
			// Assistant message with a tool call
			msgs = append(msgs, models.ChatMessage{
				Role:    "assistant",
				Content: "",
				ToolCalls: []models.ToolCall{
					{
						ID:   item.CallID,
						Type: "function",
						Function: models.ToolCallFunction{
							Name:      item.Name,
							Arguments: item.Arguments,
						},
					},
				},
			})

		case "function_call_output":
			// Tool response message
			var output string
			if len(item.Output) > 0 {
				// Output can be a string or structured content; for now treat as string
				if err := json.Unmarshal(item.Output, &output); err != nil {
					// If not a plain string, stringify the JSON
					output = string(item.Output)
				}
			}
			msgs = append(msgs, models.ChatMessage{
				Role:       "tool",
				Content:    output,
				ToolCallID: item.CallID,
			})

		case "item_reference":
			// Item references require previous_response_id support; skip for now
			continue

		case "reasoning":
			// Reasoning items are echoed back for context; pass as a system hint
			// Most providers will ignore this, but it preserves the conversation shape
			continue

		case "":
			// Legacy: no type field — try as a plain message with role/content
			msg, err := convertResponsesMessageItem(item)
			if err != nil {
				return nil, fmt.Errorf("input[%d]: %w", i, err)
			}
			msgs = append(msgs, msg)

		default:
			// Unknown/extension types — skip gracefully per spec
			continue
		}
	}
	if len(msgs) == 0 {
		return nil, errors.New("input must contain at least one processable item")
	}
	return msgs, nil
}

func convertResponsesMessageItem(item responsesInputItem) (models.ChatMessage, error) {
	role := strings.TrimSpace(item.Role)
	if role == "" {
		role = "user"
	}
	// Map Open Responses roles to provider roles
	switch role {
	case "developer":
		role = "system" // Most providers understand "system" not "developer"
	case "user", "assistant", "system":
		// pass through
	default:
		return models.ChatMessage{}, fmt.Errorf("unsupported role %q", role)
	}

	// Content can be a string or array of content parts
	if len(item.Content) == 0 {
		return models.ChatMessage{Role: role}, nil
	}

	var text string
	if err := json.Unmarshal(item.Content, &text); err == nil {
		return models.ChatMessage{Role: role, Content: text}, nil
	}

	// Try as array of content parts (InputTextContentParam, InputImageContentParam, etc.)
	var parts []models.MessageContentPart
	if err := json.Unmarshal(item.Content, &parts); err == nil {
		msg := models.ChatMessage{Role: role, ContentParts: parts}
		// Also set plain text content for providers that don't support parts
		msg.Content = models.TextFromContentParts(parts)
		return msg, nil
	}

	return models.ChatMessage{}, errors.New("invalid content format")
}

// ---------------------------------------------------------------------------
// Response conversion (sync)
// ---------------------------------------------------------------------------

// applyAllowedTools checks if tool_choice contains an "allowed" array per the
// Open Responses spec. If so, it filters the tools/toolDefs to only those named
// in the allowed list and returns tool_choice as "required" for the provider
// (since all remaining tools are allowed, the provider should use any of them).
// If tool_choice does not contain "allowed", the original values are returned unchanged.
func applyAllowedTools(toolChoice json.RawMessage, tools []models.Tool, toolDefs []responsesToolDef) ([]models.Tool, []responsesToolDef, json.RawMessage) {
	// Try to parse as object with "allowed" field
	var obj struct {
		Type    string   `json:"type"`
		Allowed []string `json:"allowed"`
		Name    string   `json:"name"`
	}
	if err := json.Unmarshal(toolChoice, &obj); err != nil {
		return tools, toolDefs, toolChoice
	}
	if len(obj.Allowed) == 0 {
		return tools, toolDefs, toolChoice
	}

	// Build allow set
	allowSet := make(map[string]bool, len(obj.Allowed))
	for _, name := range obj.Allowed {
		allowSet[name] = true
	}

	// Filter tools
	filteredTools := make([]models.Tool, 0, len(tools))
	filteredDefs := make([]responsesToolDef, 0, len(toolDefs))
	for _, t := range tools {
		if allowSet[t.Function.Name] {
			filteredTools = append(filteredTools, t)
		}
	}
	for _, d := range toolDefs {
		if allowSet[d.Name] {
			filteredDefs = append(filteredDefs, d)
		}
	}

	// Convert tool_choice to "required" for the provider
	providerChoice := json.RawMessage(`"required"`)
	return filteredTools, filteredDefs, providerChoice
}

// truncateMessages trims older non-system messages to fit within the model's context
// window. Uses a rough chars/4 token estimate. System/developer messages at the
// front and the last user message are always preserved.
func truncateMessages(msgs []models.ChatMessage, contextWindow int) []models.ChatMessage {
	if len(msgs) == 0 || contextWindow <= 0 {
		return msgs
	}
	// Reserve ~25% of context window for output
	maxInputTokens := contextWindow * 3 / 4
	estimateTokens := func(m models.ChatMessage) int {
		text := m.Text()
		n := len(text) / 4
		if n < 1 {
			n = 1
		}
		// Tool calls add overhead
		for _, tc := range m.ToolCalls {
			n += (len(tc.Function.Arguments) + len(tc.Function.Name)) / 4
		}
		return n
	}

	total := 0
	for _, m := range msgs {
		total += estimateTokens(m)
	}
	if total <= maxInputTokens {
		return msgs
	}

	// Identify system/developer prefix and keep it
	prefixEnd := 0
	for i, m := range msgs {
		if m.Role == "system" || m.Role == "developer" {
			prefixEnd = i + 1
		} else {
			break
		}
	}

	// Always keep prefix + last message
	result := make([]models.ChatMessage, 0, len(msgs))
	result = append(result, msgs[:prefixEnd]...)
	budget := maxInputTokens
	for _, m := range result {
		budget -= estimateTokens(m)
	}
	// Always keep the last message
	last := msgs[len(msgs)-1]
	budget -= estimateTokens(last)

	// Add middle messages from most recent backward until budget exhausted
	middle := msgs[prefixEnd : len(msgs)-1]
	kept := make([]models.ChatMessage, 0, len(middle))
	for i := len(middle) - 1; i >= 0; i-- {
		cost := estimateTokens(middle[i])
		if budget-cost < 0 {
			break
		}
		budget -= cost
		kept = append(kept, middle[i])
	}
	// Reverse kept to restore original order
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}
	result = append(result, kept...)
	result = append(result, last)
	return result
}

// outputItemsToMessages converts stored output items (JSON array of responsesOutputItem)
// back into ChatMessages for conversation reconstruction.
func outputItemsToMessages(outputJSON json.RawMessage) ([]models.ChatMessage, error) {
	if len(outputJSON) == 0 {
		return nil, nil
	}
	var items []responsesOutputItem
	if err := json.Unmarshal(outputJSON, &items); err != nil {
		return nil, err
	}
	msgs := make([]models.ChatMessage, 0, len(items))
	for _, item := range items {
		switch item.Type {
		case "message":
			role := item.Role
			if role == "" {
				role = "assistant"
			}
			var content string
			if len(item.Content) > 0 {
				content = item.Content[0].Text
			}
			msgs = append(msgs, models.ChatMessage{Role: role, Content: content})
		case "function_call":
			msgs = append(msgs, models.ChatMessage{
				Role: "assistant",
				ToolCalls: []models.ToolCall{
					{
						ID:   item.CallID,
						Type: "function",
						Function: models.ToolCallFunction{
							Name:      item.Name,
							Arguments: item.Arguments,
						},
					},
				},
			})
		}
	}
	return msgs, nil
}

func convertResponsesResponse(resp models.ChatResponse, alias string, opts openAIResponseOptions) responsesPayload {
	return buildResponsesPayload(resp, alias, opts, "")
}

func buildResponsesPayload(resp models.ChatResponse, alias string, opts openAIResponseOptions, statusOverride string) responsesPayload {
	outputs := make([]responsesOutputItem, 0, len(resp.Choices)*2)
	overallStatus := statusOverride
	if overallStatus == "" {
		overallStatus = "completed"
	}

	for _, choice := range resp.Choices {
		status := mapResponseStatus(choice.FinishReason)
		if status != "completed" && statusOverride == "" {
			overallStatus = status
		}

		// Emit reasoning output item if model returned reasoning content
		reasoning := strings.TrimSpace(choice.Message.Reasoning)
		if reasoning == "" {
			reasoning = strings.TrimSpace(choice.Message.ReasoningContent)
		}
		if reasoning != "" {
			outputs = append(outputs, responsesOutputItem{
				Type:   "reasoning",
				ID:     fmt.Sprintf("rs_%s-%d", resp.ID, choice.Index),
				Status: status,
				Summary: []responsesOutputContent{
					{
						Type: "summary_text",
						Text: reasoning,
					},
				},
			})
		}

		// Emit function_call items for tool calls
		for j, tc := range choice.Message.ToolCalls {
			tcStatus := "completed"
			if status == "incomplete" {
				tcStatus = "incomplete"
			}
			outputs = append(outputs, responsesOutputItem{
				Type:      "function_call",
				ID:        fmt.Sprintf("fc_%s-%d-%d", resp.ID, choice.Index, j),
				Status:    tcStatus,
				CallID:    tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}

		// Emit message item if there's text content
		text := choice.Message.Text()
		if text != "" || len(choice.Message.ToolCalls) == 0 {
			msgRole := choice.Message.Role
			if msgRole == "" {
				msgRole = "assistant"
			}
			outputs = append(outputs, responsesOutputItem{
				Type:   "message",
				ID:     fmt.Sprintf("msg_%s-%d", resp.ID, choice.Index),
				Status: status,
				Role:   msgRole,
				Content: []responsesOutputContent{
					{
						Type:        "output_text",
						Text:        text,
						Annotations: []interface{}{},
					},
				},
			})
		}
	}

	usage := responsesUsage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
		InputTokensDetails: responsesInputUsage{
			CachedTokens: 0,
		},
		OutputTokensDetails: responsesOutputUsage{
			ReasoningTokens: resp.Usage.ReasoningTokens,
		},
	}

	// Build incomplete_details if needed
	var incomplete *incompleteDetails
	if overallStatus == "incomplete" {
		for _, choice := range resp.Choices {
			reason := strings.ToLower(strings.TrimSpace(choice.FinishReason))
			switch reason {
			case "length":
				incomplete = &incompleteDetails{Reason: "max_output_tokens"}
			case "content_filter":
				incomplete = &incompleteDetails{Reason: "content_filter"}
			}
			if incomplete != nil {
				break
			}
		}
	}

	// completed_at
	var completedAt *int64
	if overallStatus == "completed" || overallStatus == "incomplete" {
		ts := time.Now().Unix()
		completedAt = &ts
	}

	// Nullable string fields
	var instrPtr *string
	if instr := strings.TrimSpace(opts.Instructions); instr != "" {
		instrPtr = &instr
	}
	var prevRespPtr *string
	if opts.PreviousResponseID != "" {
		prevRespPtr = &opts.PreviousResponseID
	}
	var safetyPtr *string
	if opts.SafetyIdentifier != "" {
		safetyPtr = &opts.SafetyIdentifier
	}
	var cacheKeyPtr *string
	if opts.PromptCacheKey != "" {
		cacheKeyPtr = &opts.PromptCacheKey
	}

	// Ensure metadata is not nil
	metadata := opts.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}

	// Ensure tools is not nil
	toolsList := opts.Tools
	if toolsList == nil {
		toolsList = []responsesToolDef{}
	}

	serviceTier := opts.ServiceTier
	if serviceTier == "" {
		serviceTier = "default"
	}

	return responsesPayload{
		ID:                 resp.ID,
		Object:             "response",
		CreatedAt:          resp.Created.Unix(),
		CompletedAt:        completedAt,
		Status:             overallStatus,
		IncompleteDetails:  incomplete,
		Model:              alias,
		PreviousResponseID: prevRespPtr,
		Instructions:       instrPtr,
		Output:             outputs,
		Error:              nil,
		Tools:              toolsList,
		ToolChoice:         opts.ToolChoice,
		Truncation:         opts.Truncation,
		ParallelToolCalls:  opts.ParallelToolCalls,
		Text:               opts.Text,
		TopP:               opts.TopP,
		PresencePenalty:    opts.PresencePenalty,
		FrequencyPenalty:   opts.FrequencyPenalty,
		TopLogprobs:        opts.TopLogprobs,
		Temperature:        opts.Temperature,
		Reasoning:          opts.Reasoning,
		Usage:              usage,
		MaxOutputTokens:    opts.MaxOutputTokens,
		MaxToolCalls:       opts.MaxToolCalls,
		Store:              opts.Store,
		Background:         opts.Background,
		ServiceTier:        serviceTier,
		Metadata:           metadata,
		SafetyIdentifier:   safetyPtr,
		PromptCacheKey:     cacheKeyPtr,
	}
}

func mapResponseStatus(reason string) string {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "length", "content_filter":
		return "incomplete"
	case "tool_calls":
		return "completed"
	case "":
		return "completed"
	default:
		return "completed"
	}
}

// ---------------------------------------------------------------------------
// Error helpers (Open Responses spec error shape)
// ---------------------------------------------------------------------------

func writeResponsesError(c *fiber.Ctx, status int, errType, code, message string, param *string) error {
	return c.Status(status).JSON(fiber.Map{
		"error": responsesError{
			Type:    errType,
			Code:    code,
			Message: message,
			Param:   param,
		},
	})
}

func strPtr(s string) *string {
	return &s
}
