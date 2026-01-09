package admin

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
)

func (h *tenantHandler) listMemberships(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleAdmin); err != nil {
		return err
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	members, err := h.service.ListMemberships(c.Context(), tenantID)
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	out := make([]membershipResponse, 0, len(members))
	for _, member := range members {
		out = append(out, membershipResponse{
			TenantID:  member.TenantID.String(),
			UserID:    member.UserID.String(),
			Email:     member.Email,
			Role:      string(member.Role),
			CreatedAt: member.Created,
		})
	}

	if err := recordAudit(c, h.container, "tenant.memberships.view", "tenant", tenantID.String(), fiber.Map{
		"count": len(out),
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"memberships": out})
}

func (h *tenantHandler) upsertMembership(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleOwner); err != nil {
		return err
	}

	var req membershipRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "email is required")
	}
	role, ok := rbac.ParseRole(req.Role)
	if !ok {
		return httputil.WriteError(c, fiber.StatusBadRequest, "role must be owner, admin, viewer, or user")
	}

	if h.container == nil || h.container.Data == nil || h.container.Data.Queries == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	if _, err := h.container.Data.Queries.GetUserByEmail(c.Context(), req.Email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusBadRequest, "user does not exist")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	result, err := h.service.UpsertMembership(c.Context(), tenantID, req.Email, role, strings.TrimSpace(req.Password))
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	resp := membershipResponse{
		TenantID:  result.TenantID.String(),
		UserID:    result.UserID.String(),
		Email:     result.Email,
		Role:      string(result.Role),
		CreatedAt: result.Created,
	}

	if err := recordAudit(c, h.container, "membership.upsert", "tenant", tenantID.String(), fiber.Map{
		"user_id": resp.UserID,
		"role":    resp.Role,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *tenantHandler) removeMembership(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	userID, err := uuid.Parse(strings.TrimSpace(c.Params("userID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid user id")
	}

	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleOwner); err != nil {
		return err
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	if err := h.service.RemoveMembership(c.Context(), tenantID, userID); err != nil {
		return writeTenantServiceError(c, err)
	}

	if err := recordAudit(c, h.container, "membership.remove", "tenant", tenantID.String(), fiber.Map{
		"user_id": userID.String(),
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *tenantHandler) lookupTenantName(ctx context.Context, tenantID uuid.UUID) string {
	if h.container == nil || h.container.Data == nil || h.container.Data.Queries == nil {
		return ""
	}
	record, err := h.container.Data.Queries.GetTenantByID(ctx, toPgUUID(tenantID))
	if err != nil {
		return ""
	}
	return record.Name
}
