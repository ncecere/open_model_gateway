package admin

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
)

func recordAudit(c *fiber.Ctx, container *app.Container, action, resourceType, resourceID string, metadata any) error {
	userID, ok := adminUserIDFromContext(c.UserContext())
	if !ok {
		return fmt.Errorf("missing admin context")
	}
	if container.AdminAudit == nil {
		return fmt.Errorf("audit service unavailable")
	}

	// Build metadata map with actor email
	var metaMap fiber.Map
	if metadata == nil {
		metaMap = fiber.Map{}
	} else if m, ok := metadata.(fiber.Map); ok {
		metaMap = m
	} else if m, ok := metadata.(map[string]any); ok {
		metaMap = fiber.Map{}
		for k, v := range m {
			metaMap[k] = v
		}
	} else {
		// If metadata is something else, wrap it
		metaMap = fiber.Map{"data": metadata}
	}

	// Add actor email if available
	if user, ok := adminUserFromContext(c.UserContext()); ok && user.Email != "" {
		metaMap["actor_email"] = user.Email
	}

	if _, err := json.Marshal(metaMap); err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	return container.AdminAudit.Record(c.Context(), userID, action, resourceType, resourceID, metaMap)
}
