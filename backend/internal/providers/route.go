package providers

import (
	"context"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

// Route represents a single provider deployment that can serve a public alias.
type Route struct {
	Alias      string
	Provider   string
	Model      string
	Weight     int
	Metadata   map[string]string
	Chat       ChatCompletions
	ChatStream ChatStreaming
	Embedding  EmbeddingsProvider
	Image      ImagesProvider
	AudioTranscribe AudioTranscriber
	AudioTranslate AudioTranslator
	TextToSpeech   TextToSpeech
	TextToSpeechStream TextToSpeechStreaming
	Models     ModelLister
	Health     func(ctx context.Context) error
}

// ResolveDeployment extracts deployment identifier from route metadata.
func (r Route) ResolveDeployment() string {
	if r.Metadata != nil {
		if dep := r.Metadata["deployment"]; dep != "" {
			return dep
		}
	}
	return r.Model
}

// ToModel converts route metadata back to a models.Model struct for APIs.
func (r Route) ToModel() models.Model {
	return models.Model{
		Alias:           r.Alias,
		Provider:        r.Provider,
		ProviderModel:   r.Model,
		ContextWindow:   0,
		MaxOutputTokens: 0,
	}
}
