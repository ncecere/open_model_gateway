package admin

import (
	"testing"

	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
)

func TestValidateAPIKeyRateLimitRequest(t *testing.T) {
	tenantID := uuid.New()
	container := &app.Container{
		Config: &config.Config{
			RateLimits: config.RateLimitConfig{
				DefaultRequestsPerMinute:      100,
				DefaultTokensPerMinute:        1_000,
				DefaultParallelRequestsKey:    10,
				DefaultParallelRequestsTenant: 20,
			},
		},
		DefaultKeyLimit: limits.LimitConfig{
			RequestsPerMinute: 100,
			TokensPerMinute:   1_000,
			ParallelRequests:  10,
		},
		DefaultTenantLimit: limits.LimitConfig{
			RequestsPerMinute: 100,
			TokensPerMinute:   1_000,
			ParallelRequests:  20,
		},
		TenantRateLimits: map[uuid.UUID]limits.LimitConfig{
			tenantID: {
				RequestsPerMinute: 80,
				TokensPerMinute:   800,
				ParallelRequests:  8,
			},
		},
	}

	payload := &apiKeyRateLimitRequest{
		RequestsPerMinute: 60,
		TokensPerMinute:   600,
		ParallelRequests:  4,
	}
	cfg, err := validateAPIKeyRateLimitRequest(container, tenantID, payload)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if cfg.RequestsPerMinute != 60 || cfg.TokensPerMinute != 600 || cfg.ParallelRequests != 4 {
		t.Fatalf("unexpected config %+v", cfg)
	}

	payload.RequestsPerMinute = 90
	if _, err := validateAPIKeyRateLimitRequest(container, tenantID, payload); err == nil {
		t.Fatal("expected error when exceeding tenant limit")
	}

	if _, err := validateAPIKeyRateLimitRequest(container, tenantID, &apiKeyRateLimitRequest{}); err != nil {
		t.Fatalf("zero payload should be ignored: %v", err)
	}
}

func TestValidateQuotaAgainstBudget(t *testing.T) {
	if err := validateQuotaAgainstBudget(&quotaPayload{BudgetUSD: 200}, 100); err == nil {
		t.Fatal("expected error for exceeding tenant budget")
	}
	if err := validateQuotaAgainstBudget(&quotaPayload{WarningThreshold: 2}, 100); err == nil {
		t.Fatal("expected error for invalid threshold")
	}
	if err := validateQuotaAgainstBudget(&quotaPayload{BudgetUSD: 80, WarningThreshold: 0.8}, 100); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}
