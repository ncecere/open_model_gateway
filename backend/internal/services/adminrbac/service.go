package adminrbac

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
)

// Service provides RBAC helpers for admin surfaces.
type Service struct {
	repo rbac.Repository
}

// NewService builds an admin RBAC service.
// Deprecated: Use NewServiceWithRepository for new code.
func NewService(queries *db.Queries) *Service {
	return NewServiceWithRepository(rbac.NewQueriesRepository(queries))
}

// NewServiceWithRepository builds an admin RBAC service with a Repository interface.
func NewServiceWithRepository(repo rbac.Repository) *Service {
	return &Service{repo: repo}
}

var (
	ErrUnauthorized = errors.New("missing admin context")
	ErrForbidden    = errors.New("insufficient permissions")
)

// RequireTenantRole ensures the user has the specified role on the tenant.
func (s *Service) RequireTenantRole(ctx context.Context, tenantID, userID uuid.UUID, role db.MembershipRole, superAdmin bool) error {
	if superAdmin {
		return nil
	}
	if s == nil || s.repo == nil {
		return errors.New("rbac service not initialized")
	}
	_, err := rbac.EnsureWithRepo(ctx, s.repo, tenantID, userID, role)
	if err != nil {
		if err == rbac.ErrForbidden || err == pgx.ErrNoRows {
			return ErrForbidden
		}
		return err
	}
	return nil
}

// RequireAnyRole checks user has at least given role in any tenant.
func (s *Service) RequireAnyRole(ctx context.Context, userID uuid.UUID, role db.MembershipRole, superAdmin bool) error {
	if superAdmin {
		return nil
	}
	if s == nil || s.repo == nil {
		return errors.New("rbac service not initialized")
	}
	_, err := rbac.EnsureAnyWithRepo(ctx, s.repo, userID, role)
	if err != nil {
		if err == rbac.ErrForbidden || err == pgx.ErrNoRows {
			return ErrForbidden
		}
		return err
	}
	return nil
}
