package admin

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	adminusersvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminuser"
)

func registerAdminUserRoutes(router fiber.Router, container *app.Container) {
	handler := &adminUserHandler{
		container: container,
		service:   container.AdminUsers,
	}
	group := router.Group("/users")
	group.Get("/", handler.list)
	group.Post("/", handler.create)
	group.Get("/:userID/tenants", handler.listUserTenants)
}

type adminUserHandler struct {
	container *app.Container
	service   *adminusersvc.Service
}

type adminUserResponse struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type userTenantMembershipResponse struct {
	TenantID   string    `json:"tenant_id"`
	TenantName string    `json:"tenant_name"`
	Role       string    `json:"role"`
	Status     string    `json:"status"`
	JoinedAt   time.Time `json:"joined_at"`
}

type createUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (h *adminUserHandler) list(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "user service unavailable")
	}

	limit := int32(100)
	offset := int32(0)
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	if v := strings.TrimSpace(c.Query("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}

	records, err := h.service.List(c.Context(), limit, offset)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	users := make([]adminUserResponse, 0, len(records))
	for _, u := range records {
		users = append(users, mapAdminUser(u))
	}

	return c.JSON(fiber.Map{
		"users":  users,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *adminUserHandler) create(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "user service unavailable")
	}

	var req createUserRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	user, err := h.service.Upsert(c.Context(), adminusersvc.CreateParams{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, adminusersvc.ErrEmailRequired):
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		default:
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	resp := mapAdminUser(user)

	if err := recordAudit(c, h.container, "admin_user.upsert", "user", resp.ID, fiber.Map{
		"email": resp.Email,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func mapAdminUser(user adminusersvc.User) adminUserResponse {
	return adminUserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		Name:        user.Name,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		LastLoginAt: user.LastLoginAt,
	}
}

func (h *adminUserHandler) listUserTenants(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	if h.container == nil || h.container.Queries == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "user service unavailable")
	}
	rawID := strings.TrimSpace(c.Params("userID"))
	if rawID == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "user id required")
	}
	userID, err := uuid.Parse(rawID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid user id")
	}
	rows, err := h.container.Queries.ListUserTenants(c.Context(), toPgUUID(userID))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	resp := make([]userTenantMembershipResponse, 0, len(rows))
	for _, row := range rows {
		joined, err := timeFromPg(row.CreatedAt)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
		resp = append(resp, userTenantMembershipResponse{
			TenantID:   row.TenantID.String(),
			TenantName: row.TenantName,
			Role:       string(row.Role),
			Status:     string(row.TenantStatus),
			JoinedAt:   joined,
		})
	}
	return c.JSON(fiber.Map{
		"tenants": resp,
	})
}
