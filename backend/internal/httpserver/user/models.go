package user

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

type userModelResponse struct {
	Alias    string `json:"alias"`
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
}

func (h *userHandler) listModels(c *fiber.Ctx) error {
	if h.modelSvc == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "model catalog unavailable")
	}
	models, err := h.modelSvc.List(c.Context())
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	resp := make([]userModelResponse, 0, len(models))
	for _, model := range models {
		resp = append(resp, userModelResponse{
			Alias:    model.Alias,
			Provider: model.Provider,
			Enabled:  model.Enabled,
		})
	}
	return c.JSON(fiber.Map{"models": resp})
}
