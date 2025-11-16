package openrouter

import (
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/providers/fixtures"
)

func TestConvertChatResponse(t *testing.T) {
	var payload chatCompletionResponse
	if err := fixtures.Load("openrouter_chat_completion.json", &payload); err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	resp := convertChatResponse(payload)
	if resp.ID != "chatcmpl-abc123" {
		t.Fatalf("expected chat id to match fixture, got %s", resp.ID)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "Hello from OpenRouter!" {
		t.Fatalf("unexpected chat choice conversion: %+v", resp.Choices)
	}
	if resp.Usage.TotalTokens != 20 {
		t.Fatalf("expected usage total tokens to equal 20, got %d", resp.Usage.TotalTokens)
	}
	if resp.Created.Unix() != 1700000000 {
		t.Fatalf("expected created timestamp to carry over, got %d", resp.Created.Unix())
	}
}

func TestConvertChatChunk(t *testing.T) {
	var payload chatCompletionChunk
	if err := fixtures.Load("openrouter_stream_chunk.json", &payload); err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	chunk := convertChatChunk(payload)
	if len(chunk.Choices) != 1 {
		t.Fatalf("expected single delta, got %d", len(chunk.Choices))
	}
	if chunk.Choices[0].Delta.Content != "Hello" {
		t.Fatalf("unexpected delta content %q", chunk.Choices[0].Delta.Content)
	}
	if chunk.Usage == nil || chunk.Usage.TotalTokens != 7 {
		t.Fatalf("expected usage chunk with total tokens, got %+v", chunk.Usage)
	}
}

func TestConvertEmbeddingsResponse(t *testing.T) {
	var payload embeddingResponse
	if err := fixtures.Load("openrouter_embeddings.json", &payload); err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	resp := convertEmbeddingsResponse(payload)
	if len(resp.Embeddings) != 1 {
		t.Fatalf("expected one embedding vector, got %d", len(resp.Embeddings))
	}
	if resp.Embeddings[0].Vector[0] != float32(0.1) {
		t.Fatalf("unexpected embedding contents: %+v", resp.Embeddings[0].Vector)
	}
	if resp.Usage.PromptTokens != 3 {
		t.Fatalf("expected prompt tokens to equal 3, got %d", resp.Usage.PromptTokens)
	}
}

func TestConvertModelList(t *testing.T) {
	var payload modelListResponse
	if err := fixtures.Load("openrouter_models.json", &payload); err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	models := convertModelList(payload)
	if len(models) != 1 {
		t.Fatalf("expected one model entry, got %d", len(models))
	}
	if models[0].ContextWindow != 32768 {
		t.Fatalf("unexpected context window %d", models[0].ContextWindow)
	}
}
