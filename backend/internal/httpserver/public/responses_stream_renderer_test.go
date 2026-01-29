package public

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

// parseSSEEvents extracts events from SSE output.
func parseSSEEvents(data string) []sseEvent {
	var events []sseEvent
	lines := strings.Split(data, "\n")
	var current sseEvent
	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			current.Event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			current.Data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && current.Event != "" {
			events = append(events, current)
			current = sseEvent{}
		} else if line == "" && current.Data == "[DONE]" {
			events = append(events, sseEvent{Event: "[DONE]", Data: "[DONE]"})
			current = sseEvent{}
		}
	}
	return events
}

type sseEvent struct {
	Event string
	Data  string
}

func (e sseEvent) payload() map[string]any {
	var m map[string]any
	_ = json.Unmarshal([]byte(e.Data), &m)
	return m
}

func TestStreamRenderer_TextOnly(t *testing.T) {
	r := newResponsesStreamRenderer("gpt-4", openAIResponseOptions{}).(*responsesStreamRenderer)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	chunk1 := models.ChatChunk{
		ID:      "resp-1",
		Created: time.Now(),
		Choices: []models.ChunkDelta{
			{Index: 0, Delta: models.ChatMessage{Role: "assistant"}},
		},
	}
	if err := r.HandleChunk(chunk1, w); err != nil {
		t.Fatal(err)
	}

	chunk2 := models.ChatChunk{
		ID: "resp-1",
		Choices: []models.ChunkDelta{
			{Index: 0, Delta: models.ChatMessage{Content: "Hello"}},
		},
	}
	if err := r.HandleChunk(chunk2, w); err != nil {
		t.Fatal(err)
	}

	chunk3 := models.ChatChunk{
		ID: "resp-1",
		Choices: []models.ChunkDelta{
			{Index: 0, Delta: models.ChatMessage{Content: " world"}, FinishReason: "stop"},
		},
	}
	if err := r.HandleChunk(chunk3, w); err != nil {
		t.Fatal(err)
	}

	usage := &models.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}
	if err := r.Done(w, usage); err != nil {
		t.Fatal(err)
	}
	w.Flush()

	events := parseSSEEvents(buf.String())
	eventTypes := make([]string, len(events))
	for i, e := range events {
		eventTypes[i] = e.Event
	}

	expected := []string{
		"response.created",
		"response.in_progress",
		"response.output_item.added",
		"response.content_part.added",
		"response.output_text.delta",
		"response.output_text.delta",
		"response.output_text.done",
		"response.content_part.done",
		"response.output_item.done",
		"response.completed",
		"[DONE]",
	}
	if len(eventTypes) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(eventTypes), eventTypes)
	}
	for i, exp := range expected {
		if eventTypes[i] != exp {
			t.Errorf("event[%d]: expected %q, got %q", i, exp, eventTypes[i])
		}
	}
}

func TestStreamRenderer_ToolCalls(t *testing.T) {
	idx0 := 0
	r := newResponsesStreamRenderer("gpt-4", openAIResponseOptions{}).(*responsesStreamRenderer)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	// Init chunk
	chunk1 := models.ChatChunk{
		ID:      "resp-tc",
		Created: time.Now(),
		Choices: []models.ChunkDelta{
			{
				Index: 0,
				Delta: models.ChatMessage{
					Role: "assistant",
					ToolCalls: []models.ToolCall{
						{Index: &idx0, ID: "call-1", Type: "function", Function: models.ToolCallFunction{Name: "get_weather"}},
					},
				},
			},
		},
	}
	if err := r.HandleChunk(chunk1, w); err != nil {
		t.Fatal(err)
	}

	// Arguments delta
	chunk2 := models.ChatChunk{
		ID: "resp-tc",
		Choices: []models.ChunkDelta{
			{
				Index: 0,
				Delta: models.ChatMessage{
					ToolCalls: []models.ToolCall{
						{Index: &idx0, Function: models.ToolCallFunction{Arguments: `{"city":"NYC"}`}},
					},
				},
			},
		},
	}
	if err := r.HandleChunk(chunk2, w); err != nil {
		t.Fatal(err)
	}

	// Finish
	chunk3 := models.ChatChunk{
		ID: "resp-tc",
		Choices: []models.ChunkDelta{
			{Index: 0, Delta: models.ChatMessage{}, FinishReason: "tool_calls"},
		},
	}
	if err := r.HandleChunk(chunk3, w); err != nil {
		t.Fatal(err)
	}

	if err := r.Done(w, nil); err != nil {
		t.Fatal(err)
	}
	w.Flush()

	events := parseSSEEvents(buf.String())
	eventTypes := make([]string, len(events))
	for i, e := range events {
		eventTypes[i] = e.Event
	}

	// Should have: created, in_progress, output_item.added (fc), arguments.delta, arguments.done, output_item.done (fc), completed
	hasAdded := false
	hasDelta := false
	hasDone := false
	for _, et := range eventTypes {
		switch et {
		case "response.output_item.added":
			hasAdded = true
		case "response.function_call_arguments.delta":
			hasDelta = true
		case "response.function_call_arguments.done":
			hasDone = true
		}
	}
	if !hasAdded || !hasDelta || !hasDone {
		t.Errorf("missing expected tool call events: added=%v delta=%v done=%v; events: %v", hasAdded, hasDelta, hasDone, eventTypes)
	}
}

