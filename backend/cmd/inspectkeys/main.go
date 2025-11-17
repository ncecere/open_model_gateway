package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/runtime"
)

func main() {
	ctx := context.Background()
	rt, err := runtime.New(ctx, runtime.Options{
		Config:         runtimeConfigOptions(),
		SkipMigrations: true,
	})
	if err != nil {
		panic(err)
	}
	defer rt.Shutdown(ctx)

	pool := rt.Container.DBPool
	if pool == nil {
		panic("runtime container missing DB pool")
	}

	rows, err := pool.Query(ctx, `SELECT id, prefix, tenant_id, owner_user_id FROM api_keys WHERE prefix = ANY($1)`, []string{"bnmXdiCFPq", "RRZrw1Kdkg"})
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, tenantID, ownerID pgtype.UUID
		var prefix string
		if err := rows.Scan(&id, &prefix, &tenantID, &ownerID); err != nil {
			panic(err)
		}
		fmt.Printf("id=%s prefix=%s tenant=%s owner=%s\n", formatUUID(id), prefix, formatUUID(tenantID), formatUUID(ownerID))
	}
	if rows.Err() != nil {
		panic(rows.Err())
	}
}

func runtimeConfigOptions() config.Options {
	if cfgPath := os.Getenv("ROUTER_CONFIG_FILE"); strings.TrimSpace(cfgPath) != "" {
		return config.Options{ConfigFile: cfgPath}
	}
	return config.Options{ConfigFile: "../deploy/router.local.yaml"}
}

func formatUUID(id pgtype.UUID) string {
	if !id.Valid {
		return "<invalid>"
	}
	return fmt.Sprintf("%x", id.Bytes)
}
