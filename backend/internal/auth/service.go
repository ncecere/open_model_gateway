package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/accounts"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

const (
	ProviderLocal = "local"
	ProviderOIDC  = "oidc"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrOIDCDisabled       = errors.New("oidc authentication disabled")
)

type AdminAuthService struct {
	cfg          config.AdminConfig
	queries      *db.Queries
	accounts     *accounts.PersonalService
	tokenManager *TokenManager
	oidc         *OIDCProvider
}

func NewAdminAuthService(ctx context.Context, cfg config.AdminConfig, queries *db.Queries, accountsSvc *accounts.PersonalService) (*AdminAuthService, error) {
	tokenManager, err := NewTokenManager(cfg.Session.JWTSecret, cfg.Session.AccessTokenTTL, cfg.Session.RefreshTokenTTL, "open-model-gateway-admin")
	if err != nil {
		return nil, err
	}

	var oidcProvider *OIDCProvider
	if cfg.OIDC.Enabled {
		oidcProvider, err = NewOIDCProvider(ctx, cfg.OIDC)
		if err != nil {
			return nil, err
		}
	}

	return &AdminAuthService{
		cfg:          cfg,
		queries:      queries,
		accounts:     accountsSvc,
		tokenManager: tokenManager,
		oidc:         oidcProvider,
	}, nil
}

func (s *AdminAuthService) AuthenticateLocal(ctx context.Context, email, password string) (*TokenPair, db.User, error) {
	if !s.cfg.Local.Enabled {
		return nil, db.User{}, errors.New("local authentication disabled")
	}

	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, db.User{}, ErrInvalidCredentials
		}
		return nil, db.User{}, fmt.Errorf("lookup user: %w", err)
	}
	if s.accounts != nil && !user.PersonalTenantID.Valid {
		updated, _, perr := s.accounts.EnsurePersonalTenant(ctx, user)
		if perr != nil {
			return nil, db.User{}, fmt.Errorf("ensure personal tenant: %w", perr)
		}
		user = updated
	}

	cred, err := s.queries.GetCredentialByUserAndProvider(ctx, db.GetCredentialByUserAndProviderParams{
		UserID:   user.ID,
		Provider: ProviderLocal,
		Issuer:   ProviderLocal,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, db.User{}, ErrInvalidCredentials
		}
		return nil, db.User{}, fmt.Errorf("load credential: %w", err)
	}

	if !cred.PasswordHash.Valid {
		return nil, db.User{}, ErrInvalidCredentials
	}

	match, err := VerifyPassword(password, cred.PasswordHash.String)
	if err != nil {
		return nil, db.User{}, err
	}
	if !match {
		return nil, db.User{}, ErrInvalidCredentials
	}

	if err := s.queries.UpdateUserLastLogin(ctx, user.ID); err != nil {
		return nil, db.User{}, fmt.Errorf("update last login: %w", err)
	}

	userUUID, err := toUUID(user.ID)
	if err != nil {
		return nil, db.User{}, err
	}

	pair, err := s.tokenManager.Generate(userUUID, user.Email)
	if err != nil {
		return nil, db.User{}, err
	}

	return pair, user, nil
}

func (s *AdminAuthService) UpsertLocalPassword(ctx context.Context, userID uuid.UUID, email string, password string) error {
	if !s.cfg.Local.Enabled {
		return errors.New("local authentication disabled")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	_, err = s.queries.UpsertCredential(ctx, db.UpsertCredentialParams{
		UserID:       pgtype.UUID{Bytes: userID, Valid: true},
		Provider:     ProviderLocal,
		Issuer:       ProviderLocal,
		Subject:      email,
		PasswordHash: pgtype.Text{String: hash, Valid: true},
		Metadata:     json.RawMessage(`{}`),
	})
	if err != nil {
		return fmt.Errorf("upsert credential: %w", err)
	}
	return nil
}

func (s *AdminAuthService) StartOIDCAuth(state, nonce string) (string, error) {
	if s.oidc == nil {
		return "", ErrOIDCDisabled
	}
	return s.oidc.AuthCodeURL(state, nonce), nil
}

func (s *AdminAuthService) CompleteOIDCAuth(ctx context.Context, code string, expectedNonce string) (*TokenPair, db.User, error) {
	if s.oidc == nil {
		return nil, db.User{}, ErrOIDCDisabled
	}

	identity, err := s.oidc.Exchange(ctx, code, expectedNonce)
	if err != nil {
		return nil, db.User{}, err
	}

	if identity.Email == "" {
		return nil, db.User{}, errors.New("oidc identity missing email")
	}

	var user db.User
	user, err = s.queries.GetUserByEmail(ctx, identity.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			user, err = s.createUserFromOIDC(ctx, identity)
			if err != nil {
				return nil, db.User{}, err
			}
		} else {
			return nil, db.User{}, fmt.Errorf("get user: %w", err)
		}
	}
	if s.accounts != nil && !user.PersonalTenantID.Valid {
		updated, _, perr := s.accounts.EnsurePersonalTenant(ctx, user)
		if perr != nil {
			return nil, db.User{}, fmt.Errorf("ensure personal tenant: %w", perr)
		}
		user = updated
	}

	if err := s.syncUserAdminFlag(ctx, &user, identity); err != nil {
		return nil, db.User{}, err
	}

	if err := s.persistOIDCCredential(ctx, user.ID, identity); err != nil {
		return nil, db.User{}, err
	}

	if err := s.queries.UpdateUserLastLogin(ctx, user.ID); err != nil {
		return nil, db.User{}, fmt.Errorf("update last login: %w", err)
	}

	userUUID, err := toUUID(user.ID)
	if err != nil {
		return nil, db.User{}, err
	}

	pair, err := s.tokenManager.Generate(userUUID, user.Email)
	if err != nil {
		return nil, db.User{}, err
	}

	return pair, user, nil
}

