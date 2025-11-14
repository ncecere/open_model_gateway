package user

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	usageservice "github.com/ncecere/open_model_gateway/backend/internal/services/usage"
)

type userAPIKeyResponse struct {
	ID                    string            `json:"id"`
	TenantID              string            `json:"tenant_id"`
	Prefix                string            `json:"prefix"`
	Name                  string            `json:"name"`
	Scopes                []string          `json:"scopes"`
	Quota                 *quotaPayload     `json:"quota,omitempty"`
	BudgetRefreshSchedule string            `json:"budget_refresh_schedule"`
	RateLimits            *rateLimitPayload `json:"rate_limits,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
	RevokedAt             *time.Time        `json:"revoked_at,omitempty"`
	LastUsedAt            *time.Time        `json:"last_used_at,omitempty"`
	Revoked               bool              `json:"revoked"`
}

type quotaPayload struct {
	BudgetUSD        float64 `json:"budget_usd,omitempty"`
	BudgetCents      int64   `json:"budget_cents,omitempty"`
	WarningThreshold float64 `json:"warning_threshold,omitempty"`
}

type rateLimitDetails struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	ParallelRequests  int `json:"parallel_requests"`
}

type rateLimitPayload struct {
	Key    rateLimitDetails `json:"key"`
	Tenant rateLimitDetails `json:"tenant"`
}

type createUserAPIKeyRequest struct {
	Name   string        `json:"name"`
	Scopes []string      `json:"scopes"`
	Quota  *quotaPayload `json:"quota"`
}

type createUserAPIKeyResponse struct {
	APIKey userAPIKeyResponse `json:"api_key"`
	Secret string             `json:"secret"`
	Token  string             `json:"token"`
}

func (h *userHandler) registerAPIKeyRoutes(group fiber.Router) {
	group.Get("/api-keys", h.listAPIKeys)
	group.Post("/api-keys", h.createAPIKey)
	group.Post("/api-keys/:apiKeyID/revoke", h.revokeAPIKey)
	group.Get("/api-keys/:apiKeyID/usage", h.getAPIKeyUsage)

	group.Get("/tenants/:tenantID/api-keys", h.listTenantAPIKeys)
	group.Post("/tenants/:tenantID/api-keys", h.createTenantAPIKey)
	group.Post("/tenants/:tenantID/api-keys/:apiKeyID/revoke", h.revokeTenantAPIKey)
	group.Get("/tenants/:tenantID/api-keys/:apiKeyID/usage", h.getTenantAPIKeyUsage)
}

func (h *userHandler) listAPIKeys(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}

	records, err := h.container.Queries.ListPersonalAPIKeysByUser(c.Context(), user.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]userAPIKeyResponse, 0, len(records))
	for _, rec := range records {
		resp, err := h.buildUserAPIKeyResponse(c.Context(), rec)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
		out = append(out, resp)
	}

	return c.JSON(fiber.Map{"api_keys": out})
}

func (h *userHandler) createAPIKey(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}

	var req createUserAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "name is required")
	}

	tenantID := user.PersonalTenantID
	if !tenantID.Valid {
		if h.container.Accounts == nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "personal tenant missing")
		}
		updated, tenant, err := h.container.Accounts.EnsurePersonalTenant(c.Context(), user)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
		user = updated
		tenantID = tenant.ID
	}

	scopesJSON, err := marshalScopes(req.Scopes)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid scopes")
	}
	quotaJSON, err := marshalQuota(req.Quota)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid quota")
	}

	prefix, secret, token, err := auth.GenerateAPIKey()
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to generate api key")
	}
	hash, err := auth.HashPassword(secret)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to hash api key")
	}

	record, err := h.container.Queries.CreateAPIKey(c.Context(), db.CreateAPIKeyParams{
		TenantID:    tenantID,
		Prefix:      prefix,
		SecretHash:  hash,
		Name:        req.Name,
		ScopesJson:  scopesJSON,
		QuotaJson:   quotaJSON,
		Kind:        db.ApiKeyKindPersonal,
		OwnerUserID: user.ID,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	resp, err := h.buildUserAPIKeyResponse(c.Context(), record)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(createUserAPIKeyResponse{
		APIKey: resp,
		Secret: secret,
		Token:  token,
	})
}

func (h *userHandler) revokeAPIKey(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}

	rawID := strings.TrimSpace(c.Params("apiKeyID"))
	if rawID == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "api key id required")
	}
	apiKeyUUID, err := uuid.Parse(rawID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid api key id")
	}

	record, err := h.container.Queries.GetAPIKeyByID(c.Context(), toPgUUID(apiKeyUUID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "api key not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if !record.OwnerUserID.Valid || record.OwnerUserID != user.ID {
		return httputil.WriteError(c, fiber.StatusForbidden, "cannot modify this api key")
	}

	revoked, err := h.container.Queries.RevokeAPIKey(c.Context(), toPgUUID(apiKeyUUID))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	resp, err := h.buildUserAPIKeyResponse(c.Context(), revoked)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	resp.Revoked = true

	return c.JSON(resp)
}

func (h *userHandler) getAPIKeyUsage(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	apiKeyUUID, err := uuid.Parse(strings.TrimSpace(c.Params("apiKeyID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid api key id")
	}
	record, err := h.container.Queries.GetAPIKeyByID(c.Context(), toPgUUID(apiKeyUUID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "api key not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !record.OwnerUserID.Valid || record.OwnerUserID != user.ID {
		return httputil.WriteError(c, fiber.StatusForbidden, "cannot view this api key")
	}
	period := strings.TrimSpace(c.Query("period"))
	if period == "" {
		period = "30d"
	}
	timezone := strings.TrimSpace(c.Query("timezone"))
	if h.usage == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "usage service unavailable")
	}
	summary, err := h.usage.SummarizeAPIKeyUsage(c.Context(), record, period, timezone)
	if err != nil {
		switch {
		case errors.Is(err, usageservice.ErrInvalidPeriod):
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, usageservice.ErrInvalidTimezone):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid timezone")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(summary)
}

func (h *userHandler) listTenantAPIKeys(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	role, err := h.lookupTenantRole(c.Context(), user, tenantUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	records, err := h.container.Queries.ListAPIKeysByTenant(c.Context(), toPgUUID(tenantUUID))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	responses := make([]userAPIKeyResponse, 0, len(records))
	for _, rec := range records {
		resp, err := h.buildUserAPIKeyResponse(c.Context(), rec)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
		responses = append(responses, resp)
	}
	return c.JSON(fiber.Map{
		"role":     string(role),
		"api_keys": responses,
	})
}

func (h *userHandler) createTenantAPIKey(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	role, err := h.lookupTenantRole(c.Context(), user, tenantUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !canManageTenantKeys(role) {
		return httputil.WriteError(c, fiber.StatusForbidden, "insufficient role")
	}
	var req createUserAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "name is required")
	}
	scopesJSON, err := marshalScopes(req.Scopes)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid scopes")
	}
	quotaJSON, err := marshalQuota(req.Quota)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid quota")
	}
	prefix, secret, token, err := auth.GenerateAPIKey()
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to generate api key")
	}
	hash, err := auth.HashPassword(secret)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to hash api key")
	}
	record, err := h.container.Queries.CreateAPIKey(c.Context(), db.CreateAPIKeyParams{
		TenantID:    toPgUUID(tenantUUID),
		Prefix:      prefix,
		SecretHash:  hash,
		Name:        req.Name,
		ScopesJson:  scopesJSON,
		QuotaJson:   quotaJSON,
		Kind:        db.ApiKeyKindService,
		OwnerUserID: user.ID,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	resp, err := h.buildUserAPIKeyResponse(c.Context(), record)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(createUserAPIKeyResponse{
		APIKey: resp,
		Secret: secret,
		Token:  token,
	})
}

func (h *userHandler) revokeTenantAPIKey(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	role, err := h.lookupTenantRole(c.Context(), user, tenantUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !canManageTenantKeys(role) {
		return httputil.WriteError(c, fiber.StatusForbidden, "insufficient role")
	}
	keyUUID, err := uuid.Parse(strings.TrimSpace(c.Params("apiKeyID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid api key id")
	}
	record, err := h.container.Queries.GetAPIKeyByID(c.Context(), toPgUUID(keyUUID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "api key not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	recTenant, err := uuidFromPg(record.TenantID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if recTenant != tenantUUID {
		return httputil.WriteError(c, fiber.StatusForbidden, "cannot modify this api key")
	}
	revoked, err := h.container.Queries.RevokeAPIKey(c.Context(), toPgUUID(keyUUID))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	resp, err := h.buildUserAPIKeyResponse(c.Context(), revoked)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	resp.Revoked = true
	return c.JSON(resp)
}

func (h *userHandler) getTenantAPIKeyUsage(c *fiber.Ctx) error {
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	tenantUUID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	if _, err := h.lookupTenantRole(c.Context(), user, tenantUUID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusForbidden, "membership required")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	keyUUID, err := uuid.Parse(strings.TrimSpace(c.Params("apiKeyID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid api key id")
	}
	record, err := h.container.Queries.GetAPIKeyByID(c.Context(), toPgUUID(keyUUID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.WriteError(c, fiber.StatusNotFound, "api key not found")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	recTenant, err := uuidFromPg(record.TenantID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if recTenant != tenantUUID {
		return httputil.WriteError(c, fiber.StatusForbidden, "cannot view this api key")
	}
	period := strings.TrimSpace(c.Query("period"))
	if period == "" {
		period = "30d"
	}
	timezone := strings.TrimSpace(c.Query("timezone"))
	if h.usage == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "usage service unavailable")
	}
	summary, err := h.usage.SummarizeAPIKeyUsage(c.Context(), record, period, timezone)
	if err != nil {
		switch {
		case errors.Is(err, usageservice.ErrInvalidPeriod):
			return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, usageservice.ErrInvalidTimezone):
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid timezone")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(summary)
}

func (h *userHandler) lookupTenantRole(ctx context.Context, user db.User, tenantID uuid.UUID) (db.MembershipRole, error) {
	if h.container == nil || h.container.Queries == nil {
		return "", errors.New("tenant lookup unavailable")
	}
	membership, err := h.container.Queries.GetTenantMembership(ctx, db.GetTenantMembershipParams{
		TenantID: toPgUUID(tenantID),
		UserID:   user.ID,
	})
	if err != nil {
		return "", err
	}
	return membership.Role, nil
}

func canManageTenantKeys(role db.MembershipRole) bool {
	return role == db.MembershipRoleAdmin || role == db.MembershipRoleOwner
}

func (h *userHandler) buildUserAPIKeyResponse(ctx context.Context, record db.ApiKey) (userAPIKeyResponse, error) {
	tenantID, err := uuidFromPg(record.TenantID)
	if err != nil {
		return userAPIKeyResponse{}, err
	}
	id, err := uuidFromPg(record.ID)
	if err != nil {
		return userAPIKeyResponse{}, err
	}

	created, err := timeFromPg(record.CreatedAt)
	if err != nil {
		return userAPIKeyResponse{}, err
	}

	var revokedAt *time.Time
	if record.RevokedAt.Valid {
		ts, err := timeFromPg(record.RevokedAt)
		if err != nil {
			return userAPIKeyResponse{}, err
		}
		revokedAt = &ts
	}

	var lastUsed *time.Time
	if record.LastUsedAt.Valid {
		ts, err := timeFromPg(record.LastUsedAt)
		if err != nil {
			return userAPIKeyResponse{}, err
		}
		lastUsed = &ts
	}

	scopes, err := unmarshalScopes(record.ScopesJson)
	if err != nil {
		return userAPIKeyResponse{}, err
	}
	quota, err := unmarshalQuota(record.QuotaJson)
	if err != nil {
		return userAPIKeyResponse{}, err
	}
	schedule, err := h.budgetRefreshSchedule(ctx, tenantID)
	if err != nil {
		return userAPIKeyResponse{}, err
	}

	resp := userAPIKeyResponse{
		ID:                    id.String(),
		TenantID:              tenantID.String(),
		Prefix:                record.Prefix,
		Name:                  record.Name,
		Scopes:                scopes,
		Quota:                 quota,
		BudgetRefreshSchedule: schedule,
		RateLimits:            h.buildRateLimitPayload(record.Prefix, tenantID),
		CreatedAt:             created,
		RevokedAt:             revokedAt,
		LastUsedAt:            lastUsed,
		Revoked:               record.RevokedAt.Valid,
	}
	return resp, nil
}

func (h *userHandler) budgetRefreshSchedule(ctx context.Context, tenantID uuid.UUID) (string, error) {
	if h.container == nil || h.container.Config == nil {
		return "", errors.New("configuration unavailable")
	}
	schedule := config.NormalizeBudgetRefreshSchedule(h.container.Config.Budgets.RefreshSchedule)
	if tenantID == uuid.Nil || h.container.Queries == nil {
		return schedule, nil
	}
	override, err := h.container.Queries.GetTenantBudgetOverride(ctx, toPgUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return schedule, nil
		}
		return "", err
	}
	if value := strings.TrimSpace(override.RefreshSchedule); value != "" {
		schedule = config.NormalizeBudgetRefreshSchedule(value)
	}
	return schedule, nil
}

func (h *userHandler) buildRateLimitPayload(prefix string, tenantID uuid.UUID) *rateLimitPayload {
	if h.container == nil {
		return nil
	}
	keyCfg, tenantCfg := h.container.EffectiveRateLimits(prefix, tenantID)
	return &rateLimitPayload{
		Key: rateLimitDetails{
			RequestsPerMinute: keyCfg.RequestsPerMinute,
			TokensPerMinute:   keyCfg.TokensPerMinute,
			ParallelRequests:  keyCfg.ParallelRequests,
		},
		Tenant: rateLimitDetails{
			RequestsPerMinute: tenantCfg.RequestsPerMinute,
			TokensPerMinute:   tenantCfg.TokensPerMinute,
			ParallelRequests:  tenantCfg.ParallelRequests,
		},
	}
}

func marshalScopes(scopes []string) ([]byte, error) {
	if len(scopes) == 0 {
		return []byte("[]"), nil
	}
	clean := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	return json.Marshal(clean)
}

func unmarshalScopes(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func marshalQuota(quota *quotaPayload) ([]byte, error) {
	if quota == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(quota)
}

func unmarshalQuota(raw []byte) (*quotaPayload, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, nil
	}
	var qp quotaPayload
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &qp); err != nil {
		return nil, err
	}
	return &qp, nil
}
