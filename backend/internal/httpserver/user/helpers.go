package user

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/logging"
)

type userContextKey string

const (
	ctxUserKey   = userContextKey("open-model-gateway/user")
	ctxUserIDKey = userContextKey("open-model-gateway/user-id")
)

func userContext(c *fiber.Ctx) context.Context {
	if c == nil {
		return context.Background()
	}
	if uc := c.UserContext(); uc != nil {
		return uc
	}
	return context.Background()
}

func attachUserContext(c *fiber.Ctx, user db.User, id uuid.UUID) {
	ctx := context.WithValue(userContext(c), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxUserIDKey, id)
	c.SetUserContext(ctx)
	c.Locals("userID", id.String())

	// Enrich wide event with user context
	if event, ok := logging.WideEventFromContext(ctx); ok {
		event.SetUserContext(id.String(), user.Email)
	}
}

func userFromContext(ctx context.Context) (db.User, bool) {
	if ctx == nil {
		return db.User{}, false
	}
	val := ctx.Value(ctxUserKey)
	if val == nil {
		return db.User{}, false
	}
	user, ok := val.(db.User)
	return user, ok
}

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	if ctx == nil {
		return uuid.UUID{}, false
	}
	val := ctx.Value(ctxUserIDKey)
	if val == nil {
		return uuid.UUID{}, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func timeFromPg(ts pgtype.Timestamptz) (time.Time, error) {
	if !ts.Valid {
		return time.Time{}, errors.New("invalid timestamp")
	}
	return ts.Time, nil
}

func parsePositiveInt(raw string, fallback int) int {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return fallback
	}
	value, err := strconv.Atoi(clean)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