func (s *AdminAuthService) createUserFromOIDC(ctx context.Context, identity *OIDCIdentity) (db.User, error) {
	name := identity.Name
	if name == "" {
		name = identity.PreferredName
	}
	if name == "" {
		name = identity.Email
	}

	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email: identity.Email,
		Name:  name,
	})
	if err != nil {
		return db.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *AdminAuthService) persistOIDCCredential(ctx context.Context, userID pgtype.UUID, identity *OIDCIdentity) error {
	issuer := identity.Issuer
	if issuer == "" {
		issuer = s.cfg.OIDC.Issuer
	}

	metadata := identity.MetadataJSON
	if len(metadata) == 0 {
		raw := map[string]any{
			"email": identity.Email,
			"name":  identity.Name,
		}
		b, _ := json.Marshal(raw)
		metadata = b
	}

	_, err := s.queries.UpsertCredential(ctx, db.UpsertCredentialParams{
		UserID:       userID,
		Provider:     ProviderOIDC,
		Issuer:       issuer,
		Subject:      identity.Subject,
		PasswordHash: pgtype.Text{Valid: false},
		Metadata:     json.RawMessage(metadata),
	})
	if err != nil {
		return fmt.Errorf("upsert oidc credential: %w", err)
	}
	return nil
}

func (s *AdminAuthService) syncUserAdminFlag(ctx context.Context, user *db.User, identity *OIDCIdentity) error {
	if len(s.cfg.OIDC.AdminRoles) == 0 || user == nil || identity == nil {
		return nil
	}
	if user.IsSuperAdmin == identity.IsAdmin {
		return nil
	}
	if err := s.queries.SetUserSuperAdmin(ctx, db.SetUserSuperAdminParams{
		ID:           user.ID,
		IsSuperAdmin: identity.IsAdmin,
	}); err != nil {
		return fmt.Errorf("update user admin flag: %w", err)
	}
	user.IsSuperAdmin = identity.IsAdmin
	return nil
}

func (s *AdminAuthService) AllowedAuthMethods() []string {
	methods := []string{}
	if s.cfg.Local.Enabled {
		methods = append(methods, ProviderLocal)
	}
	if s.oidc != nil {
		methods = append(methods, ProviderOIDC)
	}
	return methods
}

func (s *AdminAuthService) IssueTokenPair(user db.User) (*TokenPair, error) {
	userUUID, err := toUUID(user.ID)
	if err != nil {
		return nil, err
	}
	return s.tokenManager.Generate(userUUID, user.Email)
}

func (s *AdminAuthService) ValidateRefreshToken(token string) (uuid.UUID, error) {
	if token == "" {
		return uuid.Nil, errors.New("refresh token required")
	}

	parsed, err := jwtParse(token, s.tokenManager.secret)
	if err != nil {
		return uuid.Nil, err
	}

	if parsed.Claims["typ"] != "refresh" {
		return uuid.Nil, errors.New("invalid token type")
	}

	subject, ok := parsed.Claims["sub"].(string)
	if !ok || subject == "" {
		return uuid.Nil, errors.New("invalid subject")
	}

	userID, err := uuid.Parse(subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse subject: %w", err)
	}
	return userID, nil
}

func (s *AdminAuthService) ValidateAccessToken(token string) (uuid.UUID, error) {
	if token == "" {
		return uuid.Nil, errors.New("access token required")
	}

	parsed, err := jwtParse(token, s.tokenManager.secret)
	if err != nil {
		return uuid.Nil, err
	}
	if parsed.Claims["typ"] != "access" {
		return uuid.Nil, errors.New("invalid token type")
	}
	subject, ok := parsed.Claims["sub"].(string)
	if !ok || subject == "" {
		return uuid.Nil, errors.New("invalid subject")
	}
	userID, err := uuid.Parse(subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse subject: %w", err)
	}
	return userID, nil
}

func (s *AdminAuthService) AuthorizeAccessToken(ctx context.Context, token string) (db.User, error) {
	userID, err := s.ValidateAccessToken(token)
	if err != nil {
		return db.User{}, err
	}

	record, err := s.queries.GetUserByID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return db.User{}, err
	}
	return record, nil
}

func jwtParse(token string, secret []byte) (*jwtTokenWrapper, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return &jwtTokenWrapper{Claims: claims}, nil
}

type jwtTokenWrapper struct {
	Claims jwt.MapClaims
}

func toUUID(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.Nil, errors.New("uuid is invalid")
	}
	return uuid.UUID(id.Bytes), nil
}
