package user

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

const userAuthHeaderPrefix = "bearer "

func userAuthMiddleware(container *app.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractBearer(c)
		if token == "" {
			token = strings.TrimSpace(c.Cookies(container.Config.Admin.Session.CookieName))
		}
		if token == "" {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "authorization required")
		}

		user, err := container.AdminAuth.AuthorizeAccessToken(userContext(c), token)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid or expired token")
		}
		userID, err := uuidFromPg(user.ID)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "invalid user id")
		}

		attachUserContext(c, user, userID)
		return c.Next()
	}
}

func extractBearer(c *fiber.Ctx) string {
	raw := strings.TrimSpace(c.Get("Authorization"))
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, userAuthHeaderPrefix) {
		return ""
	}
	return strings.TrimSpace(raw[len(userAuthHeaderPrefix):])
}
