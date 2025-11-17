package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/observability"
)

func TestOpenDatastoresMissingConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "router.yaml")
	if err := os.WriteFile(cfgPath, []byte("server:\n  listen_addr: \":8080\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := OpenDatastores(context.Background(), Options{
		Config: config.Options{ConfigFile: cfgPath},
	})
	if err == nil {
		t.Fatalf("expected configuration error for missing DB/Redis settings")
	}
	if !strings.Contains(err.Error(), "missing required configuration") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeShutdownClosesResources(t *testing.T) {
	t.Parallel()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	closed := false
	ds := &Datastores{
		Config:    &config.Config{},
		closeHook: func() { closed = true },
	}

	rt := &Runtime{
		Container: &app.Container{
			Observability: &observability.Provider{},
		},
		datastores: ds,
		redis:      redisClient,
	}

	if err := rt.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if !closed {
		t.Fatalf("expected datastores close hook to run")
	}
	if err := redisClient.Ping(ctx).Err(); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("expected redis client closed error, got %v", err)
	}
}
