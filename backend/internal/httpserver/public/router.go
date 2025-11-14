package public

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
)

// Register wires up the OpenAI-compatible public API routes.
func Register(app *fiber.App, container *app.Container) {
	group := app.Group("/v1", apiKeyAuth(container))
	handler := &openAIHandler{container: container, executor: executor.New(container)}
	group.Get("/models", handler.listModels)
	group.Post("/chat/completions", handler.chatCompletions)
	group.Post("/embeddings", handler.embeddings)
	group.Post("/images/generations", handler.imageGenerations)
	group.Post("/images/edits", handler.imageEdits)
	group.Post("/images/variations", handler.imageVariations)
	group.Post("/audio/transcriptions", handler.audioTranscriptions)
	group.Post("/audio/translations", handler.audioTranslations)
	group.Post("/audio/speech", handler.audioSpeech)

	filesHandler := &filesHandler{container: container}
	group.Get("/files", filesHandler.list)
	group.Post("/files", filesHandler.upload)
	group.Get("/files/:id", filesHandler.get)
	group.Delete("/files/:id", filesHandler.delete)
	group.Get("/files/:id/content", filesHandler.download)
	group.Post("/uploads", filesHandler.createUpload)

	batchHandler := &batchHandler{container: container}
	group.Get("/batches", batchHandler.list)
	group.Post("/batches", batchHandler.create)
	group.Get("/batches/:id", batchHandler.get)
	group.Post("/batches/:id/cancel", batchHandler.cancel)
	group.Get("/batches/:id/output", batchHandler.output)
	group.Get("/batches/:id/errors", batchHandler.errors)
}
