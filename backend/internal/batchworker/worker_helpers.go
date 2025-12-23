package batchworker

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"strings"
)

func parseStop(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return []string{single}, nil
	}
	var multi []string
	if err := json.Unmarshal(raw, &multi); err == nil {
		return multi, nil
	}
	return nil, errors.New("invalid stop value")
}

func parseEmbeddingInput(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, errors.New("input is required")
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return []string{str}, nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, errors.New("input must be string or array of strings")
}

func parseBatchFileRefs(single json.RawMessage, multi json.RawMessage) ([]string, error) {
	refs := make([]string, 0)
	if len(single) > 0 && !isNullJSON(single) {
		var ref string
		if err := json.Unmarshal(single, &ref); err != nil {
			return nil, fmt.Errorf("file references must be strings or arrays of strings")
		}
		refs = append(refs, ref)
	}
	if len(multi) > 0 && !isNullJSON(multi) {
		var arr []string
		if err := json.Unmarshal(multi, &arr); err != nil {
			return nil, fmt.Errorf("file references must be strings or arrays of strings")
		}
		refs = append(refs, arr...)
	}
	clean := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref != "" {
			clean = append(clean, ref)
		}
	}
	return clean, nil
}

func isNullJSON(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return len(raw) == 0 || trimmed == "" || strings.EqualFold(trimmed, "null")
}

func decodeBatchRequest(item batchItem, expectedPath string) (batchRequest, []byte) {
	var req batchRequest
	if err := json.Unmarshal(item.Input, &req); err != nil {
		return batchRequest{}, encodeErrorPayload("invalid_batch_input", fmt.Sprintf("invalid JSON payload: %v", err))
	}
	method := strings.TrimSpace(req.Method)
	if method == "" {
		method = "POST"
	}
	if !strings.EqualFold(method, "POST") {
		return batchRequest{}, encodeErrorPayload("invalid_method", "batch entries must use POST")
	}
	path := strings.TrimSpace(req.URL)
	if path == "" {
		path = expectedPath
	}
	if path != expectedPath {
		return batchRequest{}, encodeErrorPayload("invalid_endpoint", "batch URL mismatch")
	}
	if len(req.Body) == 0 {
		return batchRequest{}, encodeErrorPayload("invalid_request_error", "body is required")
	}
	return req, nil
}

func errMessage(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}

func validateResponsesMetadata(meta map[string]string) error {
	if len(meta) > 16 {
		return fmt.Errorf("metadata cannot exceed 16 entries")
	}
	for k, v := range meta {
		key := strings.TrimSpace(k)
		if key == "" {
			return fmt.Errorf("metadata keys cannot be empty")
		}
		if len(key) > 64 {
			return fmt.Errorf("metadata key %q exceeds 64 characters", key)
		}
		if len(v) > 512 {
			return fmt.Errorf("metadata value for %q exceeds 512 characters", key)
		}
	}
	return nil
}

func buildResponsesMessages(instructions string, input json.RawMessage) ([]models.ChatMessage, error) {
	messages := make([]models.ChatMessage, 0, 1)
	if instr := strings.TrimSpace(instructions); instr != "" {
		messages = append(messages, models.ChatMessage{Role: "system", Content: instr})
	}
	inputMessages, err := parseResponsesInputItems(input)
	if err != nil {
		return nil, err
	}
	messages = append(messages, inputMessages...)
	return messages, nil
}

func parseResponsesInputItems(raw json.RawMessage) ([]models.ChatMessage, error) {
	if len(raw) == 0 {
		return nil, errors.New("input is required")
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []models.ChatMessage{{Role: "user", Content: text}}, nil
	}
	var items []chatMessage
	if err := json.Unmarshal(raw, &items); err == nil {
		if len(items) == 0 {
			return nil, errors.New("input array must contain at least one item")
		}
		messages := make([]models.ChatMessage, 0, len(items))
		for idx, item := range items {
			role := strings.ToLower(strings.TrimSpace(item.Role))
			if role == "" {
				role = "user"
			}
			textContent, parts, err := models.ParseMessageContent(item.Content)
			if err != nil {
				return nil, fmt.Errorf("invalid content for input item %d: %v", idx, err)
			}
			messages = append(messages, models.ChatMessage{
				Role:         role,
				Content:      textContent,
				ContentParts: parts,
			})
		}
		return messages, nil
	}
	return nil, errors.New("input must be a string or array of message objects")
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	var out pgtype.UUID
	if id == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	copy(out.Bytes[:], id[:])
	out.Valid = true
	return out
}

func fromPgUUID(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, fmt.Errorf("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}
