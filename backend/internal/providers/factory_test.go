package providers

import (
	"context"
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

type stubTranscriber struct{}

func (stubTranscriber) Transcribe(ctx context.Context, req models.AudioTranscriptionRequest) (models.AudioTranscriptionResponse, error) {
	return models.AudioTranscriptionResponse{}, nil
}

type stubSpeech struct{}

func (stubSpeech) Synthesize(ctx context.Context, req models.AudioSpeechRequest) (models.AudioSpeechResponse, error) {
	return models.AudioSpeechResponse{}, nil
}

func TestValidateRouteCapabilities(t *testing.T) {
	entry := config.ModelCatalogEntry{ModelType: "audio_transcription"}
	route := Route{AudioTranscribe: stubTranscriber{}}
	if err := validateRouteCapabilities(entry, route); err != nil {
		t.Fatalf("expected transcription route to pass validation, got %v", err)
	}

	entry.ModelType = "audio_speech"
	if err := validateRouteCapabilities(entry, Route{TextToSpeech: stubSpeech{}}); err != nil {
		t.Fatalf("expected speech route to pass validation, got %v", err)
	}

	if err := validateRouteCapabilities(config.ModelCatalogEntry{ModelType: "audio_transcription"}, Route{}); err == nil {
		t.Fatalf("expected validation error when transcription capability missing")
	}
	if err := validateRouteCapabilities(config.ModelCatalogEntry{ModelType: "audio_speech"}, Route{}); err == nil {
		t.Fatalf("expected validation error when speech capability missing")
	}
}

func TestFactoryDetectsMissingAudioCapabilities(t *testing.T) {
	cfg := &config.Config{
		ModelCatalog: []config.ModelCatalogEntry{
			{Alias: "audio-1", Provider: "stub", ProviderModel: "x", ModelType: "audio_transcription"},
		},
	}
	factory := NewFactory(cfg)
	factory.Register("stub", func(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
		return Route{Alias: entry.Alias}, nil
	})
	if _, err := factory.Build(context.Background()); err == nil {
		t.Fatalf("expected error when audio_transcription route lacks capability")
	}

	cfg.ModelCatalog[0].ModelType = "audio_transcription"
	factory.Register("stub", func(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
		return Route{Alias: entry.Alias, AudioTranscribe: stubTranscriber{}}, nil
	})
	if _, err := factory.Build(context.Background()); err != nil {
		t.Fatalf("expected factory to succeed once capability added, got %v", err)
	}

	cfg.ModelCatalog[0].ModelType = "audio_speech"
	factory.Register("stub", func(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
		return Route{Alias: entry.Alias}, nil
	})
	if _, err := factory.Build(context.Background()); err == nil {
		t.Fatalf("expected error when audio_speech route lacks tts capability")
	}
	factory.Register("stub", func(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
		return Route{Alias: entry.Alias, TextToSpeech: stubSpeech{}}, nil
	})
	if _, err := factory.Build(context.Background()); err != nil {
		t.Fatalf("expected factory to succeed after adding tts capability, got %v", err)
	}
}
