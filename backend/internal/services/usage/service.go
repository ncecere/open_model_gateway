package usage

import (
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Service exposes usage aggregation helpers shared across admin and user surfaces.
type Service struct {
	repo     Repository
	timezone *time.Location
}

// NewService builds a usage service.
// Deprecated: Use NewServiceWithRepository for new code.
func NewService(queries *db.Queries, timezone *time.Location) *Service {
	return NewServiceWithRepository(NewQueriesRepository(queries), timezone)
}

// NewServiceWithRepository builds a usage service with a Repository interface.
func NewServiceWithRepository(repo Repository, timezone *time.Location) *Service {
	return &Service{repo: repo, timezone: timezone}
}
