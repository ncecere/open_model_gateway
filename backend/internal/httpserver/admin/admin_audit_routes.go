package admin

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	auditservice "github.com/ncecere/open_model_gateway/backend/internal/services/audit"
)

type auditRoutes struct {
	container *app.Container
	service   *auditservice.Service
}

type auditLogResponse struct {
	ID         string          `json:"id"`
	UserID     *string         `json:"user_id,omitempty"`
	Action     string          `json:"action"`
	Resource   string          `json:"resource"`
	ResourceID string          `json:"resource_id"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time       `json:"created_at"`
}

func registerAdminAuditRoutes(router fiber.Router, container *app.Container) {
	handler := &auditRoutes{container: container, service: auditservice.NewService(container.Queries)}
	group := router.Group("/audit")
	group.Get("/logs", handler.list)
}

func (h *auditRoutes) list(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	filter := auditservice.Filter{Limit: 50}
	if val := strings.TrimSpace(c.Query("limit")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			filter.Limit = int32(parsed)
		}
	}
	if val := strings.TrimSpace(c.Query("offset")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			filter.Offset = int32(parsed)
		}
	}
	if val := strings.TrimSpace(c.Query("user_id")); val != "" {
		id, err := uuid.Parse(val)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid user_id")
		}
		filter.UserID = id
	}
	filter.Action = strings.TrimSpace(c.Query("action"))
	filter.ResourceType = strings.TrimSpace(c.Query("resource"))

	logs, err := h.service.List(c.Context(), filter)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(fiber.Map{"logs": []any{}, "limit": filter.Limit, "offset": filter.Offset})
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	response := make([]auditLogResponse, 0, len(logs))
	for _, entry := range logs {
		resp := auditLogResponse{
			ID:         entry.ID.String(),
			Action:     entry.Action,
			Resource:   entry.Resource,
			ResourceID: entry.ResourceID,
			Metadata:   json.RawMessage(entry.Metadata),
			CreatedAt:  entry.CreatedAt,
		}
		if entry.UserID != nil {
			id := entry.UserID.String()
			resp.UserID = &id
		}
		response = append(response, resp)
	}

	return c.JSON(fiber.Map{
		"logs":   response,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}
