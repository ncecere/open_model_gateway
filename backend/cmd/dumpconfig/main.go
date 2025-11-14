package main

import (
	"log"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func main() {
	cfg, err := config.Load(config.Options{ConfigFile: "../deploy/router.local.yaml"})
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	for _, entry := range cfg.ModelCatalog {
		if entry.Provider == "vertex" {
			log.Printf("vertex overrides: %+v", entry.ProviderOverrides.Vertex)
		}
	}
}
