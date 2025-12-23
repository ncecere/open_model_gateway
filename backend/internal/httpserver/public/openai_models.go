package public

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

func (h *openAIHandler) listModels(c *fiber.Ctx) error {
	ctx := c.UserContext()
	var rc *requestctx.Context
	if rctx, ok := requestctx.FromContext(ctx); ok {
		rc = rctx
		if status, err := h.container.UsageLogger.CheckBudget(ctx, rctx, time.Now().UTC()); err == nil {
			httputil.ApplyBudgetHeaders(c, status)
		}
	}

	aliases := h.container.Engine.ListAliases()
	models := make([]openAIModel, 0, len(aliases))
	now := time.Now().Unix()

	for alias, routes := range aliases {
		if len(routes) == 0 {
			continue
		}
		if rc != nil && !h.container.IsModelAllowed(rc.TenantID, alias) {
			continue
		}
		route := routes[0]
		deployment := route.Metadata["deployment"]
		models = append(models, openAIModel{
			ID:         alias,
			Object:     "model",
			OwnedBy:    route.Provider,
			Created:    now,
			Deployment: deployment,
		})
	}

	return c.JSON(openAIModelList{
		Object: "list",
		Data:   models,
	})
}

type openAIModel struct {
	ID         string `json:"id"`
	Object     string `json:"object"`
	OwnedBy    string `json:"owned_by"`
	Created    int64  `json:"created"`
	Deployment string `json:"deployment"`
}

type openAIModelList struct {
	Object string        `json:"object"`
	Data   []openAIModel `json:"data"`
}
