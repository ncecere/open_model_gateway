package public

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

type responsesStreamRenderer struct {
	alias      string
	options    openAIResponseOptions
	response   string
	created    time.Time
	sequence   int
	items      map[int]*responseStreamItem
	toolCalls  map[int]*toolCallStream // keyed by tool call index
	nextOutput int                     // next output index for tool call items
	reasoning  *reasoningStreamItem    // tracks reasoning output during streaming
}

type responseStreamItem struct {
	ItemID       string
	Role         string
	OutputIndex  int
	ContentIndex int
	Text         strings.Builder
	FinishReason string
	IsToolCall   bool // true if this item is a function_call, not a message
}

// reasoningStreamItem tracks reasoning content during streaming.
type reasoningStreamItem struct {
	ItemID      string
	OutputIndex int
	Text        strings.Builder
	Started     bool
}

// toolCallStream tracks incremental tool call arguments during streaming.
type toolCallStream struct {
	ItemID      string
	OutputIndex int
	CallID      string
	Name        string
	Arguments   strings.Builder
	Done        bool
}

func newResponsesStreamRenderer(alias string, opts openAIResponseOptions) streamRenderer {
	return &responsesStreamRenderer{
		alias:     alias,
		options:   opts,
		items:     make(map[int]*responseStreamItem),
		toolCalls: make(map[int]*toolCallStream),
	}
}

func (r *responsesStreamRenderer) ContentType() string { return "text/event-stream" }

func (r *responsesStreamRenderer) Init(*bufio.Writer) error { return nil }

