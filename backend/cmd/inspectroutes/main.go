package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
	"github.com/ncecere/open_model_gateway/backend/internal/runtime"
)

func main() {
	ctx := context.Background()
	rt, err := runtime.New(ctx, runtime.Options{
		Config:         runtimeConfigOptions(),
		SkipMigrations: true,
	})
	if err != nil {
		log.Fatalf("init runtime: %v", err)
	}
	defer rt.Shutdown(ctx)

	container := rt.Container
	if container == nil || container.Queries == nil {
		log.Fatalf("runtime container missing queries")
	}
	rows, err := container.Queries.ListModelCatalog(ctx)
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	entries, err := router.MergeEntries(rt.Config.ModelCatalog, rows)
	if err != nil {
		log.Fatalf("merge: %v", err)
	}
	for _, entry := range entries {
		if entry.Provider == "vertex" {
			data := []byte(entry.ProviderOverrides.Vertex.CredentialsJSON)
			valid := json.Valid(data)
			if !valid {
				log.Printf("raw creds: %q", data)
				var tmp map[string]any
				if err := json.Unmarshal(data, &tmp); err != nil {
					log.Printf("unmarshal err: %v", err)
				}
				log.Printf("merged entry overrides: %+v metadata: %+v jsonValid=%v", entry.ProviderOverrides.Vertex, entry.Metadata, valid)
				continue
			}
			if strings.EqualFold(entry.ProviderOverrides.Vertex.CredentialsFormat, "base64") {
				decoded, err := base64.StdEncoding.DecodeString(entry.ProviderOverrides.Vertex.CredentialsJSON)
				if err != nil {
					log.Printf("decode err: %v", err)
				} else {
					log.Printf("decoded json valid=%v", json.Valid(decoded))
				}
			}
			log.Printf("merged entry overrides: %+v metadata: %+v jsonValid=%v", entry.ProviderOverrides.Vertex, entry.Metadata, valid)
		}
	}
}

func runtimeConfigOptions() config.Options {
	if cfgPath := os.Getenv("ROUTER_CONFIG_FILE"); strings.TrimSpace(cfgPath) != "" {
		return config.Options{ConfigFile: cfgPath}
	}
	return config.Options{ConfigFile: "../deploy/router.local.yaml"}
}
