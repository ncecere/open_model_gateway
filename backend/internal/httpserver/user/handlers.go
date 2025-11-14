package user

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/batchdto"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	admincatalogsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admincatalog"
	tenantservice "github.com/ncecere/open_model_gateway/backend/internal/services/tenant"
	usageservice "github.com/ncecere/open_model_gateway/backend/internal/services/usage"
)

type userHandler struct {
	container *app.Container
	usage     *usageservice.Service
	tenantSvc *tenantservice.Service
	modelSvc  *admincatalogsvc.Service
}

type userProfileResponse struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	Name              string     `json:"name"`
	ThemePreference   string     `json:"theme_preference"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`
	PersonalTenantID  *string    `json:"personal_tenant_id,omitempty"`
	IsSuperAdmin      bool       `json:"is_super_admin"`
	CanChangePassword bool       `json:"can_change_password"`
}

type userTenantResponse struct {
	TenantID   string    `json:"tenant_id"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Role       string    `json:"role"`
	JoinedAt   time.Time `json:"joined_at"`
	CreatedAt  time.Time `json:"created_at"`
	IsPersonal bool      `json:"is_personal"`
}

type tenantSummaryResponse struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Status    string              `json:"status"`
	Role      string              `json:"role"`
	CreatedAt time.Time           `json:"created_at"`
	Budget    tenantBudgetSummary `json:"budget"`
}

type tenantBudgetSummary struct {
	LimitUSD         float64 `json:"limit_usd"`
	UsedUSD          float64 `json:"used_usd"`
	RemainingUSD     float64 `json:"remaining_usd"`
	WarningThreshold float64 `json:"warning_threshold"`
	RefreshSchedule  string  `json:"refresh_schedule"`
}

type updateProfileRequest struct {
	Name            *string `json:"name"`
	ThemePreference *string `json:"theme_preference"`
}

var validThemePreferences = map[string]struct{}{
	"system": {},
	"light":  {},
	"dark":   {},
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *userHandler) profile(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}

	resp, err := h.buildProfileResponse(c.Context(), user)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(resp)
}

func (h *userHandler) updateProfile(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.container == nil || h.container.Queries == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "profile service unavailable")
	}

	var req updateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}

	params := db.UpdateUserProfileParams{ID: user.ID}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return httputil.WriteError(c, fiber.StatusBadRequest, "name is required")
		}
		params.Name = pgtype.Text{String: trimmed, Valid: true}
	}

	if req.ThemePreference != nil {
		pref, err := normalizeThemePreference(*req.ThemePreference)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		}
		params.ThemePreference = pgtype.Text{String: pref, Valid: true}
	}

	if !params.Name.Valid && !params.ThemePreference.Valid {
		return httputil.WriteError(c, fiber.StatusBadRequest, "no changes supplied")
	}

	updated, err := h.container.Queries.UpdateUserProfile(c.Context(), params)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	resp, err := h.buildProfileResponse(c.Context(), updated)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(resp)
}

func (h *userHandler) changePassword(c *fiber.Ctx) error {
	if h.container == nil || h.container.AdminAuth == nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "password service unavailable")
	}
	if !h.container.Config.Admin.Local.Enabled {
		return httputil.WriteError(c, fiber.StatusBadRequest, "local authentication disabled")
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}

	var req changePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.CurrentPassword = strings.TrimSpace(req.CurrentPassword)
	req.NewPassword = strings.TrimSpace(req.NewPassword)
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "passwords are required")
	}
	if utf8.RuneCountInString(req.NewPassword) < 8 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "new password must be at least 8 characters")
	}

	cred, err := h.container.Queries.GetCredentialByUserAndProvider(c.Context(), db.GetCredentialByUserAndProviderParams{
		UserID:   user.ID,
		Provider: auth.ProviderLocal,
		Issuer:   auth.ProviderLocal,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusBadRequest, "password login not enabled for this account")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !cred.PasswordHash.Valid {
		return httputil.WriteError(c, fiber.StatusBadRequest, "password login not enabled for this account")
	}

	match, err := auth.VerifyPassword(req.CurrentPassword, cred.PasswordHash.String)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !match {
		return httputil.WriteError(c, fiber.StatusBadRequest, "current password is incorrect")
	}

	userUUID, err := uuidFromPg(user.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "invalid user id")
	}

	if err := h.container.AdminAuth.UpsertLocalPassword(c.Context(), userUUID, user.Email, req.NewPassword); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *userHandler) tenants(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.tenantSvc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	memberships, err := h.tenantSvc.ListUserMemberships(c.Context(), user)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	out := make([]userTenantResponse, 0, len(memberships))
	for _, m := range memberships {
		name := strings.TrimSpace(m.Name)
		if m.IsPersonal {
			name = "Personal"
		} else if name == "" {
			name = fmt.Sprintf("Tenant %s", m.TenantID.String())
		}
		out = append(out, userTenantResponse{
			TenantID:   m.TenantID.String(),
			Name:       name,
			Status:     string(m.Status),
			Role:       string(m.Role),
			JoinedAt:   m.JoinedAt,
			CreatedAt:  m.CreatedAt,
			IsPersonal: m.IsPersonal,
		})
	}
	return c.JSON(fiber.Map{"tenants": out})
}

func (h *userHandler) registerBatchRoutes(group fiber.Router) {
	group.Get("/tenants/:tenantID/batches", h.userListBatches)
	group.Get("/tenants/:tenantID/batches/:batchID", h.userGetBatch)
	group.Post("/tenants/:tenantID/batches/:batchID/cancel", h.userCancelBatch)
	group.Get("/tenants/:tenantID/batches/:batchID/output", h.userDownloadBatchOutput)
	group.Get("/tenants/:tenantID/batches/:batchID/errors", h.userDownloadBatchErrors)
}

func (h *userHandler) userListBatches(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.tenantSvc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	if h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "batches service unavailable")
	}

	tenantUUID, err := uuid.Parse(c.Params("tenantID"))
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

	limit, offset := parseUserBatchPagination(c)
	records, err := h.container.Batches.List(c.UserContext(), tenantUUID, limit, offset)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]batchdto.Batch, 0, len(records))
	for _, record := range records {
		out = append(out, batchdto.FromBatch(record))
	}

	return c.JSON(fiber.Map{
		"object": "list",
		"data":   out,
		"tenant": summary.TenantID.String(),
	})
}

func (h *userHandler) userGetBatch(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.tenantSvc == nil || h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "batch service unavailable")
	}

	tenantUUID, batchUUID, ok := h.parseUserBatchParams(c)
	if !ok {
		return nil
	}
	if _, err := h.tenantSvc.GetTenantSummary(c.Context(), user, tenantUUID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	record, err := h.container.Batches.Get(c.UserContext(), tenantUUID, batchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "batch not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(batchdto.FromBatch(record))
}

func (h *userHandler) userCancelBatch(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.tenantSvc == nil || h.container.Batches == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "batch service unavailable")
	}

	tenantUUID, batchUUID, ok := h.parseUserBatchParams(c)
	if !ok {
		return nil
	}
	summary, err := h.tenantSvc.GetTenantSummary(c.Context(), user, tenantUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !canManageBatches(summary.Role) {
		return httputil.WriteError(c, fiber.StatusForbidden, "insufficient permissions to cancel batches")
	}

	record, err := h.container.Batches.Cancel(c.UserContext(), tenantUUID, batchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "batch not found or cannot transition")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(batchdto.FromBatch(record))
}

func (h *userHandler) userDownloadBatchOutput(c *fiber.Ctx) error {
	return h.userStreamBatchFile(c, true)
}

func (h *userHandler) userDownloadBatchErrors(c *fiber.Ctx) error {
	return h.userStreamBatchFile(c, false)
}

func (h *userHandler) userStreamBatchFile(c *fiber.Ctx, output bool) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.tenantSvc == nil || h.container.Batches == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "batch files unavailable")
	}

	tenantUUID, batchUUID, ok := h.parseUserBatchParams(c)
	if !ok {
		return nil
	}
	if _, err := h.tenantSvc.GetTenantSummary(c.Context(), user, tenantUUID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	batch, err := h.container.Batches.Get(c.UserContext(), tenantUUID, batchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "batch not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	var fileID *uuid.UUID
	filenameSuffix := "output"
	if output {
		fileID = batch.ResultFileID
	} else {
		fileID = batch.ErrorFileID
		filenameSuffix = "errors"
	}
	if fileID == nil {
		return httputil.WriteError(c, fiber.StatusNotFound, fmt.Sprintf("batch %s file not available", filenameSuffix))
	}

	reader, fileRec, err := h.container.Files.Open(c.UserContext(), tenantUUID, *fileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "file not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer reader.Close()

	contentType := fileRec.ContentType
	if contentType == "" {
		contentType = "application/x-ndjson"
	}
	c.Set("Content-Type", contentType)
	if fileRec.Bytes > 0 {
		c.Set("Content-Length", strconv.FormatInt(fileRec.Bytes, 10))
	}
	filename := fileRec.Filename
	if filename == "" {
		filename = fmt.Sprintf("batch_%s_%s.jsonl", batchUUID.String(), filenameSuffix)
	}
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Set("Cache-Control", "no-store")
	if _, err := io.Copy(c, reader); err != nil {
		return err
	}
	return nil
}

func (h *userHandler) parseUserBatchParams(c *fiber.Ctx) (uuid.UUID, uuid.UUID, bool) {
	tenantUUID, err := uuid.Parse(c.Params("tenantID"))
	if err != nil {
		_ = httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	batchUUID, err := uuid.Parse(c.Params("batchID"))
	if err != nil {
		_ = httputil.WriteError(c, fiber.StatusBadRequest, "invalid batch id")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return tenantUUID, batchUUID, true
}

func parseUserBatchPagination(c *fiber.Ctx) (int32, int32) {
	limit := int32(50)
	offset := int32(0)
	if val := strings.TrimSpace(c.Query("limit")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	if val := strings.TrimSpace(c.Query("offset")); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}
	return limit, offset
}

func canManageBatches(role db.MembershipRole) bool {
	return role == db.MembershipRoleOwner || role == db.MembershipRoleAdmin
}

func (h *userHandler) tenantSummary(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	if h.tenantSvc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
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
	resp := tenantSummaryResponse{
		ID:        summary.TenantID.String(),
		Name:      summary.Name,
		Status:    string(summary.Status),
		Role:      string(summary.Role),
		CreatedAt: summary.CreatedAt,
		Budget: tenantBudgetSummary{
			LimitUSD:         summary.Budget.LimitUSD,
			UsedUSD:          summary.Budget.UsedUSD,
			RemainingUSD:     summary.Budget.RemainingUSD,
			WarningThreshold: summary.Budget.WarningThreshold,
			RefreshSchedule:  summary.Budget.RefreshSchedule,
		},
	}
	return c.JSON(resp)
}

func mustUUIDString(id pgtype.UUID) string {
	val, err := uuidFromPg(id)
	if err != nil {
		return ""
	}
	return val.String()
}

func normalizeThemePreference(value string) (string, error) {
	pref := strings.ToLower(strings.TrimSpace(value))
	if pref == "" {
		return "", errors.New("theme preference is required")
	}
	if _, ok := validThemePreferences[pref]; !ok {
		return "", fmt.Errorf("theme preference must be one of 'system', 'light', or 'dark'")
	}
	return pref, nil
}

func (h *userHandler) buildProfileResponse(ctx context.Context, user db.User) (userProfileResponse, error) {
	created, err := timeFromPg(user.CreatedAt)
	if err != nil {
		return userProfileResponse{}, errors.New("invalid created_at")
	}
	updated, err := timeFromPg(user.UpdatedAt)
	if err != nil {
		return userProfileResponse{}, errors.New("invalid updated_at")
	}

	var lastLogin *time.Time
	if user.LastLoginAt.Valid {
		ts, err := timeFromPg(user.LastLoginAt)
		if err != nil {
			return userProfileResponse{}, errors.New("invalid last_login_at")
		}
		lastLogin = &ts
	}

	var personalID *string
	if user.PersonalTenantID.Valid {
		id, err := uuidFromPg(user.PersonalTenantID)
		if err != nil {
			return userProfileResponse{}, errors.New("invalid personal tenant id")
		}
		str := id.String()
		personalID = &str
	}

	canChange := false
	if h.container != nil && h.container.Config.Admin.Local.Enabled && h.container.Queries != nil {
		creds, err := h.container.Queries.ListCredentialsForUser(ctx, user.ID)
		if err == nil {
			for _, cred := range creds {
				if cred.Provider == auth.ProviderLocal && cred.Issuer == auth.ProviderLocal {
					canChange = true
					break
				}
			}
		}
	}

	theme := strings.TrimSpace(user.ThemePreference)
	if theme == "" {
		theme = "system"
	}

	return userProfileResponse{
		ID:                mustUUIDString(user.ID),
		Email:             user.Email,
		Name:              user.Name,
		ThemePreference:   theme,
		CreatedAt:         created,
		UpdatedAt:         updated,
		LastLoginAt:       lastLogin,
		PersonalTenantID:  personalID,
		IsSuperAdmin:      user.IsSuperAdmin,
		CanChangePassword: canChange,
	}, nil
}
