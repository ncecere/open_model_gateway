package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/database"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
)

func main() {
	cfg, err := config.Load(config.Options{ConfigFile: "../deploy/router.local.yaml"})
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	queries := db.New(pool)
	rows, err := queries.ListModelCatalog(ctx)
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	entries, err := router.MergeEntries(cfg.ModelCatalog, rows)
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
