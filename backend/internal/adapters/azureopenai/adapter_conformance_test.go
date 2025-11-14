package azureopenai

import (
	"testing"

	"github.com/openai/openai-go/v3"

	"github.com/ncecere/open_model_gateway/backend/internal/providers/fixtures"
)

func TestConvertChatResponseFixture(t *testing.T) {
	var resp openai.ChatCompletion
	if err := fixtures.Load("azure_chat_completion.json", &resp); err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	converted := convertChatResponse(resp)
	if converted.ID != "chatcmpl-fixture-sync" {
		t.Fatalf("unexpected id: %s", converted.ID)
	}
	if converted.Model != "gpt-5-mini" {
		t.Fatalf("unexpected model: %s", converted.Model)
	}
	if len(converted.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(converted.Choices))
	}
	if converted.Choices[0].Message.Content != "Fixture response" {
		t.Fatalf("unexpected content: %s", converted.Choices[0].Message.Content)
	}
	if converted.Usage.TotalTokens != 170 || converted.Usage.PromptTokens != 42 || converted.Usage.CompletionTokens != 128 {
		t.Fatalf("usage mismatch: %+v", converted.Usage)
	}
}

func TestConvertChatChunkFixture(t *testing.T) {
	var chunk openai.ChatCompletionChunk
	if err := fixtures.Load("azure_stream_chunk.json", &chunk); err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	converted := convertChatChunk(chunk)
	if converted.ID != "chatcmpl-fixture-stream" {
		t.Fatalf("unexpected id: %s", converted.ID)
	}
	if len(converted.Choices) != 1 {
		t.Fatalf("expected chunk choice, got %d", len(converted.Choices))
	}
	if converted.Choices[0].Delta.Content != "Hello from fixture" {
		t.Fatalf("unexpected delta: %s", converted.Choices[0].Delta.Content)
	}
	if converted.Usage == nil {
		t.Fatalf("expected usage payload")
	}
	if converted.Usage.TotalTokens != 26 || converted.Usage.PromptTokens != 21 || converted.Usage.CompletionTokens != 5 {
		t.Fatalf("usage mismatch: %+v", converted.Usage)
	}
}
