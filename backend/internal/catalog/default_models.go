package catalog

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

var (
	ErrServiceUnavailable = errors.New("default model service not initialised")
	ErrAliasRequired      = errors.New("alias is required")
	ErrAliasUnknown       = errors.New("model alias not found")
)

// DefaultModelService manages global default model entitlements.
type DefaultModelService struct {
	queries *db.Queries
}

func NewDefaultModelService(queries *db.Queries) *DefaultModelService {
	return &DefaultModelService{queries: queries}
}

// List returns the sorted list of default model aliases.
func (s *DefaultModelService) List(ctx context.Context) ([]string, error) {
	if s == nil || s.queries == nil {
		return nil, ErrServiceUnavailable
	}
	return s.queries.ListDefaultModels(ctx)
}

// Add validates the alias exists in the catalog before inserting.
func (s *DefaultModelService) Add(ctx context.Context, alias string) error {
	if s == nil || s.queries == nil {
		return ErrServiceUnavailable
	}
	norm := normalizeAlias(alias)
	if norm == "" {
		return ErrAliasRequired
	}
	if _, err := s.queries.GetModelByAlias(ctx, norm); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %s", ErrAliasUnknown, norm)
		}
		return err
	}
	return s.queries.UpsertDefaultModel(ctx, norm)
}

// Remove deletes the alias from the defaults list.
func (s *DefaultModelService) Remove(ctx context.Context, alias string) error {
	if s == nil || s.queries == nil {
		return ErrServiceUnavailable
	}
	norm := normalizeAlias(alias)
	if norm == "" {
		return ErrAliasRequired
	}
	return s.queries.DeleteDefaultModel(ctx, norm)
}

func normalizeAlias(alias string) string {
	return strings.TrimSpace(strings.ToLower(alias))
}
