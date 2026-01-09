package config

import "time"

// AdminConfig holds admin portal configuration.
type AdminConfig struct {
	Session AdminSessionConfig `mapstructure:"session"`
	Local   LocalAuthConfig    `mapstructure:"local"`
	OIDC    OIDCConfig         `mapstructure:"oidc"`
}

// AdminSessionConfig holds session settings for the admin portal.
type AdminSessionConfig struct {
	JWTSecret       string        `mapstructure:"jwt_secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	CookieName      string        `mapstructure:"cookie_name"`
}

// LocalAuthConfig holds local authentication settings.
type LocalAuthConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// OIDCConfig holds OIDC authentication settings.
type OIDCConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	Issuer         string        `mapstructure:"issuer"`
	ClientID       string        `mapstructure:"client_id"`
	ClientSecret   string        `mapstructure:"client_secret"`
	RedirectURL    string        `mapstructure:"redirect_url"`
	Scopes         []string      `mapstructure:"scopes"`
	AllowedDomains []string      `mapstructure:"allowed_domains"`
	HTTPTimeout    time.Duration `mapstructure:"http_timeout"`
	RolesClaim     string        `mapstructure:"roles_claim"`
	AllowedRoles   []string      `mapstructure:"allowed_roles"`
	AdminRoles     []string      `mapstructure:"admin_roles"`
}
