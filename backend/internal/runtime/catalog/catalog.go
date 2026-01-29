package catalog

import (
	"context"
	"encoding/json"

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
		// Run through the shared normalizer so config-seeded entries get the
		// same defaults (model_type inference, modalities, tool support, etc.)
		// as UI-created entries.
		result := catalog.NormalizeEntry(entry)
		e := result.Entry

		modalitiesJSON, err := json.Marshal(e.Modalities)
		if err != nil {
			return err
		}
		metadataJSON, err := json.Marshal(e.Metadata)
		if err != nil {
			return err
		}
		pricingJSON, err := json.Marshal(e.PricingTiers)
		if err != nil {
			return err
		}
		providerCfgJSON, err := json.Marshal(e.ProviderOverrides)
		if err != nil {
			return err
		}

		priceInput := decimal.NewFromFloat(e.PriceInput)
		priceOutput := decimal.NewFromFloat(e.PriceOutput)

		_, err = queries.UpsertModelCatalogEntry(ctx, db.UpsertModelCatalogEntryParams{
			Alias:              e.Alias,
			Provider:           e.Provider,
			ProviderModel:      e.ProviderModel,
			ModelType:          e.ModelType,
			ContextWindow:      e.ContextWindow,
			MaxOutputTokens:    e.MaxOutputTokens,
			ModalitiesJson:     modalitiesJSON,
			SupportsTools:      e.SupportsTools,
			PricingTiersJson:   pricingJSON,
			PriceInput:         priceInput,
			PriceOutput:        priceOutput,
			Currency:           e.Currency,
			Enabled:            e.IsEnabled(),
			Deployment:         e.Deployment,
			Endpoint:           e.Endpoint,
			ApiKey:             e.APIKey,
			ApiVersion:         e.APIVersion,
			Region:             e.Region,
			MetadataJson:       metadataJSON,
			Weight:             int32(e.Weight),
			ProviderConfigJson: providerCfgJSON,
			ManagedBy:          "config",
		})
		if err != nil {
			return err
		}
	}
	return nil
}
