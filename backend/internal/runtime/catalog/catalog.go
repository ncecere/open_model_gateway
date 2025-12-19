package catalog

import (
	"context"
	"encoding/json"
	"strings"

	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/catalog"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// EnsurePersisted upserts the merged catalog entries into the database so the
// runtime has a durable view of provider metadata. It is safe to call on every
// startup.
func EnsurePersisted(ctx context.Context, queries *db.Queries, entries []config.ModelCatalogEntry) error {
	if queries == nil {
		return nil
	}
	for _, entry := range entries {
		modalitiesJSON, err := json.Marshal(entry.Modalities)
		if err != nil {
			return err
		}
		metadataJSON, err := json.Marshal(entry.Metadata)
		if err != nil {
			return err
		}
		pricingTiers := entry.PricingTiers
		if pricingTiers == nil {
			pricingTiers = config.PricingTiers{}
		}
		pricingJSON, err := json.Marshal(pricingTiers)
		if err != nil {
			return err
		}
		providerCfgJSON, err := json.Marshal(entry.ProviderOverrides)
		if err != nil {
			return err
		}

		priceInput := decimal.NewFromFloat(entry.PriceInput)
		priceOutput := decimal.NewFromFloat(entry.PriceOutput)
		if priceInput.IsNegative() {
			priceInput = decimal.Zero
		}
		if priceOutput.IsNegative() {
			priceOutput = decimal.Zero
		}
		currency := entry.Currency
		if strings.TrimSpace(currency) == "" {
			currency = "USD"
		}

		provider := catalog.NormalizeProviderSlug(entry.Provider)
		modelType := catalog.NormalizeModelType(entry.ModelType)
		if modelType == "" {
			modelType = "llm"
		}

		_, err = queries.UpsertModelCatalogEntry(ctx, db.UpsertModelCatalogEntryParams{
			Alias:              entry.Alias,
			Provider:           provider,
			ProviderModel:      entry.ProviderModel,
			ModelType:          modelType,
			ContextWindow:      entry.ContextWindow,
			MaxOutputTokens:    entry.MaxOutputTokens,
			ModalitiesJson:     modalitiesJSON,
			SupportsTools:      entry.SupportsTools,
			PricingTiersJson:   pricingJSON,
			PriceInput:         priceInput,
			PriceOutput:        priceOutput,
			Currency:           currency,
			Enabled:            entry.IsEnabled(),
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
			return err
		}
	}
	return nil
}
