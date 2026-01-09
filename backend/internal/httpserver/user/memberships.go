package user

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
)

func (h *userHandler) registerTenantManagementRoutes(group fiber.Router) {
	group.Get("/tenants/:tenantID/memberships",
		h.requireTenantCapability("tenantID", rbac.CapabilityManageMemberships),
		h.listTenantMemberships,
	)
	group.Post("/tenants/:tenantID/memberships",
		h.requireTenantCapability("tenantID", rbac.CapabilityManageMemberships),
		h.inviteTenantMembership,
	)
	group.Put("/tenants/:tenantID/memberships/:userID/budget",
		h.requireTenantCapability("tenantID", rbac.CapabilityManageMemberBudgets),
		h.updateTenantMembershipBudget,
	)
	group.Delete("/tenants/:tenantID/memberships/:userID",
		h.requireTenantCapability("tenantID", rbac.CapabilityManageMemberships),
		h.removeTenantMembership,
	)
	group.Get("/directory/users", h.listDirectoryUsers)
}

type userMembershipRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type userMembershipResponse struct {
	TenantID  string               `json:"tenant_id"`
	UserID    string               `json:"user_id"`
	Email     string               `json:"email"`
	Role      string               `json:"role"`
	Budget    *memberBudgetPayload `json:"budget,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
	Self      bool                 `json:"self"`
}

type memberBudgetPayload struct {
	BudgetUSD        float64 `json:"budget_usd,omitempty"`
	WarningThreshold float64 `json:"warning_threshold,omitempty"`
	TokenCap         int64   `json:"token_cap,omitempty"`
}

type memberBudgetUpdateRequest struct {
	BudgetUSD        *float64 `json:"budget_usd,omitempty"`
	WarningThreshold *float64 `json:"warning_threshold,omitempty"`
	TokenCap         *int64   `json:"token_cap,omitempty"`
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

	summary, ok := tenantSummaryFromLocals(c)
	if !ok || summary.TenantID != tenantUUID {
		var err error
		summary, err = h.tenantSvc.GetTenantSummary(c.Context(), user, tenantUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
			}
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
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
			Budget:    buildMemberBudget(member.BudgetUsd, member.WarningThreshold, member.TokenCap),
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

	summary, ok := tenantSummaryFromLocals(c)
	if !ok || summary.TenantID != tenantUUID {
		var err error
		summary, err = h.tenantSvc.GetTenantSummary(c.Context(), user, tenantUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
			}
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
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

	if h.container == nil || h.container.Data == nil || h.container.Data.Queries == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	if _, err := h.container.Data.Queries.GetUserByEmail(c.Context(), email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusBadRequest, "user does not exist")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if h.container.AdminTenants == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}

	result, err := h.container.AdminTenants.UpsertMembership(c.Context(), tenantUUID, email, role, "")
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := recordUserAudit(c, h.container, "membership.upsert", "tenant", tenantUUID.String(), fiber.Map{
		"member_id": result.UserID.String(),
		"email":     result.Email,
		"role":      string(result.Role),
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	resp := userMembershipResponse{
		TenantID:  result.TenantID.String(),
		UserID:    result.UserID.String(),
		Email:     result.Email,
		Role:      string(result.Role),
		Budget:    buildMemberBudget(result.BudgetUsd, result.WarningThreshold, result.TokenCap),
		CreatedAt: result.Created,
		Self:      false,
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *userHandler) updateTenantMembershipBudget(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.container == nil || h.container.Data == nil || h.container.Data.Queries == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "membership service unavailable")
	}

	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	memberUUID, err := uuid.Parse(strings.TrimSpace(c.Params("userID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid user id")
	}

	var req memberBudgetUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	if req.BudgetUSD == nil && req.WarningThreshold == nil && req.TokenCap == nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "budget update required")
	}

	if req.BudgetUSD != nil {
		if *req.BudgetUSD < 0 {
			return httputil.WriteError(c, fiber.StatusBadRequest, "budget_usd must be >= 0")
		}
		if *req.BudgetUSD > 0 {
			tenantBudget, _, err := h.container.EffectiveTenantBudget(c.Context(), tenantUUID)
			if err != nil {
				return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
			}
			if tenantBudget > 0 && *req.BudgetUSD > tenantBudget {
				return httputil.WriteError(c, fiber.StatusBadRequest, "budget_usd exceeds tenant budget")
			}
		}
	}
	if req.WarningThreshold != nil {
		if *req.WarningThreshold < 0 || *req.WarningThreshold > 1 {
			return httputil.WriteError(c, fiber.StatusBadRequest, "warning_threshold must be between 0 and 1")
		}
	}
	if req.TokenCap != nil && *req.TokenCap < 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "token_cap must be >= 0")
	}

	current, err := h.container.Data.Queries.GetTenantMembership(c.Context(), db.GetTenantMembershipParams{
		TenantID: toPgUUID(tenantUUID),
		UserID:   toPgUUID(memberUUID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "membership not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	budgetUSD := current.BudgetUsd
	warningThreshold := current.WarningThreshold
	tokenCap := current.TokenCap
	if req.BudgetUSD != nil {
		budgetUSD = decimal.NewFromFloat(*req.BudgetUSD).Round(2)
	}
	if req.WarningThreshold != nil {
		warningThreshold = decimal.NewFromFloat(*req.WarningThreshold)
	}
	if req.TokenCap != nil {
		tokenCap = *req.TokenCap
	}

	updated, err := h.container.Data.Queries.UpdateTenantMembershipBudget(c.Context(), db.UpdateTenantMembershipBudgetParams{
		TenantID:         toPgUUID(tenantUUID),
		UserID:           toPgUUID(memberUUID),
		BudgetUsd:        budgetUSD,
		WarningThreshold: warningThreshold,
		TokenCap:         tokenCap,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if err := recordUserAudit(c, h.container, "membership.budget.update", "tenant", tenantUUID.String(), fiber.Map{
		"member_id":         memberUUID.String(),
		"budget_usd":        budgetUSD.InexactFloat64(),
		"warning_threshold": warningThreshold.InexactFloat64(),
		"token_cap":         tokenCap,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	self := false
	if currentUserID, err := uuidFromPg(user.ID); err == nil && currentUserID == memberUUID {
		self = true
	}
	return c.JSON(fiber.Map{
		"tenant_id": tenantUUID.String(),
		"user_id":   memberUUID.String(),
		"budget":    buildMemberBudget(updated.BudgetUsd, updated.WarningThreshold, updated.TokenCap),
		"self":      self,
	})
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

	summary, ok := tenantSummaryFromLocals(c)
	if !ok || summary.TenantID != tenantUUID {
		var err error
		summary, err = h.tenantSvc.GetTenantSummary(c.Context(), user, tenantUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
			}
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	currentUserID, err := uuidFromPg(user.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "user context invalid")
	}
	if memberUUID == currentUserID {
		return httputil.WriteError(c, fiber.StatusBadRequest, "cannot remove your own membership")
	}

	targetMembership, err := h.container.Data.Queries.GetTenantMembership(c.Context(), db.GetTenantMembershipParams{
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

	if err := recordUserAudit(c, h.container, "membership.remove", "tenant", tenantUUID.String(), fiber.Map{
		"member_id": memberUUID.String(),
		"role":      string(targetMembership.Role),
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func canManageMemberships(role db.MembershipRole) bool {
	return rbac.HasCapability(role, rbac.CapabilityManageMemberships)
}

func buildMemberBudget(budgetUSD decimal.Decimal, warningThreshold decimal.Decimal, tokenCap int64) *memberBudgetPayload {
	if budgetUSD.Equal(decimal.Zero) && warningThreshold.Equal(decimal.Zero) && tokenCap == 0 {
		return nil
	}
	payload := &memberBudgetPayload{}
	if !budgetUSD.Equal(decimal.Zero) {
		payload.BudgetUSD = budgetUSD.InexactFloat64()
	}
	if !warningThreshold.Equal(decimal.Zero) {
		payload.WarningThreshold = warningThreshold.InexactFloat64()
	}
	if tokenCap > 0 {
		payload.TokenCap = tokenCap
	}
	return payload
}

type directoryUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *userHandler) listDirectoryUsers(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.container == nil || h.container.Data == nil || h.container.Data.Queries == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "directory unavailable")
	}

	allowed := user.IsSuperAdmin
	if !allowed {
		memberships, err := h.tenantSvc.ListUserMemberships(c.Context(), user)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
		for _, membership := range memberships {
			if canManageMemberships(membership.Role) {
				allowed = true
				break
			}
		}
	}
	if !allowed {
		return httputil.WriteError(c, fiber.StatusForbidden, "insufficient permissions")
	}

	query := strings.TrimSpace(c.Query("query"))
	if len(query) < 2 {
		return c.JSON(fiber.Map{"users": []directoryUser{}})
	}
	limit := int32(10)
	if val := strings.TrimSpace(c.Query("limit")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}

	rows, err := h.container.Data.Queries.SearchUsers(c.Context(), db.SearchUsersParams{
		Lower: strings.ToLower(query),
		Limit: limit,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	resp := make([]directoryUser, 0, len(rows))
	for _, row := range rows {
		id, err := uuidFromPg(row.ID)
		if err != nil {
			continue
		}
		resp = append(resp, directoryUser{
			ID:    id.String(),
			Email: row.Email,
			Name:  row.Name,
		})
	}
	return c.JSON(fiber.Map{"users": resp})
}
