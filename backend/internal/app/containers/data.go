package containers

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// DataContainer holds data access primitives (database, cache, queries).
type DataContainer struct {
	DBPool  *pgxpool.Pool
	Redis   *redis.Client
	Queries *db.Queries
}

// NewDataContainer creates a new DataContainer with the given primitives.
func NewDataContainer(pool *pgxpool.Pool, redisClient *redis.Client) *DataContainer {
	return &DataContainer{
		DBPool:  pool,
		Redis:   redisClient,
		Queries: db.New(pool),
	}
}
