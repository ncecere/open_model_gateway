package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

type OIDCIdentity struct {
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	PreferredName string
	Roles         []string
	IsAdmin       bool
	MetadataJSON  []byte
	IDToken       string
	AccessToken   string
	RefreshToken  string
	Expiry        time.Time
}

type OIDCProvider struct {
	cfg            config.OIDCConfig
	provider       *oidc.Provider
	oauth2Config   *oauth2.Config
	verifier       *oidc.IDTokenVerifier
	allowedDomains map[string]struct{}
	rolesClaim     string
	allowedRoles   map[string]struct{}
	adminRoles     map[string]struct{}
}

func NewOIDCProvider(ctx context.Context, cfg config.OIDCConfig) (*OIDCProvider, error) {
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	allowed := make(map[string]struct{}, len(cfg.AllowedDomains))
	for _, d := range cfg.AllowedDomains {
		allowed[strings.ToLower(strings.TrimSpace(d))] = struct{}{}
	}

	rolesClaim := strings.TrimSpace(cfg.RolesClaim)
	allowedRoles := normalizeRoleSet(cfg.AllowedRoles)
	adminRoles := normalizeRoleSet(cfg.AdminRoles)

	return &OIDCProvider{
		cfg:            cfg,
		provider:       provider,
		oauth2Config:   oauth2Config,
		verifier:       verifier,
		allowedDomains: allowed,
		rolesClaim:     rolesClaim,
		allowedRoles:   allowedRoles,
		adminRoles:     adminRoles,
	}, nil
}

func (p *OIDCProvider) AuthCodeURL(state string, nonce string) string {
	opts := []oauth2.AuthCodeOption{}
	if nonce != "" {
		opts = append(opts, oidc.Nonce(nonce))
	}
	return p.oauth2Config.AuthCodeURL(state, opts...)
}

func (p *OIDCProvider) Exchange(ctx context.Context, code string, expectedNonce string) (*OIDCIdentity, error) {
	timeout := p.cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	exchangeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	oauth2Token, err := p.oauth2Config.Exchange(exchangeCtx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange auth code: %w", err)
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("oidc: missing id_token in token response")
	}

	idToken, err := p.verifier.Verify(exchangeCtx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify id token: %w", err)
	}

	if expectedNonce != "" && idToken.Nonce != expectedNonce {
		return nil, errors.New("oidc: nonce mismatch")
	}

	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("parse id token claims: %w", err)
	}

	var rawClaims map[string]any
	if err := idToken.Claims(&rawClaims); err != nil {
		return nil, fmt.Errorf("parse raw id token claims: %w", err)
	}

	identity := &OIDCIdentity{
		Issuer:        idToken.Issuer,
		Subject:       idToken.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		PreferredName: claims.PreferredUsername,
		IDToken:       rawIDToken,
		AccessToken:   oauth2Token.AccessToken,
		RefreshToken:  oauth2Token.RefreshToken,
		Expiry:        oauth2Token.Expiry,
		Roles:         extractRolesFromClaims(rawClaims, p.rolesClaim),
	}

	if identity.Email == "" || (p.rolesClaim != "" && len(identity.Roles) == 0) {
		claims, err := p.populateFromUserInfo(exchangeCtx, oauth2Token, identity)
		if err != nil {
			return nil, err
		}
		if len(identity.Roles) == 0 {
			identity.Roles = extractRolesFromClaims(claims, p.rolesClaim)
		}
	}

	if len(p.allowedDomains) > 0 {
		domain, err := emailDomain(identity.Email)
		if err != nil {
			return nil, err
		}
		if _, ok := p.allowedDomains[domain]; !ok {
			return nil, fmt.Errorf("email domain %s not permitted", domain)
		}
	}

	if err := p.applyRoleRules(identity); err != nil {
		return nil, err
	}

	claimsBytes, err := json.Marshal(map[string]any{
		"id_token":      rawIDToken,
		"email":         identity.Email,
		"emailVerified": identity.EmailVerified,
		"name":          identity.Name,
		"preferredName": identity.PreferredName,
		"expiry":        identity.Expiry,
		"roles":         identity.Roles,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal claims: %w", err)
	}
	identity.MetadataJSON = claimsBytes

	return identity, nil
}

func (p *OIDCProvider) populateFromUserInfo(ctx context.Context, token *oauth2.Token, identity *OIDCIdentity) (map[string]any, error) {
	userInfo, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}

	var claims map[string]any
	if err := userInfo.Claims(&claims); err != nil {
		return nil, fmt.Errorf("parse userinfo claims: %w", err)
	}

	if identity.Email == "" {
		email, _ := claims["email"].(string)
		if email == "" {
			return nil, errors.New("oidc: email not present in claims")
		}
		identity.Email = email
		if verified, ok := claims["email_verified"].(bool); ok {
			identity.EmailVerified = verified
		}
	}

	if identity.Name == "" {
		if name, ok := claims["name"].(string); ok && name != "" {
			identity.Name = name
		}
	}

	return claims, nil
}

func emailDomain(email string) (string, error) {
	parts := strings.Split(email, "@")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid email address %q", email)
	}
	domain := strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
	if domain == "" {
		return "", fmt.Errorf("invalid email domain %q", email)
	}
	return domain, nil
}

func (p *OIDCProvider) applyRoleRules(identity *OIDCIdentity) error {
	if len(p.allowedRoles) > 0 {
		if !hasMatchingRole(identity.Roles, p.allowedRoles) {
			return fmt.Errorf("oidc: user missing required role")
		}
	}
	if len(p.adminRoles) > 0 {
		identity.IsAdmin = hasMatchingRole(identity.Roles, p.adminRoles)
	}
	return nil
}

func extractRolesFromClaims(claims map[string]any, field string) []string {
	if len(claims) == 0 || strings.TrimSpace(field) == "" {
		return nil
	}
	value, ok := claims[field]
	if !ok || value == nil {
		return nil
	}
	var roles []string
	switch v := value.(type) {
	case string:
		if role := normalizeRole(v); role != "" {
			roles = append(roles, role)
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				if role := normalizeRole(s); role != "" {
					roles = append(roles, role)
				}
			}
		}
	case []string:
		for _, s := range v {
			if role := normalizeRole(s); role != "" {
				roles = append(roles, role)
			}
		}
	default:
		if arr, ok := tryConvertToStringSlice(v); ok {
			for _, s := range arr {
				if role := normalizeRole(s); role != "" {
					roles = append(roles, role)
				}
			}
		}
	}
	return dedupeRoles(roles)
}

func tryConvertToStringSlice(value any) ([]string, bool) {
	switch v := value.(type) {
	case map[string]any:
		var roles []string
		for key, raw := range v {
			if b, ok := raw.(bool); ok && b {
				roles = append(roles, key)
			}
		}
		return roles, true
	default:
		return nil, false
	}
}

func normalizeRole(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return ""
	}
	return strings.ToLower(role)
}

func dedupeRoles(roles []string) []string {
	if len(roles) == 0 {
		return roles
	}
	seen := make(map[string]struct{}, len(roles))
	result := make([]string, 0, len(roles))
	for _, r := range roles {
		if r == "" {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		result = append(result, r)
	}
	return result
}

func normalizeRoleSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		if role := normalizeRole(v); role != "" {
			set[role] = struct{}{}
		}
	}
	return set
}

func hasMatchingRole(roles []string, set map[string]struct{}) bool {
	if len(set) == 0 {
		return false
	}
	for _, role := range roles {
		if _, ok := set[role]; ok {
			return true
		}
	}
	return false
}
