package admin

import (
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	admintenantsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admintenant"
)

type tenantHandler struct {
	container *app.Container
	service   *admintenantsvc.Service
}

type listTenantResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	BudgetLimitUSD   float64   `json:"budget_limit_usd"`
	BudgetUsedUSD    float64   `json:"budget_used_usd"`
	WarningThreshold *float64  `json:"warning_threshold,omitempty"`
}

type listPersonalTenantResponse struct {
	TenantID         string    `json:"tenant_id"`
	UserID           string    `json:"user_id"`
	UserEmail        string    `json:"user_email"`
	UserName         string    `json:"user_name"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	BudgetLimitUSD   float64   `json:"budget_limit_usd"`
	BudgetUsedUSD    float64   `json:"budget_used_usd"`
	WarningThreshold *float64  `json:"warning_threshold,omitempty"`
	MembershipCount  int64     `json:"membership_count"`
}

type createTenantRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type updateTenantStatusRequest struct {
	Status string `json:"status"`
}

type updateTenantDetailsRequest struct {
	Name string `json:"name"`
}

type createAPIKeyRequest struct {
	Name       string                  `json:"name"`
	Scopes     []string                `json:"scopes"`
	Quota      *quotaPayload           `json:"quota"`
	RateLimits *apiKeyRateLimitRequest `json:"rate_limits,omitempty"`
}

type quotaPayload struct {
	BudgetUSD        float64 `json:"budget_usd,omitempty"`
	BudgetCents      int64   `json:"budget_cents,omitempty"`
	WarningThreshold float64 `json:"warning_threshold,omitempty"`
}

type tenantRateLimitRequest struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	ParallelRequests  int `json:"parallel_requests"`
}

type tenantRateLimitResponse struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	ParallelRequests  int `json:"parallel_requests"`
}

type apiKeyRateLimitRequest struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	ParallelRequests  int `json:"parallel_requests"`
}

type membershipRequest struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

type tenantModelsRequest struct {
	Models []string `json:"models"`
}

type membershipResponse struct {
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
