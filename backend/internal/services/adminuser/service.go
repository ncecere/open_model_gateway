package adminuser

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/accounts"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

var (
	ErrServiceUnavailable = errors.New("admin user service not initialized")
	ErrEmailRequired      = errors.New("email is required")
)

// Service manages admin-facing user operations.
type Service struct {
	queries   *db.Queries
	accounts  *accounts.PersonalService
	adminAuth *auth.AdminAuthService
}

// NewService wires dependencies for the admin user service.
func NewService(queries *db.Queries, accounts *accounts.PersonalService, adminAuth *auth.AdminAuthService) *Service {
	return &Service{queries: queries, accounts: accounts, adminAuth: adminAuth}
}

// User represents an admin-facing user record.
type User struct {
	ID          uuid.UUID
	Email       string
	Name        string
	Theme       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
}

// CreateParams describes the inputs for creating or updating a user.
type CreateParams struct {
	Email    string
	Name     string
	Password string
}

// List returns paginated users.
func (s *Service) List(ctx context.Context, limit, offset int32) ([]User, error) {
	if s == nil || s.queries == nil {
		return nil, ErrServiceUnavailable
	}
	rows, err := s.queries.ListUsers(ctx, db.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]User, 0, len(rows))
	for _, row := range rows {
		user, err := convertUser(row)
		if err != nil {
			return nil, err
		}
		out = append(out, user)
	}
	return out, nil
}

// Upsert ensures a user exists (creating if necessary) and optionally sets a password.
func (s *Service) Upsert(ctx context.Context, params CreateParams) (User, error) {
	if s == nil || s.queries == nil {
		return User{}, ErrServiceUnavailable
	}
	email := strings.TrimSpace(params.Email)
	name := strings.TrimSpace(params.Name)
	if email == "" {
		return User{}, ErrEmailRequired
	}
	if name == "" {
		name = email
	}

	userRow, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return User{}, err
		}
		userRow, err = s.queries.CreateUser(ctx, db.CreateUserParams{
			Email: email,
			Name:  name,
		})
		if err != nil {
			return User{}, err
		}
	}

	if s.accounts != nil && !userRow.PersonalTenantID.Valid {
		if updated, _, err := s.accounts.EnsurePersonalTenant(ctx, userRow); err == nil {
			userRow = updated
		} else {
			return User{}, err
		}
	}

	userID, err := fromPgUUID(userRow.ID)
	if err != nil {
		return User{}, err
	}

	if pw := strings.TrimSpace(params.Password); pw != "" && s.adminAuth != nil {
		if err := s.adminAuth.UpsertLocalPassword(ctx, userID, email, pw); err != nil {
			return User{}, err
		}
	}

	return convertUser(userRow)
}

func convertUser(row db.User) (User, error) {
	id, err := fromPgUUID(row.ID)
	if err != nil {
		return User{}, err
	}
	created, err := timeFromPg(row.CreatedAt)
	if err != nil {
		return User{}, err
	}
	updated, err := timeFromPg(row.UpdatedAt)
	if err != nil {
		return User{}, err
	}
	var lastLogin *time.Time
	if row.LastLoginAt.Valid {
		ts, err := timeFromPg(row.LastLoginAt)
		if err != nil {
			return User{}, err
		}
		lastLogin = &ts
	}
	theme := strings.TrimSpace(row.ThemePreference)
	if theme == "" {
		theme = "system"
	}

	return User{
		ID:          id,
		Email:       row.Email,
		Name:        row.Name,
		Theme:       theme,
		CreatedAt:   created,
		UpdatedAt:   updated,
		LastLoginAt: lastLogin,
	}, nil
}

func fromPgUUID(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.Nil, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

func timeFromPg(ts pgtype.Timestamptz) (time.Time, error) {
	if !ts.Valid {
		return time.Time{}, errors.New("invalid timestamp")
	}
	return ts.Time, nil
}
