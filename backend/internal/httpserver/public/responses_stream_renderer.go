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
	alias    string
	options  openAIResponseOptions
	response string
	created  time.Time
	sequence int
	items    map[int]*responseStreamItem
}

type responseStreamItem struct {
	ItemID       string
	Role         string
	OutputIndex  int
	ContentIndex int
	Text         strings.Builder
	FinishReason string
}

func newResponsesStreamRenderer(alias string, opts openAIResponseOptions) streamRenderer {
	return &responsesStreamRenderer{
		alias:   alias,
		options: opts,
		items:   make(map[int]*responseStreamItem),
	}
}

func (r *responsesStreamRenderer) ContentType() string { return "text/event-stream" }

func (r *responsesStreamRenderer) Init(*bufio.Writer) error { return nil }

func (r *responsesStreamRenderer) HandleChunk(chunk models.ChatChunk, w *bufio.Writer) error {
	if err := r.ensureResponseInitialized(chunk, w); err != nil {
		return err
	}
	for _, choice := range chunk.Choices {
		item, err := r.ensureItem(chunk, choice, w)
		if err != nil {
			return err
		}
		delta := choice.Delta.Text()
		if delta != "" {
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
		if finish := strings.TrimSpace(choice.FinishReason); finish != "" {
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
					"content": []openAIResponsesOutputContent{
						{
							Type: "output_text",
							Text: text,
						},
					},
				},
			}); err != nil {
				return err
			}
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
	resp := models.ChatResponse{
		ID:      r.response,
		Created: r.created,
		Model:   r.alias,
	}
	if usage != nil {
		resp.Usage = *usage
	}
	choices := make([]models.ChatChoice, 0, len(r.items))
	for _, idx := range r.sortedItemIndexes() {
		item := r.items[idx]
		choices = append(choices, models.ChatChoice{
			Index: idx,
			Message: models.ChatMessage{
				Role:    item.Role,
				Content: item.Text.String(),
			},
			FinishReason: item.FinishReason,
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
	item := &responseStreamItem{
		ItemID:       fmt.Sprintf("%s-%d", chunk.ID, choice.Index),
		Role:         role,
		OutputIndex:  choice.Index,
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
