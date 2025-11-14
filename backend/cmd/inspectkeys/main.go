package main

import (
    "context"
    "fmt"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jackc/pgx/v5/pgtype"
)

func main() {
    ctx := context.Background()
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        dbURL = "postgres://open_gateway:open_gateway@localhost:5432/open_gateway?sslmode=disable"
    }
    pool, err := pgxpool.New(ctx, dbURL)
    if err != nil {
        panic(err)
    }
    defer pool.Close()

    rows, err := pool.Query(ctx, `SELECT id, prefix, tenant_id, owner_user_id FROM api_keys WHERE prefix = ANY($1)` , []string{"bnmXdiCFPq", "RRZrw1Kdkg"})
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

func formatUUID(id pgtype.UUID) string {
    if !id.Valid {
        return "<invalid>"
    }
    return fmt.Sprintf("%x", id.Bytes)
}
