package user

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
)

func recordUserAudit(c *fiber.Ctx, container *app.Container, action, resourceType, resourceID string, metadata any) error {
	userID, ok := userIDFromContext(c.UserContext())
	if !ok {
		return fmt.Errorf("missing user context")
	}
	if container.AdminAudit == nil {
		return fmt.Errorf("audit service unavailable")
	}
	meta := metadata
	if metadata == nil {
		meta = fiber.Map{}
	}
	if _, err := json.Marshal(meta); err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	return container.AdminAudit.Record(c.Context(), userID, action, resourceType, resourceID, metadata)
}
