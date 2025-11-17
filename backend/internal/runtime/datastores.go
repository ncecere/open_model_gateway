package runtime

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/database"
)

// Datastores holds shared configuration and database connections.
type Datastores struct {
	Config *config.Config
	DB     *pgxpool.Pool
	// closeHook allows tests to observe Close() without requiring a real pool.
	closeHook func()
}

// OpenDatastores loads configuration, optionally runs migrations, and connects
// to Postgres. Call Close when finished.
func OpenDatastores(ctx context.Context, opts Options) (*Datastores, error) {
	cfg, err := config.Load(opts.Config)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if !opts.SkipMigrations {
		if err := database.RunMigrations(ctx, cfg.Database); err != nil {
			return nil, fmt.Errorf("run migrations: %w", err)
		}
	}

	pool, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return &Datastores{
		Config: cfg,
		DB:     pool,
	}, nil
}

// Close releases database resources.
func (d *Datastores) Close() {
	if d == nil {
		return
	}
	if d.closeHook != nil {
		d.closeHook()
	}
	if d.DB != nil {
		d.DB.Close()
		d.DB = nil
	}
}
