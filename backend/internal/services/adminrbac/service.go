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
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
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
	if s == nil || s.queries == nil {
		return errors.New("rbac service not initialized")
	}
	_, err := rbac.Ensure(ctx, s.queries, tenantID, userID, role)
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
	if s == nil || s.queries == nil {
		return errors.New("rbac service not initialized")
	}
	_, err := rbac.EnsureAny(ctx, s.queries, userID, role)
	if err != nil {
		if err == rbac.ErrForbidden || err == pgx.ErrNoRows {
			return ErrForbidden
		}
		return err
	}
	return nil
}
