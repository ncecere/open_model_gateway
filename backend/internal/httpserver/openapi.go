package httpserver

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/yokeTH/gofiber-scalar/scalar/v2"
	"gopkg.in/yaml.v3"

	"github.com/ncecere/open_model_gateway/backend/openapi"
)

var (
	openAPISpecYAML     = openapi.SpecYAML
	openAPISpecJSON     []byte
	openAPISpecJSONOnce sync.Once
	openAPISpecJSONErr  error
)

func registerOpenAPIRoutes(app *fiber.App) {
	jsonSpec, err := loadOpenAPISpecJSON()
	if err != nil {
		jsonSpec = []byte("{}")
	}

	app.Get("/openapi.yaml", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/yaml")
		c.Set(fiber.HeaderCacheControl, "no-store")
		return c.Send(openAPISpecYAML)
	})

	app.Get("/openapi.json", func(c *fiber.Ctx) error {
		if jsonSpec == nil {
			return fiber.NewError(fiber.StatusInternalServerError, "openapi json unavailable")
		}
		c.Set(fiber.HeaderContentType, "application/json")
		c.Set(fiber.HeaderCacheControl, "no-store")
		return c.Send(jsonSpec)
	})

	app.Get("/docs/*", scalar.New(scalar.Config{
		Path:              "/docs",
		Title:             "Open Model Gateway API Docs",
		FileContentString: string(jsonSpec),
		RawSpecUrl:        "openapi.json",
		ForceOffline:      scalar.ForceOfflineTrue,
	}))
}

func loadOpenAPISpecJSON() ([]byte, error) {
	openAPISpecJSONOnce.Do(func() {
		if len(openAPISpecYAML) == 0 {
			openAPISpecJSONErr = fmt.Errorf("openapi spec is empty")
			return
		}
		var payload any
		if err := yaml.Unmarshal(openAPISpecYAML, &payload); err != nil {
			openAPISpecJSONErr = fmt.Errorf("invalid openapi yaml: %w", err)
			return
		}
		openAPISpecJSON, openAPISpecJSONErr = json.Marshal(payload)
	})
	return openAPISpecJSON, openAPISpecJSONErr
}
