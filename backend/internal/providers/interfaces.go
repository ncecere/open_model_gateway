package providers

import (
	"context"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

type ChatCompletions interface {
	Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error)
}

type ChatStreaming interface {
	ChatStream(ctx context.Context, req models.ChatRequest) (<-chan models.ChatChunk, func() error, error)
}

type ModelLister interface {
	Models(ctx context.Context) ([]models.Model, error)
}

type EmbeddingsProvider interface {
	Embed(ctx context.Context, req models.EmbeddingsRequest) (models.EmbeddingsResponse, error)
}

type ImagesProvider interface {
	Generate(ctx context.Context, req models.ImageRequest) (models.ImageResponse, error)
	Edit(ctx context.Context, req models.ImageEditRequest) (models.ImageResponse, error)
	Variation(ctx context.Context, req models.ImageVariationRequest) (models.ImageResponse, error)
}

type AudioTranscriber interface {
	Transcribe(ctx context.Context, req models.AudioTranscriptionRequest) (models.AudioTranscriptionResponse, error)
}

type AudioTranslator interface {
	Translate(ctx context.Context, req models.AudioTranscriptionRequest) (models.AudioTranscriptionResponse, error)
}

type TextToSpeech interface {
	Synthesize(ctx context.Context, req models.AudioSpeechRequest) (models.AudioSpeechResponse, error)
}

type TextToSpeechStreaming interface {
	SynthesizeStream(ctx context.Context, req models.AudioSpeechRequest) (<-chan models.AudioSpeechChunk, func() error, error)
}