func (r *responsesStreamRenderer) HandleChunk(chunk models.ChatChunk, w *bufio.Writer) error {
	if err := r.ensureResponseInitialized(chunk, w); err != nil {
		return err
	}
	for _, choice := range chunk.Choices {
		// Handle reasoning content deltas
		reasoningDelta := strings.TrimSpace(choice.Delta.Reasoning)
		if reasoningDelta == "" {
			reasoningDelta = strings.TrimSpace(choice.Delta.ReasoningContent)
		}
		if reasoningDelta != "" {
			if err := r.handleReasoningDelta(chunk, choice, reasoningDelta, w); err != nil {
				return err
			}
		}

		// Handle tool call deltas
		if len(choice.Delta.ToolCalls) > 0 {
			if err := r.handleToolCallDeltas(chunk, choice, w); err != nil {
				return err
			}
		}

		// Handle text deltas
		delta := choice.Delta.Text()
		if delta != "" {
			item, err := r.ensureItem(chunk, choice, w)
			if err != nil {
				return err
			}
			item.Text.WriteString(delta)
			if err := r.emitEvent(w, "response.output_text.delta", map[string]any{
				"item_id":       item.ItemID,
				"output_index":  item.OutputIndex,
				"content_index": item.ContentIndex,
				"delta":         delta,
			}); err != nil {
				return err
			}
		}

		// Handle finish
		if finish := strings.TrimSpace(choice.FinishReason); finish != "" {
			// Finalize reasoning if open
			if err := r.finalizeReasoning(w); err != nil {
				return err
			}
			// Finalize any open tool calls
			if err := r.finalizeToolCalls(w); err != nil {
				return err
			}
			// Finalize text message item if we have one
			if item, ok := r.items[choice.Index]; ok && !item.IsToolCall {
				item.FinishReason = finish
				text := item.Text.String()
				if err := r.emitEvent(w, "response.output_text.done", map[string]any{
					"item_id":       item.ItemID,
					"output_index":  item.OutputIndex,
					"content_index": item.ContentIndex,
					"text":          text,
				}); err != nil {
					return err
				}
				if err := r.emitEvent(w, "response.content_part.done", map[string]any{
					"item_id":       item.ItemID,
					"output_index":  item.OutputIndex,
					"content_index": item.ContentIndex,
					"part": map[string]any{
						"type":        "output_text",
						"text":        text,
						"annotations": []any{},
					},
				}); err != nil {
					return err
				}
				if err := r.emitEvent(w, "response.output_item.done", map[string]any{
					"output_index": item.OutputIndex,
					"item": map[string]any{
						"id":     item.ItemID,
						"status": "completed",
						"type":   "message",
						"role":   item.Role,
						"content": []responsesOutputContent{
							{
								Type:        "output_text",
								Text:        text,
								Annotations: []interface{}{},
							},
						},
					},
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *responsesStreamRenderer) handleReasoningDelta(chunk models.ChatChunk, choice models.ChunkDelta, delta string, w *bufio.Writer) error {
	if r.reasoning == nil {
		// First reasoning delta — create the item and emit events
		outputIdx := r.nextOutput
		r.nextOutput++
		itemID := fmt.Sprintf("%s-rs-%d", chunk.ID, choice.Index)
		r.reasoning = &reasoningStreamItem{
			ItemID:      itemID,
			OutputIndex: outputIdx,
			Started:     true,
		}
		if err := r.emitEvent(w, "response.output_item.added", map[string]any{
			"output_index": outputIdx,
			"item": map[string]any{
				"id":      itemID,
				"type":    "reasoning",
				"status":  "in_progress",
				"summary": []any{},
			},
		}); err != nil {
			return err
		}
		if err := r.emitEvent(w, "response.reasoning_summary_part.added", map[string]any{
			"item_id":       itemID,
			"output_index":  outputIdx,
			"content_index": 0,
			"part": map[string]any{
				"type": "summary_text",
				"text": "",
			},
		}); err != nil {
			return err
		}
	}
	r.reasoning.Text.WriteString(delta)
	return r.emitEvent(w, "response.reasoning_summary_text.delta", map[string]any{
		"item_id":       r.reasoning.ItemID,
		"output_index":  r.reasoning.OutputIndex,
		"content_index": 0,
		"delta":         delta,
	})
}

func (r *responsesStreamRenderer) finalizeReasoning(w *bufio.Writer) error {
	if r.reasoning == nil {
		return nil
	}
	rs := r.reasoning
	text := rs.Text.String()
	if err := r.emitEvent(w, "response.reasoning_summary_text.done", map[string]any{
		"item_id":       rs.ItemID,
		"output_index":  rs.OutputIndex,
		"content_index": 0,
		"text":          text,
	}); err != nil {
		return err
	}
	if err := r.emitEvent(w, "response.reasoning_summary_part.done", map[string]any{
		"item_id":       rs.ItemID,
		"output_index":  rs.OutputIndex,
		"content_index": 0,
		"part": map[string]any{
			"type": "summary_text",
			"text": text,
		},
	}); err != nil {
		return err
	}
	return r.emitEvent(w, "response.output_item.done", map[string]any{
		"output_index": rs.OutputIndex,
		"item": map[string]any{
			"id":     rs.ItemID,
			"type":   "reasoning",
			"status": "completed",
			"summary": []map[string]any{
				{
					"type": "summary_text",
					"text": text,
				},
			},
		},
	})
}

func (r *responsesStreamRenderer) handleToolCallDeltas(chunk models.ChatChunk, choice models.ChunkDelta, w *bufio.Writer) error {
	for _, tc := range choice.Delta.ToolCalls {
		idx := 0
		if tc.Index != nil {
			idx = *tc.Index
		}
		tcs, exists := r.toolCalls[idx]
		if !exists {
			// New tool call — assign a deterministic output index
			outputIdx := r.nextOutput
			r.nextOutput++
			itemID := fmt.Sprintf("%s-fc-%d", chunk.ID, idx)
			tcs = &toolCallStream{
				ItemID:      itemID,
				OutputIndex: outputIdx,
				CallID:      tc.ID,
				Name:        tc.Function.Name,
			}
			r.toolCalls[idx] = tcs
			if err := r.emitEvent(w, "response.output_item.added", map[string]any{
				"output_index": tcs.OutputIndex,
				"item": map[string]any{
					"id":        tcs.ItemID,
					"type":      "function_call",
					"status":    "in_progress",
					"call_id":   tcs.CallID,
					"name":      tcs.Name,
					"arguments": "",
				},
			}); err != nil {
				return err
			}
		} else {
			// Update call_id/name if provider sends them incrementally
			if tc.ID != "" {
				tcs.CallID = tc.ID
			}
			if tc.Function.Name != "" {
				tcs.Name += tc.Function.Name
			}
		}

		// Emit arguments delta
		if tc.Function.Arguments != "" {
			tcs.Arguments.WriteString(tc.Function.Arguments)
			if err := r.emitEvent(w, "response.function_call_arguments.delta", map[string]any{
				"item_id":      tcs.ItemID,
				"output_index": tcs.OutputIndex,
				"delta":        tc.Function.Arguments,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *responsesStreamRenderer) finalizeToolCalls(w *bufio.Writer) error {
	for _, tcs := range r.toolCalls {
		if tcs.Done {
			continue
		}
		tcs.Done = true
		args := tcs.Arguments.String()
		if err := r.emitEvent(w, "response.function_call_arguments.done", map[string]any{
			"item_id":      tcs.ItemID,
			"output_index": tcs.OutputIndex,
			"arguments":    args,
		}); err != nil {
			return err
		}
		if err := r.emitEvent(w, "response.output_item.done", map[string]any{
			"output_index": tcs.OutputIndex,
			"item": map[string]any{
				"id":        tcs.ItemID,
				"type":      "function_call",
				"status":    "completed",
				"call_id":   tcs.CallID,
				"name":      tcs.Name,
				"arguments": args,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *responsesStreamRenderer) Done(w *bufio.Writer, usage *models.Usage) error {
	if r.response == "" {
		if _, err := w.WriteString("data: [DONE]\n\n"); err != nil {
			return err
		}
		return w.Flush()
	}

	// Build a synthetic ChatResponse that includes tool calls so
	// buildResponsesPayload produces function_call output items.
	resp := models.ChatResponse{
		ID:      r.response,
		Created: r.created,
		Model:   r.alias,
	}
	if usage != nil {
		resp.Usage = *usage
	}

	// Assemble message choices (text items)
	choices := make([]models.ChatChoice, 0, len(r.items))
	for _, idx := range r.sortedItemIndexes() {
		item := r.items[idx]
		msg := models.ChatMessage{
			Role:    item.Role,
			Content: item.Text.String(),
		}
		// Attach accumulated tool calls to the message so buildResponsesPayload
		// emits function_call output items.
		if len(r.toolCalls) > 0 {
			tcs := make([]models.ToolCall, 0, len(r.toolCalls))
			for _, tcIdx := range r.sortedToolCallIndexes() {
				tc := r.toolCalls[tcIdx]
				tcs = append(tcs, models.ToolCall{
					ID:   tc.CallID,
					Type: "function",
					Function: models.ToolCallFunction{
						Name:      tc.Name,
						Arguments: tc.Arguments.String(),
					},
				})
			}
			msg.ToolCalls = tcs
		}
		choices = append(choices, models.ChatChoice{
			Index:        idx,
			Message:      msg,
			FinishReason: item.FinishReason,
		})
	}

	// If there are tool calls but no text message item, create a synthetic choice
	if len(choices) == 0 && len(r.toolCalls) > 0 {
		tcs := make([]models.ToolCall, 0, len(r.toolCalls))
		for _, tcIdx := range r.sortedToolCallIndexes() {
			tc := r.toolCalls[tcIdx]
			tcs = append(tcs, models.ToolCall{
				ID:   tc.CallID,
				Type: "function",
				Function: models.ToolCallFunction{
					Name:      tc.Name,
					Arguments: tc.Arguments.String(),
				},
			})
		}
		choices = append(choices, models.ChatChoice{
			Index: 0,
			Message: models.ChatMessage{
				Role:      "assistant",
				ToolCalls: tcs,
			},
			FinishReason: "tool_calls",
		})
	}

	resp.Choices = choices
	payload := buildResponsesPayload(resp, r.alias, r.options, "")
	if err := r.emitEvent(w, "response.completed", map[string]any{"response": payload}); err != nil {
		return err
	}
	if _, err := w.WriteString("data: [DONE]\n\n"); err != nil {
		return err
	}
	return w.Flush()
}

func (r *responsesStreamRenderer) sortedToolCallIndexes() []int {
	indexes := make([]int, 0, len(r.toolCalls))
	for idx := range r.toolCalls {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	return indexes
}

func (r *responsesStreamRenderer) ensureResponseInitialized(chunk models.ChatChunk, w *bufio.Writer) error {
	if r.response != "" {
		return nil
	}
	r.response = chunk.ID
	r.created = chunk.Created
	resp := models.ChatResponse{ID: r.response, Created: r.created, Model: r.alias}
	base := buildResponsesPayload(resp, r.alias, r.options, "in_progress")
	if err := r.emitEvent(w, "response.created", map[string]any{"response": base}); err != nil {
		return err
	}
	return r.emitEvent(w, "response.in_progress", map[string]any{"response": base})
}

func (r *responsesStreamRenderer) ensureItem(chunk models.ChatChunk, choice models.ChunkDelta, w *bufio.Writer) (*responseStreamItem, error) {
	if item, ok := r.items[choice.Index]; ok {
		return item, nil
	}
	role := strings.TrimSpace(choice.Delta.Role)
	if role == "" {
		role = "assistant"
	}
	outputIdx := r.nextOutput
	r.nextOutput++
	item := &responseStreamItem{
		ItemID:       fmt.Sprintf("%s-%d", chunk.ID, choice.Index),
		Role:         role,
		OutputIndex:  outputIdx,
		ContentIndex: 0,
	}
	r.items[choice.Index] = item
	if err := r.emitEvent(w, "response.output_item.added", map[string]any{
		"output_index": item.OutputIndex,
		"item": map[string]any{
			"id":      item.ItemID,
			"status":  "in_progress",
			"type":    "message",
			"role":    item.Role,
			"content": []any{},
		},
	}); err != nil {
		return nil, err
	}
	if err := r.emitEvent(w, "response.content_part.added", map[string]any{
		"item_id":       item.ItemID,
		"output_index":  item.OutputIndex,
		"content_index": item.ContentIndex,
		"part": map[string]any{
			"type":        "output_text",
			"text":        "",
			"annotations": []any{},
		},
	}); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *responsesStreamRenderer) emitEvent(w *bufio.Writer, event string, payload map[string]any) error {
	r.sequence++
	payload["type"] = event
	payload["sequence_number"] = r.sequence
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err = w.WriteString("event: " + event + "\n"); err != nil {
		return err
	}
	if _, err = w.WriteString("data: "); err != nil {
		return err
	}
	if _, err = w.Write(data); err != nil {
		return err
	}
	if _, err = w.WriteString("\n\n"); err != nil {
		return err
	}
	return nil
}

func (r *responsesStreamRenderer) sortedItemIndexes() []int {
	indexes := make([]int, 0, len(r.items))
	for idx := range r.items {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	return indexes
}
