package public

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature *float32            `json:"temperature,omitempty"`
	TopP        *float32            `json:"top_p,omitempty"`
	MaxTokens   *int32              `json:"max_tokens,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	StopRaw     json.RawMessage     `json:"stop,omitempty"`

	// Tool calling fields
	Tools             json.RawMessage `json:"tools,omitempty"`
	ToolChoice        json.RawMessage `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool           `json:"parallel_tool_calls,omitempty"`

	// Structured output / response format
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`

	// Legacy function calling (deprecated, converted to tools)
	Functions    json.RawMessage `json:"functions,omitempty"`
	FunctionCall json.RawMessage `json:"function_call,omitempty"`
}

type openAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openAIChatChoice `json:"choices"`
	Usage   openAIUsage        `json:"usage"`
}

func (h *openAIHandler) chatCompletions(c *fiber.Ctx) error {
	var req openAIChatRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if len(req.Messages) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "messages are required")
	}
	stop, err := parseStop(req.StopRaw)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid stop field")
	}

	messages, err := convertOpenAIMessages(req.Messages, "message")
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	// Parse tools (or convert from legacy functions)
	tools, toolChoice, parallelToolCalls, err := parseToolsFromRequest(req)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	if !h.container.IsModelAllowed(rc.TenantID, req.Model) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	// Validate that the model supports tools if tools are provided
	if len(tools) > 0 {
		model, err := h.container.Queries.GetModelByAlias(ctx, req.Model)
		if err == nil && !model.SupportsTools {
			return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support tool calling")
		}
	}

	traceID := traceIDFromContext(c)
	alias := req.Model
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))

	modelReq := models.ChatRequest{
		Messages:           messages,
		Temperature:        req.Temperature,
		TopP:               req.TopP,
		MaxTokens:          req.MaxTokens,
		Stop:               stop,
		Tools:              tools,
		ToolChoice:         toolChoice,
		ParallelToolCalls:  parallelToolCalls,
		ChatResponseFormat: req.ResponseFormat,
	}

	if req.Stream {
		return h.chatStreamPipeline.Stream(c, rc, alias, traceID, idempotencyKey, modelReq)
	}

	return h.chatPipeline.Execute(c, rc, alias, traceID, idempotencyKey, modelReq)
}

// parseToolsFromRequest extracts tools from the request, converting legacy functions if needed.
func parseToolsFromRequest(req openAIChatRequest) ([]models.Tool, json.RawMessage, *bool, error) {
	var tools []models.Tool
	var toolChoice json.RawMessage
	var parallelToolCalls *bool

	// Check for modern tools first
	if len(req.Tools) > 0 {
		var parsedTools []openAITool
		if err := json.Unmarshal(req.Tools, &parsedTools); err != nil {
			return nil, nil, nil, err
		}
		tools = make([]models.Tool, 0, len(parsedTools))
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
		}
		toolChoice = req.ToolChoice
		parallelToolCalls = req.ParallelToolCalls
	} else if len(req.Functions) > 0 {
		// Convert legacy functions to tools
		var functions []models.ToolFunction
		if err := json.Unmarshal(req.Functions, &functions); err != nil {
			return nil, nil, nil, err
		}
		convertedTools, convertedChoice := models.ConvertLegacyFunctions(functions, req.FunctionCall)
		tools = convertedTools
		if convertedChoice != nil {
			choiceBytes, err := json.Marshal(convertedChoice)
			if err != nil {
				return nil, nil, nil, err
			}
			toolChoice = choiceBytes
		}
	}

	return tools, toolChoice, parallelToolCalls, nil
}
