package usage

import (
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"time"
)

// Service exposes usage aggregation helpers shared across admin and user surfaces.
type Service struct {
	queries  *db.Queries
	timezone *time.Location
}

func NewService(queries *db.Queries, timezone *time.Location) *Service {
	return &Service{queries: queries, timezone: timezone}
}