func TestStreamRenderer_ReasoningPlusText(t *testing.T) {
	r := newResponsesStreamRenderer("o3", openAIResponseOptions{}).(*responsesStreamRenderer)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	// Reasoning delta
	chunk1 := models.ChatChunk{
		ID:      "resp-rs",
		Created: time.Now(),
		Choices: []models.ChunkDelta{
			{Index: 0, Delta: models.ChatMessage{Role: "assistant", Reasoning: "Let me think..."}},
		},
	}
	if err := r.HandleChunk(chunk1, w); err != nil {
		t.Fatal(err)
	}

	// Text delta
	chunk2 := models.ChatChunk{
		ID: "resp-rs",
		Choices: []models.ChunkDelta{
			{Index: 0, Delta: models.ChatMessage{Content: "The answer is 42"}},
		},
	}
	if err := r.HandleChunk(chunk2, w); err != nil {
		t.Fatal(err)
	}

	// Finish
	chunk3 := models.ChatChunk{
		ID: "resp-rs",
		Choices: []models.ChunkDelta{
			{Index: 0, Delta: models.ChatMessage{}, FinishReason: "stop"},
		},
	}
	if err := r.HandleChunk(chunk3, w); err != nil {
		t.Fatal(err)
	}

	if err := r.Done(w, nil); err != nil {
		t.Fatal(err)
	}
	w.Flush()

	events := parseSSEEvents(buf.String())
	eventTypes := make([]string, len(events))
	for i, e := range events {
		eventTypes[i] = e.Event
	}

	// Should include reasoning events before text events
	hasReasoningAdded := false
	hasReasoningDelta := false
	hasReasoningDone := false
	hasTextDelta := false
	for _, et := range eventTypes {
		switch et {
		case "response.reasoning_summary_part.added":
			hasReasoningAdded = true
		case "response.reasoning_summary_text.delta":
			hasReasoningDelta = true
		case "response.reasoning_summary_text.done":
			hasReasoningDone = true
		case "response.output_text.delta":
			hasTextDelta = true
		}
	}
	if !hasReasoningAdded || !hasReasoningDelta || !hasReasoningDone {
		t.Errorf("missing reasoning events: added=%v delta=%v done=%v; events: %v", hasReasoningAdded, hasReasoningDelta, hasReasoningDone, eventTypes)
	}
	if !hasTextDelta {
		t.Error("missing text delta event")
	}
}

func TestStreamRenderer_OutputIndexOrdering(t *testing.T) {
	// Verify that output indexes are sequential: reasoning(0) → tool_call(1) → message(2)
	idx0 := 0
	r := newResponsesStreamRenderer("o3", openAIResponseOptions{}).(*responsesStreamRenderer)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	// Reasoning
	chunk := models.ChatChunk{
		ID:      "resp-ord",
		Created: time.Now(),
		Choices: []models.ChunkDelta{
			{Index: 0, Delta: models.ChatMessage{Role: "assistant", Reasoning: "thinking"}},
		},
	}
	_ = r.HandleChunk(chunk, w)

	// Tool call
	chunk2 := models.ChatChunk{
		ID: "resp-ord",
		Choices: []models.ChunkDelta{
			{
				Index: 0,
				Delta: models.ChatMessage{
					ToolCalls: []models.ToolCall{
						{Index: &idx0, ID: "call-1", Type: "function", Function: models.ToolCallFunction{Name: "search", Arguments: "{}"}},
					},
				},
			},
		},
	}
	_ = r.HandleChunk(chunk2, w)

	// Text
	chunk3 := models.ChatChunk{
		ID: "resp-ord",
		Choices: []models.ChunkDelta{
			{Index: 0, Delta: models.ChatMessage{Content: "Result"}},
		},
	}
	_ = r.HandleChunk(chunk3, w)

	// Finish
	chunk4 := models.ChatChunk{
		ID: "resp-ord",
		Choices: []models.ChunkDelta{
			{Index: 0, Delta: models.ChatMessage{}, FinishReason: "stop"},
		},
	}
	_ = r.HandleChunk(chunk4, w)
	_ = r.Done(w, nil)
	w.Flush()

	// Check output_index values in output_item.added events only
	events := parseSSEEvents(buf.String())
	addedIndexes := []int{}
	for _, e := range events {
		if e.Event == "response.output_item.added" {
			p := e.payload()
			if idx, ok := p["output_index"].(float64); ok {
				addedIndexes = append(addedIndexes, int(idx))
			}
		}
	}

	// reasoning(0) → tool_call(1) → message(2)
	if len(addedIndexes) != 3 {
		t.Fatalf("expected 3 output_item.added events, got %d: %v", len(addedIndexes), addedIndexes)
	}
	// Indexes should be strictly increasing
	for i := 1; i < len(addedIndexes); i++ {
		if addedIndexes[i] <= addedIndexes[i-1] {
			t.Errorf("output indexes not strictly increasing: %v", addedIndexes)
			break
		}
	}
}
