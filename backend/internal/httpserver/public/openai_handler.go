package public

import (
	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
)

type openAIHandler struct {
	container          *app.Container
	executor           *executor.Executor
	chatPipeline       *chatPipeline
	chatStreamPipeline *chatStreamPipeline
	imagePipeline      *imagePipeline
	audioPipeline      *audioPipeline
	embeddingPipeline  *embeddingPipeline
	moderationPipeline *moderationPipeline
}

func newOpenAIHandler(container *app.Container) *openAIHandler {
	exec := executor.New(container)
	return &openAIHandler{
		container:          container,
		executor:           exec,
		chatPipeline:       newChatPipeline(container, exec),
		chatStreamPipeline: newChatStreamPipeline(container),
		imagePipeline:      newImagePipeline(container, exec),
		audioPipeline:      newAudioPipeline(container),
		embeddingPipeline:  newEmbeddingPipeline(container, exec),
		moderationPipeline: newModerationPipeline(container, exec),
	}
}
