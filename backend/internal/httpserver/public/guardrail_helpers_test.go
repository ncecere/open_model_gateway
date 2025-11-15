package public

import (
	"testing"

	guardrails "github.com/ncecere/open_model_gateway/backend/internal/guardrails"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

func TestChatPromptText(t *testing.T) {
	messages := []models.ChatMessage{
		{Content: "hello"},
		{Content: "  "},
		{Content: "world"},
	}
	got := chatPromptText(messages)
	if got != "hello\nworld" {
		t.Fatalf("unexpected prompt text: %q", got)
	}
}

func TestGuardrailStreamMonitorBlocks(t *testing.T) {
	cfg := guardrails.Config{
		Enabled: true,
		Response: guardrails.ResponseConfig{
			BlockedKeywords: []string{"stop"},
		},
	}
	runtime := guardrailRuntime{
		cfg:       cfg,
		evaluator: guardrails.NewEvaluator(cfg),
	}
	monitor := newGuardrailStreamMonitor(runtime)
	allowChunk := models.ChatChunk{Choices: []models.ChunkDelta{{Delta: models.ChatMessage{Content: "hello"}}}}
	if _, blocked := monitor.Process(allowChunk); blocked {
		t.Fatal("expected first chunk to pass guardrails")
	}

	blockChunk := models.ChatChunk{Choices: []models.ChunkDelta{{
		Delta: models.ChatMessage{Content: " stop now"},
	}}}
	res, blocked := monitor.Process(blockChunk)
	if !blocked {
		t.Fatal("expected guardrail violation")
	}
	if !guardrailBlocked(res) {
		t.Fatalf("expected action block, got %v", res.Action)
	}
}
