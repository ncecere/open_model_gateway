package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ncecere/open_model_gateway/backend/internal/accounts"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/catalog"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/email"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/runtime/bootstrap"
	runtimebudgets "github.com/ncecere/open_model_gateway/backend/internal/runtime/budgets"
	runtimeratelimits "github.com/ncecere/open_model_gateway/backend/internal/runtime/ratelimits"
	adminconfigsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminconfig"
	adminprovidersvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminprovider"
	adminusersvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminuser"
	tenantservice "github.com/ncecere/open_model_gateway/backend/internal/services/tenant"
	usageService "github.com/ncecere/open_model_gateway/backend/internal/services/usage"
)

// Services bundles the foundational services required by the container.
type Services struct {
	Config            *config.Config
	Queries           *db.Queries
	ReportingLocation *time.Location
	Personal          *accounts.PersonalService
	DefaultModels     *catalog.DefaultModelService
	Usage             *usageService.Service
	Tenant            *tenantservice.Service
	AdminProvider     *adminprovidersvc.Service
	AdminAuth         *auth.AdminAuthService
	AdminUsers        *adminusersvc.Service
	KeyRateLimits     map[string]limits.LimitConfig
	TenantRateLimits  map[uuid.UUID]limits.LimitConfig
}

// BuildServices constructs the service stage and applies config overrides.
func BuildServices(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool) (*Services, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if pool == nil {
		return nil, fmt.Errorf("db pool is required")
	}
	locName := strings.TrimSpace(cfg.Reporting.Timezone)
	if locName == "" {
		locName = "UTC"
	}
	reportingLoc, err := time.LoadLocation(locName)
	if err != nil {
		return nil, fmt.Errorf("load reporting timezone: %w", err)
	}
	queries := db.New(pool)
	adminconfigsvc.ApplyOverrides(ctx, queries, cfg)
	personalSvc := accounts.NewPersonalService(pool, queries)
	defaultModels := catalog.NewDefaultModelService(queries)
	usageSvc := usageService.NewService(queries, reportingLoc)
	tenantSvc := tenantservice.NewService(cfg, queries, reportingLoc)
	providerSvc := adminprovidersvc.NewService(cfg)
	if err := runtimeratelimits.LoadDefaults(ctx, queries, cfg); err != nil {
		return nil, fmt.Errorf("load rate limit defaults: %w", err)
	}
	if err := runtimebudgets.Load(ctx, queries, cfg); err != nil {
		return nil, fmt.Errorf("load budget defaults: %w", err)
	}
	adminAuth, err := auth.NewAdminAuthService(ctx, cfg.Admin, queries, personalSvc)
	if err != nil {
		return nil, fmt.Errorf("init admin auth: %w", err)
	}
	mailSender := email.NewSMTPSender(cfg.Budgets.Alert.SMTP)
	adminUserSvc := adminusersvc.NewService(queries, personalSvc, adminAuth, mailSender, cfg.Budgets.Alert.SMTP.From, cfg.Public.BaseURL, nil)
	keyLimitOverrides, err := runtimeratelimits.LoadKeyOverrides(ctx, queries)
	if err != nil {
		return nil, fmt.Errorf("load api key rate limits: %w", err)
	}
	if keyLimitOverrides == nil {
		keyLimitOverrides = make(map[string]limits.LimitConfig)
	}
	tenantLimitOverrides, err := runtimeratelimits.LoadTenantOverrides(ctx, queries)
	if err != nil {
		return nil, fmt.Errorf("load tenant rate limits: %w", err)
	}
	if tenantLimitOverrides == nil {
		tenantLimitOverrides = make(map[uuid.UUID]limits.LimitConfig)
	}
	if err := bootstrap.Ensure(ctx, bootstrap.Params{
		Queries:      queries,
		AdminAuth:    adminAuth,
		PersonalSvc:  personalSvc,
		Bootstrap:    cfg.Bootstrap,
		Budgets:      cfg.Budgets,
		KeyLimits:    keyLimitOverrides,
		TenantLimits: tenantLimitOverrides,
	}); err != nil {
		return nil, err
	}
	return &Services{
		Config:            cfg,
		Queries:           queries,
		ReportingLocation: reportingLoc,
		Personal:          personalSvc,
		DefaultModels:     defaultModels,
		Usage:             usageSvc,
		Tenant:            tenantSvc,
		AdminProvider:     providerSvc,
		AdminAuth:         adminAuth,
		AdminUsers:        adminUserSvc,
		KeyRateLimits:     keyLimitOverrides,
		TenantRateLimits:  tenantLimitOverrides,
	}, nil
}
