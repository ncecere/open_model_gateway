package openaihelper

import (
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

// BuildChatParams converts a ChatRequest with optional content parts into the
// OpenAI SDK chat completion input, returning an error if unsupported content is
// encountered.
func BuildChatParams(req models.ChatRequest) (openai.ChatCompletionNewParams, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for idx, msg := range req.Messages {
		union, err := convertMessage(idx, msg)
		if err != nil {
			return openai.ChatCompletionNewParams{}, err
		}
		messages = append(messages, union)
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(req.Model),
		Messages: messages,
	}
	if req.Temperature != nil {
		params.Temperature = param.NewOpt(float64(*req.Temperature))
	}
	if req.TopP != nil {
		params.TopP = param.NewOpt(float64(*req.TopP))
	}
	if req.MaxTokens != nil {
		params.MaxTokens = param.NewOpt(int64(*req.MaxTokens))
	}
	if len(req.Stop) == 1 {
		params.Stop.OfString = param.NewOpt(req.Stop[0])
	} else if len(req.Stop) > 1 {
		params.Stop.OfStringArray = append(params.Stop.OfStringArray, req.Stop...)
	}

	// Add tools
	if len(req.Tools) > 0 {
		params.Tools = convertTools(req.Tools)
	}

	// Add tool_choice
	if len(req.ToolChoice) > 0 {
		toolChoice, err := convertToolChoice(req.ToolChoice)
		if err != nil {
			return openai.ChatCompletionNewParams{}, err
		}
		params.ToolChoice = toolChoice
	}

	// Add parallel_tool_calls
	if req.ParallelToolCalls != nil {
		params.ParallelToolCalls = param.NewOpt(*req.ParallelToolCalls)
	}

	return params, nil
}

// convertTools converts the internal Tool representation to OpenAI SDK format.
func convertTools(tools []models.Tool) []openai.ChatCompletionToolUnionParam {
	result := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		if tool.Type != "function" && tool.Type != "" {
			// Skip non-function tools
			continue
		}

		funcDef := shared.FunctionDefinitionParam{
			Name: tool.Function.Name,
		}
		if tool.Function.Description != "" {
			funcDef.Description = param.NewOpt(tool.Function.Description)
		}
		if tool.Function.Strict != nil {
			funcDef.Strict = param.NewOpt(*tool.Function.Strict)
		}
		if len(tool.Function.Parameters) > 0 {
			var params shared.FunctionParameters
			if err := json.Unmarshal(tool.Function.Parameters, &params); err == nil {
				funcDef.Parameters = params
			}
		}

		result = append(result, openai.ChatCompletionFunctionTool(funcDef))
	}
	return result
}

// convertToolChoice converts the tool_choice parameter to OpenAI SDK format.
func convertToolChoice(raw json.RawMessage) (openai.ChatCompletionToolChoiceOptionUnionParam, error) {
	var result openai.ChatCompletionToolChoiceOptionUnionParam

	// Try string first: "none", "auto", "required"
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		mode := strings.ToLower(strings.TrimSpace(str))
		// OfAuto is used for all string values: "none", "auto", "required"
		result.OfAuto = param.NewOpt(mode)
		return result, nil
	}

	// Try object: { "type": "function", "function": { "name": "..." } }
	var obj struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return result, fmt.Errorf("invalid tool_choice format: %w", err)
	}

	if obj.Function.Name != "" {
		result = openai.ToolChoiceOptionFunctionToolChoice(openai.ChatCompletionNamedToolChoiceFunctionParam{
			Name: obj.Function.Name,
		})
	}

	return result, nil
}

func convertMessage(idx int, msg models.ChatMessage) (openai.ChatCompletionMessageParamUnion, error) {
	role := strings.ToLower(strings.TrimSpace(msg.Role))
	switch role {
	case "system", "developer":
		return buildSystemLikeMessage(role, msg)
	case "assistant":
		return buildAssistantMessage(msg), nil
	case "tool":
		return buildToolMessage(msg), nil
	case "user", "":
		return buildUserMessage(idx, msg)
	default:
		return buildUserMessage(idx, msg)
	}
}

func buildSystemLikeMessage(role string, msg models.ChatMessage) (openai.ChatCompletionMessageParamUnion, error) {
	var union openai.ChatCompletionMessageParamUnion
	parts := msg.ContentParts
	if len(parts) > 0 {
		textParts, err := convertTextParts(parts)
		if err != nil {
			return union, err
		}
		if role == "developer" {
			union = openai.DeveloperMessage(textParts)
		} else {
			union = openai.SystemMessage(textParts)
		}
	} else {
		text := strings.TrimSpace(msg.Text())
		if role == "developer" {
			union = openai.DeveloperMessage(text)
		} else {
			union = openai.SystemMessage(text)
		}
	}
	applyName(&union, msg.Name)
	return union, nil
}

