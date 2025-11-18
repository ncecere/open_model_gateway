package adminuser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/accounts"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/email"
)

var (
	ErrServiceUnavailable = errors.New("admin user service not initialized")
	ErrEmailRequired      = errors.New("email is required")
)

// Service manages admin-facing user operations.
type Service struct {
	queries     *db.Queries
	accounts    *accounts.PersonalService
	adminAuth   *auth.AdminAuthService
	emailSender email.Sender
	emailFrom   string
	baseURL     string
	logger      *slog.Logger
}

// NewService wires dependencies for the admin user service.
func NewService(queries *db.Queries, accounts *accounts.PersonalService, adminAuth *auth.AdminAuthService, sender email.Sender, from string, baseURL string, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		queries:     queries,
		accounts:    accounts,
		adminAuth:   adminAuth,
		emailSender: sender,
		emailFrom:   strings.TrimSpace(from),
		baseURL:     strings.TrimSpace(baseURL),
		logger:      logger,
	}
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
	Email      string
	Name       string
	Password   string
	SendInvite bool
	InviteBase string
	InvitedBy  string
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
func (s *Service) Upsert(ctx context.Context, params CreateParams) (User, bool, error) {
	if s == nil || s.queries == nil {
		return User{}, false, ErrServiceUnavailable
	}
	email := strings.TrimSpace(params.Email)
	name := strings.TrimSpace(params.Name)
	if email == "" {
		return User{}, false, ErrEmailRequired
	}
	if name == "" {
		name = email
	}

	userRow, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return User{}, false, err
		}
		userRow, err = s.queries.CreateUser(ctx, db.CreateUserParams{
			Email: email,
			Name:  name,
		})
		if err != nil {
			return User{}, false, err
		}
	}

	if s.accounts != nil && !userRow.PersonalTenantID.Valid {
		if updated, _, err := s.accounts.EnsurePersonalTenant(ctx, userRow); err == nil {
			userRow = updated
		} else {
			return User{}, false, err
		}
	}

	userID, err := fromPgUUID(userRow.ID)
	if err != nil {
		return User{}, false, err
	}

	if pw := strings.TrimSpace(params.Password); pw != "" && s.adminAuth != nil {
		if err := s.adminAuth.UpsertLocalPassword(ctx, userID, email, pw); err != nil {
			return User{}, false, err
		}
	}

	user, err := convertUser(userRow)
	if err != nil {
		return User{}, false, err
	}

	inviteSent := false
	if params.SendInvite {
		if err := s.SendInvite(ctx, userRow, params.InviteBase, params.InvitedBy); err != nil {
			if s.logger != nil {
				s.logger.Error("send invite email", "error", err)
			}
		} else {
			inviteSent = true
		}
	}

	return user, inviteSent, nil
}

// SendInvite dispatches an email invite using the configured sender.
func (s *Service) SendInvite(ctx context.Context, user db.User, baseURL string, invitedBy string) error {
	if s == nil {
		return ErrServiceUnavailable
	}
	if s.emailSender == nil || s.emailFrom == "" {
		return errors.New("email sender not configured")
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = strings.TrimRight(s.baseURL, "/")
	}
	if base == "" {
		return errors.New("invite base url required")
	}
	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = user.Email
	}
	adminURL := fmt.Sprintf("%s/admin/ui", base)
	subject := "You've been invited to Open Model Gateway"
	inviter := strings.TrimSpace(invitedBy)
	if inviter == "" {
		inviter = "an administrator"
	}
	body := fmt.Sprintf(
		"Hi %s,\n\n%s invited you to collaborate in Open Model Gateway. Visit %s to access the console and manage your routing configuration. Use your organization's SSO or the credentials provided by your administrator to sign in.\n",
		name, inviter, adminURL,
	)
	htmlBody, err := email.RenderAdminInviteTemplate(email.AdminInviteTemplateData{
		RecipientName: name,
		PortalURL:     adminURL,
	})
	if err != nil && s.logger != nil {
		s.logger.Error("render invite email", "error", err)
	}
	msg := email.Message{
		From:     s.emailFrom,
		To:       []string{user.Email},
		Subject:  subject,
		Body:     body,
		HTMLBody: htmlBody,
	}
	return s.emailSender.Send(ctx, msg)
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
