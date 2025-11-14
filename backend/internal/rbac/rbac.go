package rbac

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

type RoleRank int

var roleOrder = map[db.MembershipRole]RoleRank{
	db.MembershipRoleOwner:  4,
	db.MembershipRoleAdmin:  3,
	db.MembershipRoleViewer: 2,
	db.MembershipRoleUser:   1,
}

// ParseRole converts a case-insensitive string to MembershipRole.
func ParseRole(value string) (db.MembershipRole, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "owner":
		return db.MembershipRoleOwner, true
	case "admin":
		return db.MembershipRoleAdmin, true
	case "viewer":
		return db.MembershipRoleViewer, true
	case "user":
		return db.MembershipRoleUser, true
	default:
		return "", false
	}
}

// AtLeast returns true if current role is >= required role.
func AtLeast(current, required db.MembershipRole) bool {
	return roleOrder[current] >= roleOrder[required]
}

var ErrForbidden = errors.New("forbidden")

// Ensure enforces that the user has the required role for the tenant.
func Ensure(ctx context.Context, queries *db.Queries, tenantID uuid.UUID, userID uuid.UUID, required db.MembershipRole) (db.TenantMembership, error) {
	membership, err := queries.GetTenantMembership(ctx, db.GetTenantMembershipParams{
		TenantID: pgtype.UUID{Bytes: tenantID, Valid: true},
		UserID:   pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return db.TenantMembership{}, err
	}

	if !AtLeast(membership.Role, required) {
		return db.TenantMembership{}, ErrForbidden
	}
	return membership, nil
}

// EnsureAny verifies that the user holds at least the required role for any tenant membership.
func EnsureAny(ctx context.Context, queries *db.Queries, userID uuid.UUID, required db.MembershipRole) (db.ListUserTenantsRow, error) {
	memberships, err := queries.ListUserTenants(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return db.ListUserTenantsRow{}, err
	}
	for _, membership := range memberships {
		if AtLeast(membership.Role, required) {
			return membership, nil
		}
	}
	return db.ListUserTenantsRow{}, ErrForbidden
}
