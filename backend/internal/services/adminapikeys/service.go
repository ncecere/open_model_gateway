package adminapikeys

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

const (
	ScopeAdmin  = "admin"
	ScopeSystem = "system"
)

var (
	ErrServiceUnavailable = errors.New("admin api key service unavailable")
	ErrNameRequired       = errors.New("name is required")
	ErrScopeInvalid       = errors.New("scope must be admin or system")
	ErrOwnerRequired      = errors.New("owner user required for admin scope")
	ErrOwnerNotAllowed    = errors.New("owner user not allowed for system scope")
	ErrCreatorRequired    = errors.New("creator user required")
	ErrExpiryRequired     = errors.New("expires_in_seconds required")
	ErrExpiryInvalid      = errors.New("expiry must be in the future")
	ErrTokenInvalid       = errors.New("invalid admin api key")
	ErrTokenRevoked       = errors.New("admin api key revoked")
	ErrTokenExpired       = errors.New("admin api key expired")
)

type Service struct {
	queries *db.Queries
	now     func() time.Time
}

type CreateParams struct {
	Name            string
	Scope           string
	OwnerUserID     *uuid.UUID
	CreatedByUserID uuid.UUID
	ExpiresAt       time.Time
}

type CreateResult struct {
	Key   db.AdminApiKey
	Token string
}

func NewService(queries *db.Queries) *Service {
	return &Service{
		queries: queries,
		now:     time.Now,
	}
}

func (s *Service) Create(ctx context.Context, params CreateParams) (CreateResult, error) {
	if s == nil || s.queries == nil {
		return CreateResult{}, ErrServiceUnavailable
	}
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return CreateResult{}, ErrNameRequired
	}
	scope := strings.ToLower(strings.TrimSpace(params.Scope))
	if scope == "" {
		scope = ScopeAdmin
	}
	if scope != ScopeAdmin && scope != ScopeSystem {
		return CreateResult{}, ErrScopeInvalid
	}
	if params.CreatedByUserID == uuid.Nil {
		return CreateResult{}, ErrCreatorRequired
	}
	if params.ExpiresAt.IsZero() {
		return CreateResult{}, ErrExpiryRequired
	}
	if params.ExpiresAt.Before(s.now()) {
		return CreateResult{}, ErrExpiryInvalid
	}

	switch scope {
	case ScopeAdmin:
		if params.OwnerUserID == nil || *params.OwnerUserID == uuid.Nil {
			return CreateResult{}, ErrOwnerRequired
		}
	case ScopeSystem:
		if params.OwnerUserID != nil && *params.OwnerUserID != uuid.Nil {
			return CreateResult{}, ErrOwnerNotAllowed
		}
	}

	prefix, secret, token, err := auth.GenerateAPIKey()
	if err != nil {
		return CreateResult{}, err
	}
	hash, err := auth.HashPassword(secret)
	if err != nil {
		return CreateResult{}, err
	}

	var owner pgtype.UUID
	if params.OwnerUserID != nil && *params.OwnerUserID != uuid.Nil {
		owner = toPgUUID(*params.OwnerUserID)
	}
	record, err := s.queries.CreateAdminAPIKey(ctx, db.CreateAdminAPIKeyParams{
		Name:            name,
		Prefix:          prefix,
		SecretHash:      hash,
		Scope:           db.AdminApiKeyScope(scope),
		OwnerUserID:     owner,
		CreatedByUserID: toPgUUID(params.CreatedByUserID),
		ExpiresAt:       toPgTime(params.ExpiresAt),
	})
	if err != nil {
		return CreateResult{}, err
	}

	return CreateResult{Key: record, Token: token}, nil
}

func (s *Service) List(ctx context.Context) ([]db.ListAdminAPIKeysRow, error) {
	if s == nil || s.queries == nil {
		return nil, ErrServiceUnavailable
	}
	return s.queries.ListAdminAPIKeys(ctx)
}

func (s *Service) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]db.ListAdminAPIKeysRow, error) {
	if s == nil || s.queries == nil {
		return nil, ErrServiceUnavailable
	}
	rows, err := s.queries.ListAdminAPIKeysByOwner(ctx, toPgUUID(ownerID))
	if err != nil {
		return nil, err
	}
	items := make([]db.ListAdminAPIKeysRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, db.ListAdminAPIKeysRow(row))
	}
	return items, nil
}

func (s *Service) Revoke(ctx context.Context, id uuid.UUID) (db.AdminApiKey, error) {
	if s == nil || s.queries == nil {
		return db.AdminApiKey{}, ErrServiceUnavailable
	}
	return s.queries.RevokeAdminAPIKey(ctx, toPgUUID(id))
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.AdminApiKey, error) {
	if s == nil || s.queries == nil {
		return db.AdminApiKey{}, ErrServiceUnavailable
	}
	return s.queries.GetAdminAPIKeyByID(ctx, toPgUUID(id))
}

func (s *Service) Authorize(ctx context.Context, token string) (db.User, db.AdminApiKey, error) {
	if s == nil || s.queries == nil {
		return db.User{}, db.AdminApiKey{}, ErrServiceUnavailable
	}
	prefix, secret, err := splitToken(token)
	if err != nil {
		return db.User{}, db.AdminApiKey{}, ErrTokenInvalid
	}
	record, err := s.queries.GetAdminAPIKeyByPrefix(ctx, prefix)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, db.AdminApiKey{}, ErrTokenInvalid
		}
		return db.User{}, db.AdminApiKey{}, err
	}
	if record.RevokedAt.Valid {
		return db.User{}, db.AdminApiKey{}, ErrTokenRevoked
	}
	if record.ExpiresAt.Valid && record.ExpiresAt.Time.Before(s.now()) {
		return db.User{}, db.AdminApiKey{}, ErrTokenExpired
	}
	match, err := auth.VerifyPassword(secret, record.SecretHash)
	if err != nil {
		return db.User{}, db.AdminApiKey{}, err
	}
	if !match {
		return db.User{}, db.AdminApiKey{}, ErrTokenInvalid
	}

	userID := record.CreatedByUserID
	if record.Scope == db.AdminApiKeyScope(ScopeAdmin) {
		if record.OwnerUserID.Valid {
			userID = record.OwnerUserID
		} else {
			return db.User{}, db.AdminApiKey{}, ErrTokenInvalid
		}
	}

	user, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		return db.User{}, db.AdminApiKey{}, ErrTokenInvalid
	}
	_ = s.queries.UpdateAdminAPIKeyLastUsed(ctx, record.ID)
	return user, record, nil
}

func splitToken(token string) (string, string, error) {
	clean := strings.TrimSpace(token)
	if clean == "" {
		return "", "", errors.New("token required")
	}
	if !strings.HasPrefix(clean, "sk-") {
		return "", "", errors.New("token must start with sk-")
	}
	parts := strings.SplitN(strings.TrimPrefix(clean, "sk-"), ".", 2)
	if len(parts) != 2 {
		return "", "", errors.New("token format invalid")
	}
	prefix := strings.TrimSpace(parts[0])
	secret := strings.TrimSpace(parts[1])
	if prefix == "" || secret == "" {
		return "", "", errors.New("token format invalid")
	}
	return prefix, secret, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toPgTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func fromPgUUID(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, fmt.Errorf("uuid missing")
	}
	return uuid.UUID(id.Bytes), nil
}
