package tenants

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// AccessStore manages tenant model permissions.
type AccessStore struct {
	mu   sync.RWMutex
	data map[uuid.UUID]map[string]struct{}
}

func NewAccessStore(initial map[uuid.UUID]map[string]struct{}) *AccessStore {
	if initial == nil {
		initial = make(map[uuid.UUID]map[string]struct{})
	}
	return &AccessStore{data: initial}
}

// LoadModelAccess builds a store seeded from the database.
func LoadModelAccess(ctx context.Context, queries *db.Queries) (*AccessStore, error) {
	result := make(map[uuid.UUID]map[string]struct{})
	if queries == nil {
		return NewAccessStore(result), nil
	}
	tenantRows, err := queries.ListTenants(ctx, db.ListTenantsParams{Limit: math.MaxInt32, Offset: 0})
	if err != nil {
		return nil, err
	}
	for _, row := range tenantRows {
		id, err := uuidFromPg(row.ID)
		if err != nil {
			continue
		}
		result[id] = make(map[string]struct{})
	}
	rows, err := queries.ListAllTenantModels(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if !row.TenantID.Valid {
			continue
		}
		tenantID, err := uuid.FromBytes(row.TenantID.Bytes[:])
		if err != nil {
			continue
		}
		alias := normalizeAlias(row.Alias)
		if alias == "" {
			continue
		}
		set := result[tenantID]
		if set == nil {
			set = make(map[string]struct{})
			result[tenantID] = set
		}
		set[alias] = struct{}{}
	}
	return NewAccessStore(result), nil
}

// IsAllowed reports whether a tenant can use the given alias.
func (s *AccessStore) IsAllowed(tenantID uuid.UUID, alias string) bool {
	if s == nil {
		return true
	}
	normalized := normalizeAlias(alias)
	if normalized == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.data) == 0 {
		return true
	}
	allowed, ok := s.data[tenantID]
	if !ok {
		return true
	}
	if len(allowed) == 0 {
		return false
	}
	_, exists := allowed[normalized]
	return exists
}

// Set replaces the allowed aliases for a tenant.
func (s *AccessStore) Set(tenantID uuid.UUID, aliases []string) {
	if s == nil {
		return
	}
	set := make(map[string]struct{})
	for _, alias := range aliases {
		if norm := normalizeAlias(alias); norm != "" {
			set[norm] = struct{}{}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[uuid.UUID]map[string]struct{})
	}
	s.data[tenantID] = set
}

// Clear removes all aliases for a tenant.
func (s *AccessStore) Clear(tenantID uuid.UUID) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, tenantID)
}

func normalizeAlias(alias string) string {
	return strings.ToLower(strings.TrimSpace(alias))
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}
