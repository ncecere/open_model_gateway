// Package testutil provides shared utilities for integration testing.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	_, testutilFile, _, _ = runtime.Caller(0)
	backendRoot           = filepath.Clean(filepath.Join(filepath.Dir(testutilFile), "..", ".."))
	// SchemaDir is the path to SQL schema files.
	SchemaDir = filepath.Join(backendRoot, "sql", "schema")
	// MigrationsDir is the path to migration files.
	MigrationsDir = filepath.Join(backendRoot, "migrations")
)

// PostgresConfig holds configuration for test Postgres instance.
type PostgresConfig struct {
	Username string
	Password string
	Database string
}

// DefaultPostgresConfig returns default test Postgres configuration.
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Username: "postgres",
		Password: "postgres",
		Database: "omg_test",
	}
}

// StartTestPostgres starts an embedded Postgres instance for testing.
// Returns a cleanup function and the connection DSN.
func StartTestPostgres(t *testing.T) (cleanup func(), dsn string) {
	t.Helper()
	return StartTestPostgresWithConfig(t, DefaultPostgresConfig())
}

// StartTestPostgresWithConfig starts an embedded Postgres with custom config.
func StartTestPostgresWithConfig(t *testing.T, cfg PostgresConfig) (cleanup func(), dsn string) {
	t.Helper()
	baseDir := t.TempDir()
	port := FreePort(t)

	pgCfg := embeddedpostgres.DefaultConfig().
		DataPath(filepath.Join(baseDir, "data")).
		RuntimePath(filepath.Join(baseDir, "runtime")).
		BinariesPath(filepath.Join(baseDir, "bin")).
		Username(cfg.Username).
		Password(cfg.Password).
		Database(cfg.Database).
		Port(uint32(port))

	db := embeddedpostgres.NewDatabase(pgCfg)
	if err := db.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}

	dsn = fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable",
		cfg.Username, cfg.Password, port, cfg.Database)

	return func() { _ = db.Stop() }, dsn
}

// FreePort returns an available TCP port.
func FreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// ApplySchema applies all SQL schema files to the database.
func ApplySchema(ctx context.Context, dbURL string) error {
	entries, err := os.ReadDir(SchemaDir)
	if err != nil {
		return fmt.Errorf("read schema dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	sqlDB, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("open schema database: %w", err)
	}
	defer sqlDB.Close()

	for _, name := range files {
		path := filepath.Join(SchemaDir, name)
		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		contentStr := string(contents)

		// Handle trigger function creation specially for 000_init.sql
		if name == "000_init.sql" {
			if err := ensureTriggerFunction(ctx, sqlDB, contentStr); err != nil {
				return err
			}
		}

		if _, err := sqlDB.ExecContext(ctx, contentStr); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
	}
	return nil
}

func ensureTriggerFunction(ctx context.Context, db *sql.DB, content string) error {
	const marker = "CREATE OR REPLACE FUNCTION trigger_set_timestamp()"
	idx := strings.Index(content, marker)
	if idx == -1 {
		return nil
	}
	rest := content[idx:]
	endMarker := "$func$ LANGUAGE plpgsql;"
	endIdx := strings.Index(rest, endMarker)
	if endIdx == -1 {
		return nil
	}
	endIdx += len(endMarker)
	blk := rest[:endIdx]
	if _, err := db.ExecContext(ctx, blk); err != nil {
		return fmt.Errorf("init trigger function: %w", err)
	}
	return nil
}
