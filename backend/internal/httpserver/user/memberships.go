package user

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
)

func (h *userHandler) registerTenantManagementRoutes(group fiber.Router) {
	group.Get("/tenants/:tenantID/memberships", h.listTenantMemberships)
	group.Post("/tenants/:tenantID/memberships", h.inviteTenantMembership)
	group.Delete("/tenants/:tenantID/memberships/:userID", h.removeTenantMembership)
}

type userMembershipRequest struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

type userMembershipResponse struct {
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	Self      bool      `json:"self"`
}

func (h *userHandler) listTenantMemberships(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}

	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	summary, err := h.tenantSvc.GetTenantSummary(c.Context(), user, tenantUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if !canManageMemberships(summary.Role) {
		return httputil.WriteError(c, fiber.StatusForbidden, "insufficient permissions")
	}

	if h.container.AdminTenants == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}

	members, err := h.container.AdminTenants.ListMemberships(c.Context(), tenantUUID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	currentUserID, err := uuidFromPg(user.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "user context invalid")
	}

	out := make([]userMembershipResponse, 0, len(members))
	for _, member := range members {
		out = append(out, userMembershipResponse{
			TenantID:  member.TenantID.String(),
			UserID:    member.UserID.String(),
			Email:     member.Email,
			Role:      string(member.Role),
			CreatedAt: member.Created,
			Self:      member.UserID == currentUserID,
		})
	}

	return c.JSON(fiber.Map{"memberships": out})
}

func (h *userHandler) inviteTenantMembership(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}

	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	summary, err := h.tenantSvc.GetTenantSummary(c.Context(), user, tenantUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if !canManageMemberships(summary.Role) {
		return httputil.WriteError(c, fiber.StatusForbidden, "insufficient permissions")
	}

	var req userMembershipRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}

	email := strings.TrimSpace(req.Email)
	if email == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "email is required")
	}
	role, ok := rbac.ParseRole(strings.TrimSpace(req.Role))
	if !ok {
		return httputil.WriteError(c, fiber.StatusBadRequest, "role must be owner, admin, viewer, or user")
	}
	if summary.Role == db.MembershipRoleAdmin && role == db.MembershipRoleOwner {
		return httputil.WriteError(c, fiber.StatusForbidden, "only owners can grant the owner role")
	}

	if h.container.AdminTenants == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}

	result, err := h.container.AdminTenants.UpsertMembership(c.Context(), tenantUUID, email, role, strings.TrimSpace(req.Password))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	resp := userMembershipResponse{
		TenantID:  result.TenantID.String(),
		UserID:    result.UserID.String(),
		Email:     result.Email,
		Role:      string(result.Role),
		CreatedAt: result.Created,
		Self:      false,
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *userHandler) removeTenantMembership(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}

	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	memberUUID, err := uuid.Parse(strings.TrimSpace(c.Params("userID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid user id")
	}

	summary, err := h.tenantSvc.GetTenantSummary(c.Context(), user, tenantUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !canManageMemberships(summary.Role) {
		return httputil.WriteError(c, fiber.StatusForbidden, "insufficient permissions")
	}

	currentUserID, err := uuidFromPg(user.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "user context invalid")
	}
	if memberUUID == currentUserID {
		return httputil.WriteError(c, fiber.StatusBadRequest, "cannot remove your own membership")
	}

	targetMembership, err := h.container.Queries.GetTenantMembership(c.Context(), db.GetTenantMembershipParams{
		TenantID: toPgUUID(tenantUUID),
		UserID:   toPgUUID(memberUUID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "membership not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if summary.Role == db.MembershipRoleAdmin && targetMembership.Role == db.MembershipRoleOwner {
		return httputil.WriteError(c, fiber.StatusForbidden, "only owners may remove other owners")
	}

	if h.container.AdminTenants == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	if err := h.container.AdminTenants.RemoveMembership(c.Context(), tenantUUID, memberUUID); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func canManageMemberships(role db.MembershipRole) bool {
	return role == db.MembershipRoleOwner || role == db.MembershipRoleAdmin
}
