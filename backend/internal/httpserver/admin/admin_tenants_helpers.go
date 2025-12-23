package admin

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	admintenantsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admintenant"
)

func validateQuotaAgainstBudget(quota *quotaPayload, limit float64) error {
	if quota == nil {
		return nil
	}
	if quota.BudgetUSD > 0 && limit > 0 && quota.BudgetUSD > limit {
		return fmt.Errorf("budget cannot exceed tenant limit (%.2f)", limit)
	}
	if quota.BudgetUSD < 0 {
		return fmt.Errorf("budget must be positive")
	}
	if quota.WarningThreshold > 1 || quota.WarningThreshold < 0 {
		return fmt.Errorf("warning threshold must be between 0 and 1")
	}
	return nil
}

func validateAPIKeyRateLimitRequest(container *app.Container, tenantID uuid.UUID, payload *apiKeyRateLimitRequest) (*limits.LimitConfig, error) {
	if container == nil || payload == nil {
		return nil, nil
	}
	rpm := payload.RequestsPerMinute
	tpm := payload.TokensPerMinute
	parallel := payload.ParallelRequests
	if rpm == 0 && tpm == 0 && parallel == 0 {
		return nil, nil
	}
	if rpm <= 0 || tpm <= 0 || parallel <= 0 {
		return nil, fmt.Errorf("rate limits must be positive integers")
	}
	keyCfg, tenantCfg := container.EffectiveRateLimits("", tenantID)
	if tenantCfg.RequestsPerMinute > 0 && rpm > tenantCfg.RequestsPerMinute {
		return nil, fmt.Errorf("requests_per_minute cannot exceed tenant limit (%d)", tenantCfg.RequestsPerMinute)
	}
	if keyCfg.RequestsPerMinute > 0 && rpm > keyCfg.RequestsPerMinute {
		return nil, fmt.Errorf("requests_per_minute cannot exceed default limit (%d)", keyCfg.RequestsPerMinute)
	}
	if tenantCfg.TokensPerMinute > 0 && tpm > tenantCfg.TokensPerMinute {
		return nil, fmt.Errorf("tokens_per_minute cannot exceed tenant limit (%d)", tenantCfg.TokensPerMinute)
	}
	if keyCfg.TokensPerMinute > 0 && tpm > keyCfg.TokensPerMinute {
		return nil, fmt.Errorf("tokens_per_minute cannot exceed default limit (%d)", keyCfg.TokensPerMinute)
	}
	if tenantCfg.ParallelRequests > 0 && parallel > tenantCfg.ParallelRequests {
		return nil, fmt.Errorf("parallel_requests cannot exceed tenant limit (%d)", tenantCfg.ParallelRequests)
	}
	if keyCfg.ParallelRequests > 0 && parallel > keyCfg.ParallelRequests {
		return nil, fmt.Errorf("parallel_requests cannot exceed default limit (%d)", keyCfg.ParallelRequests)
	}
	return &limits.LimitConfig{
		RequestsPerMinute: rpm,
		TokensPerMinute:   tpm,
		ParallelRequests:  parallel,
	}, nil
}

func writeTenantServiceError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, admintenantsvc.ErrInvalidModelList),
		errors.Is(err, admintenantsvc.ErrModelNotFound),
		errors.Is(err, admintenantsvc.ErrLocalAuthDisabled):
		status = fiber.StatusBadRequest
	case errors.Is(err, admintenantsvc.ErrAPIKeyTenantMismatch):
		status = fiber.StatusNotFound
	case errors.Is(err, admintenantsvc.ErrServiceUnavailable):
		status = fiber.StatusInternalServerError
	}
	return httputil.WriteError(c, status, err.Error())
}
