package admin

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

type adminContextKey string

const (
	adminAuthHeaderPrefix  = "bearer "
	adminContextUserKey    = adminContextKey("open-model-gateway/admin-user")
	adminContextUserIDKey  = adminContextKey("open-model-gateway/admin-user-id")
	adminAuthorizationName = "Authorization"
)

func adminAuthMiddleware(container *app.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := strings.TrimSpace(c.Get(adminAuthorizationName))
		token := ""
		if raw != "" && strings.HasPrefix(strings.ToLower(raw), adminAuthHeaderPrefix) {
			token = strings.TrimSpace(raw[len(adminAuthHeaderPrefix):])
		}
		if token == "" {
			token = strings.TrimSpace(c.Cookies(container.Config.Admin.Session.CookieName))
		}
		if token == "" {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "admin authorization required")
		}

		user, err := container.AdminAuth.AuthorizeAccessToken(userContext(c), token)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid or expired token")
		}

		userID, err := fromPgUUID(user.ID)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "invalid user identifier")
		}

		ctx := context.WithValue(userContext(c), adminContextUserKey, user)
		ctx = context.WithValue(ctx, adminContextUserIDKey, userID)
		c.SetUserContext(ctx)
		c.Locals("adminUserID", userID.String())
		c.Locals("adminUser", user)
		return c.Next()
	}
}

func adminUserFromContext(ctx context.Context) (db.User, bool) {
	if ctx == nil {
		return db.User{}, false
	}
	val := ctx.Value(adminContextUserKey)
	if val == nil {
		return db.User{}, false
	}
	user, ok := val.(db.User)
	return user, ok
}

func adminUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	if ctx == nil {
		return uuid.UUID{}, false
	}
	val := ctx.Value(adminContextUserIDKey)
	if val == nil {
		return uuid.UUID{}, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}
