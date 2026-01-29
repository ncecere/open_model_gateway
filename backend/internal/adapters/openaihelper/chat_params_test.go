package openaihelper

import (
	"encoding/json"
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

func TestBuildChatParams_SimpleTextMessages(t *testing.T) {
	req := models.ChatRequest{
		Model: "gpt-4",
		Messages: []models.ChatMessage{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
		},
	}
	params, err := BuildChatParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(params.Messages))
	}
}

func TestBuildChatParams_WithTemperature(t *testing.T) {
	temp := float32(0.7)
	req := models.ChatRequest{
		Model:       "gpt-4",
		Messages:    []models.ChatMessage{{Role: "user", Content: "Hi"}},
		Temperature: &temp,
	}
	params, err := BuildChatParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(params.Messages))
	}
}

func TestBuildChatParams_WithTools(t *testing.T) {
	req := models.ChatRequest{
		Model:    "gpt-4",
		Messages: []models.ChatMessage{{Role: "user", Content: "What's the weather?"}},
		Tools: []models.Tool{
			{
				Type: "function",
				Function: models.ToolFunction{
					Name:        "get_weather",
					Description: "Get weather",
					Parameters:  json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
				},
			},
		},
	}
	params, err := BuildChatParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(params.Tools))
	}
}

func TestBuildChatParams_ToolChoiceString(t *testing.T) {
	req := models.ChatRequest{
		Model:      "gpt-4",
		Messages:   []models.ChatMessage{{Role: "user", Content: "test"}},
		ToolChoice: json.RawMessage(`"auto"`),
	}
	_, err := BuildChatParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildChatParams_ToolChoiceObject(t *testing.T) {
	req := models.ChatRequest{
		Model:      "gpt-4",
		Messages:   []models.ChatMessage{{Role: "user", Content: "test"}},
		ToolChoice: json.RawMessage(`{"type":"function","function":{"name":"get_weather"}}`),
	}
	_, err := BuildChatParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildChatParams_DeveloperRole(t *testing.T) {
	req := models.ChatRequest{
		Model: "gpt-4",
		Messages: []models.ChatMessage{
			{Role: "developer", Content: "Be concise"},
			{Role: "user", Content: "Hello"},
		},
	}
	params, err := BuildChatParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(params.Messages))
	}
}

func TestBuildChatParams_AssistantWithToolCalls(t *testing.T) {
	req := models.ChatRequest{
		Model: "gpt-4",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "test"},
			{
				Role:    "assistant",
				Content: "",
				ToolCalls: []models.ToolCall{
					{
						ID:   "call_123",
						Type: "function",
						Function: models.ToolCallFunction{
							Name:      "get_weather",
							Arguments: `{"city":"NYC"}`,
						},
					},
				},
			},
			{Role: "tool", ToolCallID: "call_123", Content: "Sunny, 72F"},
		},
	}
	params, err := BuildChatParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(params.Messages))
	}
}

func TestBuildChatParams_StopSequences(t *testing.T) {
	req := models.ChatRequest{
		Model:    "gpt-4",
		Messages: []models.ChatMessage{{Role: "user", Content: "test"}},
		Stop:     []string{"END"},
	}
	_, err := BuildChatParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildChatParams_MultipleStopSequences(t *testing.T) {
	req := models.ChatRequest{
		Model:    "gpt-4",
		Messages: []models.ChatMessage{{Role: "user", Content: "test"}},
		Stop:     []string{"END", "STOP", "DONE"},
	}
	_, err := BuildChatParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildChatParams_Penalties(t *testing.T) {
	pp := float32(0.5)
	fp := float32(0.3)
	req := models.ChatRequest{
		Model:            "gpt-4",
		Messages:         []models.ChatMessage{{Role: "user", Content: "test"}},
		PresencePenalty:  &pp,
		FrequencyPenalty: &fp,
	}
	_, err := BuildChatParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