func buildAssistantMessage(msg models.ChatMessage) openai.ChatCompletionMessageParamUnion {
	content := strings.TrimSpace(msg.Text())
	union := openai.AssistantMessage(content)

	// Add tool_calls if present
	if len(msg.ToolCalls) > 0 && union.OfAssistant != nil {
		union.OfAssistant.ToolCalls = make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			union.OfAssistant.ToolCalls = append(union.OfAssistant.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID: tc.ID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				},
			})
		}
	}

	applyName(&union, msg.Name)
	return union
}

func buildToolMessage(msg models.ChatMessage) openai.ChatCompletionMessageParamUnion {
	return openai.ToolMessage(msg.ToolCallID, msg.Text())
}

func buildUserMessage(idx int, msg models.ChatMessage) (openai.ChatCompletionMessageParamUnion, error) {
	var union openai.ChatCompletionMessageParamUnion
	if len(msg.ContentParts) > 0 {
		parts, err := convertRichContentParts(msg.ContentParts)
		if err != nil {
			return union, fmt.Errorf("message %d: %w", idx, err)
		}
		union = openai.UserMessage(parts)
	} else {
		union = openai.UserMessage(msg.Text())
	}
	applyName(&union, msg.Name)
	return union, nil
}

func convertTextParts(parts []models.MessageContentPart) ([]openai.ChatCompletionContentPartTextParam, error) {
	result := make([]openai.ChatCompletionContentPartTextParam, 0, len(parts))
	for i, part := range parts {
		if !part.IsTextual() {
			return nil, fmt.Errorf("content part %d must be text for this role", i)
		}
		text := strings.TrimSpace(part.Text)
		if text == "" {
			continue
		}
		result = append(result, openai.ChatCompletionContentPartTextParam{
			Text: text,
			Type: "text",
		})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no textual content provided")
	}
	return result, nil
}

func convertRichContentParts(parts []models.MessageContentPart) ([]openai.ChatCompletionContentPartUnionParam, error) {
	result := make([]openai.ChatCompletionContentPartUnionParam, 0, len(parts))
	for i, part := range parts {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "", models.MessageContentPartTypeText, models.MessageContentPartTypeInputText, models.MessageContentPartTypeOutputText:
			text := strings.TrimSpace(part.Text)
			if text == "" {
				continue
			}
			result = append(result, openai.TextContentPart(text))
		case models.MessageContentPartTypeImageURL:
			if part.ImageURL == nil || strings.TrimSpace(part.ImageURL.URL) == "" {
				return nil, fmt.Errorf("content part %d missing image_url", i)
			}
			img := openai.ChatCompletionContentPartImageImageURLParam{URL: strings.TrimSpace(part.ImageURL.URL)}
			if detail := strings.TrimSpace(part.ImageURL.Detail); detail != "" {
				img.Detail = detail
			}
			result = append(result, openai.ImageContentPart(img))
		case models.MessageContentPartTypeImageFile:
			if part.ImageFile == nil {
				return nil, fmt.Errorf("content part %d missing file metadata", i)
			}
			fileParam := openai.ChatCompletionContentPartFileFileParam{}
			if id := strings.TrimSpace(part.ImageFile.FileID); id != "" {
				fileParam.FileID = param.NewOpt(id)
			} else if data := strings.TrimSpace(part.ImageFile.FileData); data != "" {
				fileParam.FileData = param.NewOpt(data)
			} else {
				return nil, fmt.Errorf("content part %d requires file_id or file_data", i)
			}
			result = append(result, openai.FileContentPart(fileParam))
		case models.MessageContentPartTypeInputAudio:
			if part.InputAudio == nil || strings.TrimSpace(part.InputAudio.Data) == "" {
				return nil, fmt.Errorf("content part %d missing audio payload", i)
			}
			audio := openai.ChatCompletionContentPartInputAudioInputAudioParam{
				Data:   strings.TrimSpace(part.InputAudio.Data),
				Format: strings.TrimSpace(part.InputAudio.Format),
			}
			if audio.Format == "" {
				return nil, fmt.Errorf("content part %d missing audio format", i)
			}
			result = append(result, openai.InputAudioContentPart(audio))
		default:
			return nil, fmt.Errorf("unsupported content part type %q", part.Type)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no usable content parts provided")
	}
	return result, nil
}

func applyName(union *openai.ChatCompletionMessageParamUnion, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	switch {
	case union.OfUser != nil:
		union.OfUser.Name = param.NewOpt(name)
	case union.OfAssistant != nil:
		union.OfAssistant.Name = param.NewOpt(name)
	case union.OfSystem != nil:
		union.OfSystem.Name = param.NewOpt(name)
	case union.OfDeveloper != nil:
		union.OfDeveloper.Name = param.NewOpt(name)
	}
}
