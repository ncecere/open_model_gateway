package models

type Model struct {
	Alias           string   `json:"alias"`
	Provider        string   `json:"provider"`
	ProviderModel   string   `json:"provider_model"`
	ContextWindow   int32    `json:"context_window"`
	MaxOutputTokens int32    `json:"max_output_tokens"`
	Modalities      []string `json:"modalities"`
	SupportsTools   bool     `json:"supports_tools"`
}
