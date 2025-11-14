package vertex

import (
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/providers/fixtures"
)

func TestConvertChatResponseFixture(t *testing.T) {
	var resp vertexGenerateResponse
	if err := fixtures.Load("vertex_chat_response.json", &resp); err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	converted, err := convertChatResponse(resp, "gemini-pro")
	if err != nil {
		t.Fatalf("convert response: %v", err)
	}
	if converted.Model != "gemini-pro" {
		t.Fatalf("unexpected model %s", converted.Model)
	}
	if len(converted.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(converted.Choices))
	}
	expected := "Hello adventurer,\nWhat quest shall we begin?"
	if converted.Choices[0].Message.Content != expected {
		t.Fatalf("unexpected content: %s", converted.Choices[0].Message.Content)
	}
	if converted.Choices[0].FinishReason != "stop" {
		t.Fatalf("finish reason not normalized: %s", converted.Choices[0].FinishReason)
	}
	if converted.Usage.TotalTokens != 18 || converted.Usage.PromptTokens != 11 || converted.Usage.CompletionTokens != 7 {
		t.Fatalf("usage mismatch: %+v", converted.Usage)
	}
}

func TestConvertEmbeddingsResponseFixture(t *testing.T) {
	var resp vertexPredictResponse
	if err := fixtures.Load("vertex_embed_response.json", &resp); err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	converted, err := convertEmbeddingsResponse(resp, "textembedding")
	if err != nil {
		t.Fatalf("convert embeddings: %v", err)
	}
	if converted.Model != "textembedding" {
		t.Fatalf("unexpected model: %s", converted.Model)
	}
	if len(converted.Embeddings) != 1 {
		t.Fatalf("expected single embedding, got %d", len(converted.Embeddings))
	}
	if len(converted.Embeddings[0].Vector) != 3 {
		t.Fatalf("unexpected vector length: %d", len(converted.Embeddings[0].Vector))
	}
	if converted.Usage.TotalTokens != 42 {
		t.Fatalf("usage total mismatch: %+v", converted.Usage)
	}
}
