package models

type Model struct {
	Alias           string            `json:"alias"`
	Provider        string            `json:"provider"`
	ProviderModel   string            `json:"provider_model"`
	ContextWindow   int32             `json:"context_window"`
	MaxOutputTokens int32             `json:"max_output_tokens"`
	Modalities      []string          `json:"modalities"`
	SupportsTools   bool              `json:"supports_tools"`
	Capabilities    ModelCapabilities `json:"capabilities,omitempty"`
}

type ModelCapabilities struct {
	ImageInput bool `json:"image_input,omitempty"`
	AudioInput bool `json:"audio_input,omitempty"`
	VideoInput bool `json:"video_input,omitempty"`
}
