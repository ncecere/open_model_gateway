package main

import (
	"context"
	"encoding/json"
	"log"

	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/database"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
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

	q := db.New(pool)

	for _, entry := range cfg.ModelCatalog {
		modalitiesJSON, err := json.Marshal(entry.Modalities)
		if err != nil {
			log.Fatalf("marshal modalities for %s: %v", entry.Alias, err)
		}
		metadataJSON, err := json.Marshal(entry.Metadata)
		if err != nil {
			log.Fatalf("marshal metadata for %s: %v", entry.Alias, err)
		}
		providerCfgJSON, err := json.Marshal(entry.ProviderOverrides)
		if err != nil {
			log.Fatalf("marshal provider overrides for %s: %v", entry.Alias, err)
		}

		enabled := entry.IsEnabled()
		priceInput := decimal.NewFromFloat(entry.PriceInput)
		priceOutput := decimal.NewFromFloat(entry.PriceOutput)
		if priceInput.IsNegative() {
			priceInput = decimal.Zero
		}
		if priceOutput.IsNegative() {
			priceOutput = decimal.Zero
		}
		currency := entry.Currency
		if currency == "" {
			currency = "USD"
		}

		_, err = q.UpsertModelCatalogEntry(ctx, db.UpsertModelCatalogEntryParams{
			Alias:              entry.Alias,
			Provider:           entry.Provider,
			ProviderModel:      entry.ProviderModel,
			ContextWindow:      entry.ContextWindow,
			MaxOutputTokens:    entry.MaxOutputTokens,
			ModalitiesJson:     modalitiesJSON,
			SupportsTools:      entry.SupportsTools,
			PriceInput:         priceInput,
			PriceOutput:        priceOutput,
			Currency:           currency,
			Enabled:            enabled,
			Deployment:         entry.Deployment,
			Endpoint:           entry.Endpoint,
			ApiKey:             entry.APIKey,
			ApiVersion:         entry.APIVersion,
			Region:             entry.Region,
			MetadataJson:       metadataJSON,
			Weight:             int32(entry.Weight),
			ProviderConfigJson: providerCfgJSON,
		})
		if err != nil {
			log.Fatalf("upsert %s: %v", entry.Alias, err)
		}
		log.Printf("seeded %s", entry.Alias)
	}
}
