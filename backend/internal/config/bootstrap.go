package config

import "time"

// BootstrapConfig holds bootstrap seeding configuration.
type BootstrapConfig struct {
	Tenants       []BootstrapTenant       `mapstructure:"tenants"`
	AdminUsers    []BootstrapAdminUser    `mapstructure:"admin_users"`
	APIKeys       []BootstrapAPIKey       `mapstructure:"api_keys"`
	Memberships   []BootstrapMembership   `mapstructure:"memberships"`
	TenantLimits  []BootstrapTenantLimit  `mapstructure:"tenant_limits"`
	TenantBudgets []BootstrapTenantBudget `mapstructure:"tenant_budgets"`
}

// BootstrapTenant defines a tenant to seed.
type BootstrapTenant struct {
	Name   string `mapstructure:"name"`
	Status string `mapstructure:"status"`
}

// BootstrapAdminUser defines an admin user to seed.
type BootstrapAdminUser struct {
	Email      string `mapstructure:"email"`
	Name       string `mapstructure:"name"`
	Password   string `mapstructure:"password"`
	SuperAdmin *bool  `mapstructure:"super_admin"`
}

// IsSuperAdmin returns whether this user should be a super admin.
func (u BootstrapAdminUser) IsSuperAdmin() bool {
	if u.SuperAdmin == nil {
		return true
	}
	return *u.SuperAdmin
}

// BootstrapAPIKey defines an API key to seed.
type BootstrapAPIKey struct {
	Tenant    string             `mapstructure:"tenant"`
	Prefix    string             `mapstructure:"prefix"`
	Secret    string             `mapstructure:"secret"`
	Name      string             `mapstructure:"name"`
	RateLimit BootstrapRateLimit `mapstructure:"rate_limit"`
}

// BootstrapMembership defines a tenant membership to seed.
type BootstrapMembership struct {
	Tenant string `mapstructure:"tenant"`
	Email  string `mapstructure:"email"`
	Role   string `mapstructure:"role"`
}

// BootstrapTenantLimit defines tenant rate limits to seed.
type BootstrapTenantLimit struct {
	Tenant string             `mapstructure:"tenant"`
	Limits BootstrapRateLimit `mapstructure:"limits"`
}

// BootstrapTenantBudget defines tenant budget settings to seed.
type BootstrapTenantBudget struct {
	Tenant           string        `mapstructure:"tenant"`
	BudgetUSD        *float64      `mapstructure:"budget_usd"`
	WarningThreshold *float64      `mapstructure:"warning_threshold"`
	RefreshSchedule  string        `mapstructure:"refresh_schedule"`
	AlertEmails      []string      `mapstructure:"alert_emails"`
	AlertWebhooks    []string      `mapstructure:"alert_webhooks"`
	AlertCooldown    time.Duration `mapstructure:"alert_cooldown"`
}

// BootstrapRateLimit defines rate limit settings for bootstrap.
type BootstrapRateLimit struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
	TokensPerMinute   int `mapstructure:"tokens_per_minute"`
	ParallelRequests  int `mapstructure:"parallel_requests"`
}
