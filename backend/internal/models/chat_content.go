package models

import "strings"

const (
	MessageContentPartTypeText       = "text"
	MessageContentPartTypeInputText  = "input_text"
	MessageContentPartTypeOutputText = "output_text"
	MessageContentPartTypeImageURL   = "image_url"
	MessageContentPartTypeImageFile  = "image_file"
	MessageContentPartTypeInputAudio = "input_audio"
)

// MessageContentPart represents a single piece of chat message content, mirroring the
// OpenAI Chat Completions schema so handlers can forward text, image, and audio inputs.
type MessageContentPart struct {
	Type       string                     `json:"type"`
	Text       string                     `json:"text,omitempty"`
	ImageURL   *MessageContentImageURL    `json:"image_url,omitempty"`
	ImageFile  *MessageContentImageFile   `json:"image_file,omitempty"`
	InputAudio *MessageContentInputAudio  `json:"input_audio,omitempty"`
	InputImage *MessageContentImageObject `json:"input_image,omitempty"`
}

// MessageContentImageURL captures externally hosted image references.
type MessageContentImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// MessageContentImageFile represents file-store backed image references.
type MessageContentImageFile struct {
	FileID   string `json:"file_id,omitempty"`
	FileData string `json:"file_data,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// MessageContentInputAudio conveys inline audio inputs (base64 payloads).
type MessageContentInputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format,omitempty"`
}

// MessageContentImageObject is reserved for future inline image payloads to stay aligned
// with the Responses API schema.
type MessageContentImageObject struct {
	Data   string `json:"data"`
	Format string `json:"format,omitempty"`
}

// IsTextual returns true when the content part should contribute to the legacy string
// representation (text/input/output text variations).
func (p MessageContentPart) IsTextual() bool {
	switch strings.ToLower(strings.TrimSpace(p.Type)) {
	case MessageContentPartTypeText,
		MessageContentPartTypeInputText,
		MessageContentPartTypeOutputText:
		return true
	default:
		return false
	}
}

// TextFromContentParts flattens text-like parts into a single string so existing
// text-only code paths keep functioning while we add richer content support.
func TextFromContentParts(parts []MessageContentPart) string {
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	for _, part := range parts {
		if !part.IsTextual() {
			continue
		}
		if part.Text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(part.Text)
	}
	return b.String()
}
