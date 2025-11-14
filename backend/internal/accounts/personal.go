package accounts

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// PersonalService manages per-user tenants and ownership records.
type PersonalService struct {
	pool            *pgxpool.Pool
	queries         *db.Queries
	setTenantModels func(uuid.UUID, []string)
}

// NewPersonalService returns a helper for managing personal tenants.
func NewPersonalService(pool *pgxpool.Pool, queries *db.Queries) *PersonalService {
	return &PersonalService{pool: pool, queries: queries}
}

// SetTenantModelUpdater registers a callback invoked whenever personal tenants change.
func (s *PersonalService) SetTenantModelUpdater(cb func(uuid.UUID, []string)) {
	s.setTenantModels = cb
}

// EnsurePersonalTenant guarantees that the provided user has a dedicated personal tenant
// and owner membership. It returns the (possibly updated) user record and the tenant.
func (s *PersonalService) EnsurePersonalTenant(ctx context.Context, user db.User) (db.User, db.Tenant, error) {
	if s == nil || s.pool == nil || s.queries == nil {
		return user, db.Tenant{}, errors.New("personal service not initialised")
	}

	if user.PersonalTenantID.Valid {
		tenant, err := s.queries.GetTenantByID(ctx, user.PersonalTenantID)
		if err == nil {
			return user, tenant, nil
		}
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return user, db.Tenant{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	refreshed, err := qtx.GetUserByID(ctx, user.ID)
	if err != nil {
		return user, db.Tenant{}, fmt.Errorf("reload user: %w", err)
	}
	user = refreshed
	if user.PersonalTenantID.Valid {
		tenant, err := qtx.GetTenantByID(ctx, user.PersonalTenantID)
		if err != nil {
			return user, db.Tenant{}, fmt.Errorf("load personal tenant: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return user, db.Tenant{}, fmt.Errorf("commit tx: %w", err)
		}
		return user, tenant, nil
	}

	userUUID, err := toUUID(user.ID)
	if err != nil {
		return user, db.Tenant{}, fmt.Errorf("user id: %w", err)
	}
	tenantName := fmt.Sprintf("personal:%s", userUUID.String())

	tenant, err := qtx.CreateTenant(ctx, db.CreateTenantParams{
		Name:   tenantName,
		Status: db.TenantStatusActive,
		Kind:   db.TenantKindPersonal,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			tenant, err = qtx.GetTenantByName(ctx, tenantName)
			if err != nil {
				return user, db.Tenant{}, fmt.Errorf("lookup personal tenant: %w", err)
			}
		} else {
			return user, db.Tenant{}, fmt.Errorf("create personal tenant: %w", err)
		}
	}

	user, err = qtx.UpdateUserPersonalTenant(ctx, db.UpdateUserPersonalTenantParams{
		ID:               user.ID,
		PersonalTenantID: tenant.ID,
	})
	if err != nil {
		return user, db.Tenant{}, fmt.Errorf("link user personal tenant: %w", err)
	}

	if _, err := qtx.AddTenantMembership(ctx, db.AddTenantMembershipParams{
		TenantID: tenant.ID,
		UserID:   user.ID,
		Role:     db.MembershipRoleOwner,
	}); err != nil {
		var pgErr *pgconn.PgError
		if !(errors.As(err, &pgErr) && pgErr.Code == "23505") {
			return user, db.Tenant{}, fmt.Errorf("ensure personal membership: %w", err)
		}
	}

	aliases, err := s.seedDefaultModels(ctx, qtx, tenant.ID)
	if err != nil {
		return user, db.Tenant{}, err
	}
	if s.setTenantModels != nil {
		if tenantUUID, err := toUUID(tenant.ID); err == nil {
			s.setTenantModels(tenantUUID, aliases)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return user, db.Tenant{}, fmt.Errorf("commit tx: %w", err)
	}

	return user, tenant, nil
}

func toUUID(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

func (s *PersonalService) seedDefaultModels(ctx context.Context, q *db.Queries, tenantID pgtype.UUID) ([]string, error) {
	aliases, err := q.ListDefaultModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list default models: %w", err)
	}
	return s.replaceTenantModels(ctx, q, tenantID, normalizeAliases(aliases))
}

// SyncDefaultModels reapplies the current default model list to every personal tenant.
func (s *PersonalService) SyncDefaultModels(ctx context.Context) error {
	if s == nil || s.pool == nil || s.queries == nil {
		return errors.New("personal service not initialised")
	}
	aliases, err := s.queries.ListDefaultModels(ctx)
	if err != nil {
		return fmt.Errorf("list default models: %w", err)
	}
	normalized := normalizeAliases(aliases)
	tenants, err := s.queries.ListPersonalTenantIDs(ctx)
	if err != nil {
		return fmt.Errorf("list personal tenants: %w", err)
	}
	for _, row := range tenants {
		tenantUUID, err := toUUID(row)
		if err != nil {
			return fmt.Errorf("tenant uuid: %w", err)
		}
		if err := s.syncTenantModels(ctx, tenantUUID, normalized); err != nil {
			return err
		}
	}
	return nil
}

func (s *PersonalService) syncTenantModels(ctx context.Context, tenantID uuid.UUID, aliases []string) error {
	if s.pool == nil || s.queries == nil {
		return errors.New("personal service not initialised")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	pgID := pgtype.UUID{Bytes: tenantID, Valid: true}
	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteTenantModels(ctx, pgID); err != nil {
		return fmt.Errorf("clear tenant models: %w", err)
	}
	if _, err := s.replaceTenantModels(ctx, qtx, pgID, aliases); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tenant model sync: %w", err)
	}
	if s.setTenantModels != nil {
		s.setTenantModels(tenantID, aliases)
	}
	return nil
}

func (s *PersonalService) replaceTenantModels(ctx context.Context, q *db.Queries, tenantID pgtype.UUID, aliases []string) ([]string, error) {
	inserted := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		if alias == "" {
			continue
		}
		if err := q.InsertTenantModel(ctx, db.InsertTenantModelParams{
			TenantID: tenantID,
			Alias:    alias,
		}); err != nil {
			return nil, fmt.Errorf("assign default model %s: %w", alias, err)
		}
		inserted = append(inserted, alias)
	}
	return inserted, nil
}

func normalizeAliases(aliases []string) []string {
	if len(aliases) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		norm := strings.TrimSpace(strings.ToLower(alias))
		if norm != "" {
			set[norm] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for alias := range set {
		result = append(result, alias)
	}
	sort.Strings(result)
	return result
}
