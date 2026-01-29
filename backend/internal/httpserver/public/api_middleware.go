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
	"github.com/ncecere/open_model_gateway/backend/internal/logging"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

const authBearerPrefix = "bearer "

// apiKeyAuth validates the Authorization bearer token and injects request metadata.
func apiKeyAuth(container *app.Container) fiber.Handler {
	authFail := func(failureType string) {
		if container.Observability != nil {
			container.Observability.RecordAuthFailure(failureType, "api")
		}
	}

	return func(c *fiber.Ctx) error {
		raw := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
		if raw == "" {
			authFail("missing_token")
			return httputil.WriteError(c, fiber.StatusUnauthorized, "authorization header required")
		}

		if !strings.HasPrefix(strings.ToLower(raw), authBearerPrefix) {
			authFail("missing_token")
			return httputil.WriteError(c, fiber.StatusUnauthorized, "bearer token required")
		}

		key := strings.TrimSpace(raw[len(authBearerPrefix):])
		prefix, secret, err := splitAPIKey(key)
		if err != nil {
			authFail("invalid_key")
			return httputil.WriteError(c, fiber.StatusUnauthorized, err.Error())
		}

		ctx := userContext(c)
		record, err := container.Data.Queries.GetAPIKeyByPrefix(ctx, prefix)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				authFail("invalid_key")
				return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid api key")
			}
			return httputil.WriteError(c, fiber.StatusInternalServerError, "api key lookup failed")
		}

		if record.RevokedAt.Valid {
			authFail("revoked_key")
			return httputil.WriteError(c, fiber.StatusUnauthorized, "api key revoked")
		}

		if record.SecretHash == "" {
			authFail("invalid_key")
			return httputil.WriteError(c, fiber.StatusUnauthorized, "api key invalid")
		}

		match, err := auth.VerifyPassword(secret, record.SecretHash)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "api key verification failed")
		}
		if !match {
			authFail("invalid_key")
			return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid api key")
		}

		tenant, err := container.Data.Queries.GetTenantByID(ctx, record.TenantID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				authFail("invalid_key")
				return httputil.WriteError(c, fiber.StatusUnauthorized, "tenant not found")
			}
			return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant lookup failed")
		}
		if tenant.Status != db.TenantStatusActive {
			authFail("forbidden")
			return httputil.WriteError(c, fiber.StatusForbidden, "tenant is not active")
		}

		rc, err := app.BuildRequestContext(ctx, container, record)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}

		if err := container.Data.Queries.UpdateAPIKeyLastUsed(ctx, record.ID); err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to update key usage")
		}

		c.Locals(requestctx.FiberLocalsKey(), rc)
		newCtx := requestctx.WithContext(ctx, rc)
		c.SetUserContext(newCtx)

		// Enrich wide event with tenant/API key context
		if event, ok := logging.WideEventFromContext(newCtx); ok {
			event.EnrichFromRequestContext(rc)
		}

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
