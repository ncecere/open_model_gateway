package pricing

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func TestCacheCostFallback(t *testing.T) {
	cache := NewCache()
	entry := config.ModelCatalogEntry{
		Alias:       "gpt",
		Currency:    "USD",
		PriceInput:  1.5,
		PriceOutput: 2.5,
	}
	cache.Load([]config.ModelCatalogEntry{entry})

	cost := cache.Cost("gpt", Params{PromptTokens: 500_000, CompletionTokens: 250_000})
	expected := decimal.NewFromFloat(1.5*0.5 + 2.5*0.25)
	if !cost.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected.String(), cost.String())
	}
}

func TestCacheCostWithTiers(t *testing.T) {
	cache := NewCache()
	max := 200_000.0
	entry := config.ModelCatalogEntry{
		Alias:       "gemini",
		Currency:    "USD",
		PriceInput:  5.0,
		PriceOutput: 7.0,
		PricingTiers: config.PricingTiers{
			TierInputKey: {
				{Unit: string(UnitTokensPerMillion), MaxUnits: &max, PricePerUnit: 2.0},
				{Unit: string(UnitTokensPerMillion), PricePerUnit: 4.0},
			},
		},
	}
	cache.Load([]config.ModelCatalogEntry{entry})

	params := Params{PromptTokens: 250_000, CompletionTokens: 100_000}
	cost := cache.Cost("gemini", params)

	promptExpected := decimal.NewFromFloat((200_000.0/1_000_000.0)*2.0 + (50_000.0/1_000_000.0)*4.0)
	outputExpected := decimal.NewFromFloat((100_000.0 / 1_000_000.0) * 7.0)
	expected := promptExpected.Add(outputExpected)
	if !cost.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected.String(), cost.String())
	}
}

func TestCacheLoadReplacesModels(t *testing.T) {
	cache := NewCache()
	entry := config.ModelCatalogEntry{Alias: "model", PriceInput: 1.0, PriceOutput: 1.0}
	cache.Load([]config.ModelCatalogEntry{entry})

	if cache.Cost("model", Params{PromptTokens: 1000}).Equal(decimal.Zero) {
		t.Fatalf("initial cost should be non-zero")
	}

	cache.Load(nil)
	if !cache.Cost("model", Params{PromptTokens: 1000}).Equal(decimal.Zero) {
		t.Fatalf("expected cost to be zero after reload")
	}
}
