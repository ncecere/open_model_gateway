package models

import (
	"encoding/json"
	"strings"
)

// Tool defines a callable function the model can invoke.
type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction describes the function signature for tool calling.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"` // JSON Schema
	Strict      *bool           `json:"strict,omitempty"`
}

// ToolCall represents a tool invocation in an assistant message.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
	Index    *int             `json:"index,omitempty"` // For streaming deltas
}

// ToolCallFunction contains the function name and arguments for a tool call.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolChoiceOption represents the tool_choice parameter which can be:
// - a string: "none", "auto", "required"
// - an object: { "type": "function", "function": { "name": "..." } }
type ToolChoiceOption struct {
	Mode     string              `json:"-"` // "none", "auto", "required" or empty for specific
	Type     string              `json:"type,omitempty"`
	Function *ToolChoiceFunction `json:"function,omitempty"`
}

// ToolChoiceFunction specifies a particular function to call.
type ToolChoiceFunction struct {
	Name string `json:"name"`
}

// ParseToolChoice parses the tool_choice parameter from JSON.
// It handles both string values ("none", "auto", "required") and object values.
func ParseToolChoice(raw json.RawMessage) (*ToolChoiceOption, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	// Try string first
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		mode := strings.ToLower(strings.TrimSpace(str))
		return &ToolChoiceOption{Mode: mode}, nil
	}

	// Try object
	var obj ToolChoiceOption
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// IsNone returns true if tool_choice is "none".
func (t *ToolChoiceOption) IsNone() bool {
	return t != nil && t.Mode == "none"
}

// IsAuto returns true if tool_choice is "auto" or not specified.
func (t *ToolChoiceOption) IsAuto() bool {
	return t == nil || t.Mode == "" || t.Mode == "auto"
}

// IsRequired returns true if tool_choice is "required".
func (t *ToolChoiceOption) IsRequired() bool {
	return t != nil && t.Mode == "required"
}

// IsSpecific returns true if tool_choice specifies a particular function.
func (t *ToolChoiceOption) IsSpecific() bool {
	return t != nil && t.Function != nil && t.Function.Name != ""
}

// MarshalJSON implements custom JSON marshaling for ToolChoiceOption.
func (t ToolChoiceOption) MarshalJSON() ([]byte, error) {
	if t.Mode != "" && t.Function == nil {
		return json.Marshal(t.Mode)
	}
	type alias ToolChoiceOption
	return json.Marshal(alias(t))
}

// ConvertLegacyFunctions converts deprecated functions/function_call to modern tools format.
func ConvertLegacyFunctions(functions []ToolFunction, functionCall json.RawMessage) ([]Tool, *ToolChoiceOption) {
	if len(functions) == 0 {
		return nil, nil
	}

	tools := make([]Tool, 0, len(functions))
	for _, fn := range functions {
		tools = append(tools, Tool{
			Type:     "function",
			Function: fn,
		})
	}

	var toolChoice *ToolChoiceOption
	if len(functionCall) > 0 {
		// function_call can be "none", "auto", or {"name": "function_name"}
		var str string
		if err := json.Unmarshal(functionCall, &str); err == nil {
			mode := strings.ToLower(strings.TrimSpace(str))
			toolChoice = &ToolChoiceOption{Mode: mode}
		} else {
			var obj struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(functionCall, &obj); err == nil && obj.Name != "" {
				toolChoice = &ToolChoiceOption{
					Type:     "function",
					Function: &ToolChoiceFunction{Name: obj.Name},
				}
			}
		}
	}

	return tools, toolChoice
}
