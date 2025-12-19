package models

import (
	"strings"
	"time"
)

type ChatMessage struct {
	Role             string               `json:"role"`
	Content          string               `json:"content"`
	ContentParts     []MessageContentPart `json:"content_parts,omitempty"`
	Name             string               `json:"name,omitempty"`
	Reasoning        string               `json:"reasoning,omitempty"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float32      `json:"temperature,omitempty"`
	TopP        *float32      `json:"top_p,omitempty"`
	MaxTokens   *int32        `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
}

type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	ReasoningTokens  int32 `json:"reasoning_tokens,omitempty"`
	TotalTokens      int32 `json:"total_tokens"`
}

type ChatResponse struct {
	ID      string       `json:"id"`
	Created time.Time    `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   Usage        `json:"usage"`
}

type ChatChunk struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Created time.Time    `json:"created"`
	Choices []ChunkDelta `json:"choices"`
	Usage   *Usage       `json:"-"`
}

func (c ChatChunk) IsUsageOnly() bool {
	return len(c.Choices) == 0 && c.Usage != nil && (c.Usage.PromptTokens > 0 || c.Usage.CompletionTokens > 0 || c.Usage.TotalTokens > 0)
}

type ChunkDelta struct {
	Index        int         `json:"index"`
	Delta        ChatMessage `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

// Text returns a text-only representation of the chat message, preferring any
// structured content parts when available to maintain compatibility with legacy
// handlers that only accept strings.
func (m ChatMessage) Text() string {
	if len(m.ContentParts) > 0 {
		if text := TextFromContentParts(m.ContentParts); text != "" {
			return text
		}
	}
	if strings.TrimSpace(m.Content) == "" && strings.TrimSpace(m.ReasoningContent) != "" {
		return m.ReasoningContent
	}
	return m.Content
}

// CapabilityRequirements summarizes which multimodal inputs a chat request needs.
type CapabilityRequirements struct {
	NeedsImage bool
	NeedsAudio bool
	NeedsVideo bool
}

// HasRequirements reports whether the request needs any non-text inputs.
func (c CapabilityRequirements) HasRequirements() bool {
	return c.NeedsImage || c.NeedsAudio || c.NeedsVideo
}

// CapabilityRequirements aggregates the capabilities required by the chat request.
func (r ChatRequest) CapabilityRequirements() CapabilityRequirements {
	var req CapabilityRequirements
	for _, msg := range r.Messages {
		req.merge(msg.capabilityRequirements())
	}
	return req
}

func (c *CapabilityRequirements) merge(other CapabilityRequirements) {
	c.NeedsImage = c.NeedsImage || other.NeedsImage
	c.NeedsAudio = c.NeedsAudio || other.NeedsAudio
	c.NeedsVideo = c.NeedsVideo || other.NeedsVideo
}

func (m ChatMessage) capabilityRequirements() CapabilityRequirements {
	var req CapabilityRequirements
	for _, part := range m.ContentParts {
		req.mergePart(part)
	}
	return req
}

func (c *CapabilityRequirements) mergePart(part MessageContentPart) {
	typeName := strings.ToLower(strings.TrimSpace(part.Type))
	switch typeName {
	case MessageContentPartTypeImageURL,
		MessageContentPartTypeImageFile,
		"input_image":
		c.NeedsImage = true
	case MessageContentPartTypeInputAudio:
		c.NeedsAudio = true
	}
}

// Modalities returns the human-readable set of modalities required by the request.
func (c CapabilityRequirements) Modalities() []string {
	var modes []string
	if c.NeedsImage {
		modes = append(modes, "image")
	}
	if c.NeedsAudio {
		modes = append(modes, "audio")
	}
	if c.NeedsVideo {
		modes = append(modes, "video")
	}
	return modes
}

// Describe summarizes the request's multimodal needs for error messages/logging.
func (c CapabilityRequirements) Describe() string {
	modes := c.Modalities()
	switch len(modes) {
	case 0:
		return "text"
	case 1:
		return modes[0]
	case 2:
		return modes[0] + " and " + modes[1]
	default:
		return strings.Join(modes[:len(modes)-1], ", ") + ", and " + modes[len(modes)-1]
	}
}

// HasNonTextContent reports whether the message includes image/audio/file parts that
// providers without multimodal support cannot handle.
func (m ChatMessage) HasNonTextContent() bool {
	for _, part := range m.ContentParts {
		if !part.IsTextual() {
			return true
		}
	}
	return false
}

// FirstNonTextPartType returns the first non-textual part type, or an empty string.
func (m ChatMessage) FirstNonTextPartType() string {
	for _, part := range m.ContentParts {
		if !part.IsTextual() {
			typeName := strings.TrimSpace(part.Type)
			if typeName == "" {
				return "non-text"
			}
			return typeName
		}
	}
	return ""
}
