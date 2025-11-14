package public

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

const authBearerPrefix = "bearer "

// apiKeyAuth validates the Authorization bearer token and injects request metadata.
func apiKeyAuth(container *app.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
		if raw == "" {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "authorization header required")
		}

		if !strings.HasPrefix(strings.ToLower(raw), authBearerPrefix) {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "bearer token required")
		}

		key := strings.TrimSpace(raw[len(authBearerPrefix):])
		prefix, secret, err := splitAPIKey(key)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusUnauthorized, err.Error())
		}

		ctx := userContext(c)
		record, err := container.Queries.GetAPIKeyByPrefix(ctx, prefix)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid api key")
			}
			return httputil.WriteError(c, fiber.StatusInternalServerError, "api key lookup failed")
		}

		if record.RevokedAt.Valid {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "api key revoked")
		}

		if record.SecretHash == "" {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "api key invalid")
		}

		match, err := auth.VerifyPassword(secret, record.SecretHash)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "api key verification failed")
		}
		if !match {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid api key")
		}

		tenant, err := container.Queries.GetTenantByID(ctx, record.TenantID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httputil.WriteError(c, fiber.StatusUnauthorized, "tenant not found")
			}
			return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant lookup failed")
		}
		if tenant.Status != db.TenantStatusActive {
			return httputil.WriteError(c, fiber.StatusForbidden, "tenant is not active")
		}

		rc, err := app.BuildRequestContext(ctx, container, record)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}

		if err := container.Queries.UpdateAPIKeyLastUsed(ctx, record.ID); err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to update key usage")
		}

		c.Locals(requestctx.FiberLocalsKey(), rc)
		newCtx := requestctx.WithContext(ctx, rc)
		c.SetUserContext(newCtx)

		return c.Next()
	}
}

func splitAPIKey(token string) (string, string, error) {
	if token == "" {
		return "", "", errors.New("api key required")
	}

	withoutPrefix := strings.TrimPrefix(token, "sk-")
	if withoutPrefix == token {
		return "", "", errors.New("api key must start with sk-")
	}

	parts := strings.SplitN(withoutPrefix, ".", 2)
	if len(parts) != 2 {
		return "", "", errors.New("api key format invalid")
	}

	prefix := parts[0]
	secret := strings.TrimSpace(parts[1])
	if prefix == "" || secret == "" {
		return "", "", errors.New("api key format invalid")
	}

	return prefix, secret, nil
}

func userContext(c *fiber.Ctx) context.Context {
	if c == nil {
		return context.Background()
	}
	if uc := c.UserContext(); uc != nil {
		return uc
	}
	return context.Background()
}
