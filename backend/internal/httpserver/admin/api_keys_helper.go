package admin

import (
    "context"
    "encoding/json"
    "errors"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"

    "github.com/ncecere/open_model_gateway/backend/internal/app"
    "github.com/ncecere/open_model_gateway/backend/internal/config"
    "github.com/ncecere/open_model_gateway/backend/internal/db"
)

type apiKeyIssuer struct {
    Type  string `json:"type"`
    Label string `json:"label"`
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

type apiKeyResponse struct {
    ID                    string            `json:"id"`
    TenantID              string            `json:"tenant_id"`
    TenantName            string            `json:"tenant_name"`
    Issuer                apiKeyIssuer      `json:"issuer"`
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

type createAPIKeyResponse struct {
    APIKeyResponse apiKeyResponse `json:"api_key"`
    Secret         string         `json:"secret"`
    Token          string         `json:"token"`
}

func buildAPIKeyResponse(ctx context.Context, container *app.Container, key db.ApiKey, tenantID uuid.UUID, tenantName string, issuer apiKeyIssuer) (apiKeyResponse, error) {
    if container == nil {
        return apiKeyResponse{}, errors.New("container unavailable")
    }
    created, err := timeFromPg(key.CreatedAt)
    if err != nil {
        return apiKeyResponse{}, err
    }
    var revokedAt *time.Time
    if key.RevokedAt.Valid {
        ts, err := timeFromPg(key.RevokedAt)
        if err != nil {
            return apiKeyResponse{}, err
        }
        revokedAt = &ts
    }
    var lastUsed *time.Time
    if key.LastUsedAt.Valid {
        ts, err := timeFromPg(key.LastUsedAt)
        if err != nil {
            return apiKeyResponse{}, err
        }
        lastUsed = &ts
    }
    var scopes []string
    if len(key.ScopesJson) > 0 {
        _ = json.Unmarshal(key.ScopesJson, &scopes)
    }
    var quota *quotaPayload
    if len(key.QuotaJson) > 0 && string(key.QuotaJson) != "{}" {
        var q quotaPayload
        if err := json.Unmarshal(key.QuotaJson, &q); err == nil {
            quota = &q
        }
    }
    keyID, err := fromPgUUID(key.ID)
    if err != nil {
        return apiKeyResponse{}, err
    }
    schedule, err := budgetRefreshSchedule(ctx, container, tenantID)
    if err != nil {
        return apiKeyResponse{}, err
    }
    return apiKeyResponse{
        ID:                    keyID.String(),
        TenantID:              tenantID.String(),
        TenantName:            tenantName,
        Issuer:                issuer,
        Prefix:                key.Prefix,
        Name:                  key.Name,
        Scopes:                scopes,
        Quota:                 quota,
        BudgetRefreshSchedule: schedule,
        RateLimits:            buildRateLimitPayload(container, key.Prefix, tenantID),
        CreatedAt:             created,
        RevokedAt:             revokedAt,
        LastUsedAt:            lastUsed,
        Revoked:               revokedAt != nil,
    }, nil
}

func budgetRefreshSchedule(ctx context.Context, container *app.Container, tenantID uuid.UUID) (string, error) {
    if container == nil || container.Config == nil {
        return "", errors.New("configuration unavailable")
    }
    schedule := config.NormalizeBudgetRefreshSchedule(container.Config.Budgets.RefreshSchedule)
    if tenantID == uuid.Nil || container.Queries == nil {
        return schedule, nil
    }
    override, err := container.Queries.GetTenantBudgetOverride(ctx, toPgUUID(tenantID))
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

func buildRateLimitPayload(container *app.Container, prefix string, tenantID uuid.UUID) *rateLimitPayload {
    if container == nil {
        return nil
    }
    keyCfg, tenantCfg := container.EffectiveRateLimits(prefix, tenantID)
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

func resolveAPIKeyIssuer(key db.ApiKey, tenantName string, ownerName string, ownerEmail string) apiKeyIssuer {
    issuer := strings.TrimSpace(tenantName)
    issuerType := "tenant"
    if key.Kind == db.ApiKeyKindPersonal {
        issuerType = "personal"
        issuer = strings.TrimSpace(ownerName)
        if issuer == "" {
            issuer = strings.TrimSpace(ownerEmail)
        }
        if issuer == "" {
            issuer = "Personal"
        }
    }
    return apiKeyIssuer{Type: issuerType, Label: issuer}
}
