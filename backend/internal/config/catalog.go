package config

// PricingTier defines a pricing tier for a model.
type PricingTier struct {
	Unit         string            `mapstructure:"unit" json:"unit"`
	MaxUnits     *float64          `mapstructure:"max_units" json:"max_units,omitempty"`
	PricePerUnit float64           `mapstructure:"price_per_unit" json:"price_per_unit"`
	Metadata     map[string]string `mapstructure:"metadata" json:"metadata"`
}

// PricingTiers maps modality buckets (input/output/audio/image/etc.) to ordered tier slices.
type PricingTiers map[string][]PricingTier

// ModelCatalogEntry defines a model in the catalog.
type ModelCatalogEntry struct {
	Alias             string            `mapstructure:"alias"`
	Provider          string            `mapstructure:"provider"`
	ProviderModel     string            `mapstructure:"provider_model"`
	ModelType         string            `mapstructure:"model_type"`
	ContextWindow     int32             `mapstructure:"context_window"`
	MaxOutputTokens   int32             `mapstructure:"max_output_tokens"`
	Modalities        []string          `mapstructure:"modalities"`
	SupportsTools     bool              `mapstructure:"supports_tools"`
	Enabled           *bool             `mapstructure:"enabled"`
	Deployment        string            `mapstructure:"deployment"`
	Endpoint          string            `mapstructure:"endpoint"`
	APIKey            string            `mapstructure:"api_key"`
	APIVersion        string            `mapstructure:"api_version"`
	Region            string            `mapstructure:"region"`
	Weight            int               `mapstructure:"weight"`
	Metadata          map[string]string `mapstructure:"metadata"`
	PricingTiers      PricingTiers      `mapstructure:"pricing_tiers"`
	ProviderOverrides `mapstructure:",squash"`
	PriceInput        float64 `mapstructure:"price_input"`
	PriceOutput       float64 `mapstructure:"price_output"`
	Currency          string  `mapstructure:"currency"`
}

// IsEnabled returns whether this model is enabled.
func (e ModelCatalogEntry) IsEnabled() bool {
	if e.Enabled == nil {
		return true
	}
	return *e.Enabled
}
