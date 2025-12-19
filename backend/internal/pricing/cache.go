package pricing

import (
	"sort"
	"strings"
	"sync"

	"github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

const (
	TierInputKey       = "input"
	TierOutputKey      = "output"
	TierCachedInputKey = "cache"
)

// Unit is the billing unit identifier used in pricing tiers.
type Unit string

const (
	UnitTokensPerMillion     Unit = "tokens_per_million"
	UnitTokensPerThousand    Unit = "tokens_per_thousand"
	UnitPerImage             Unit = "per_image"
	UnitPerMegapixel         Unit = "per_megapixel"
	UnitPerSecond            Unit = "per_second"
	UnitPerMinute            Unit = "per_minute"
	UnitPerMillionCharacters Unit = "per_million_characters"
)

// Params captures the measurable usage for calculating cost.
type Params struct {
	PromptTokens       int64
	CompletionTokens   int64
	CachedPromptTokens int64
	Metadata           map[string]string
}

// Cache stores pricing metadata indexed by model alias.
type Cache struct {
	mu     sync.RWMutex
	models map[string]ModelPrice
}

// NewCache constructs an empty pricing cache.
func NewCache() *Cache {
	return &Cache{models: make(map[string]ModelPrice)}
}

// Load replaces the cache contents with the provided entries.
func (c *Cache) Load(entries []config.ModelCatalogEntry) {
	models := make(map[string]ModelPrice, len(entries))
	for _, entry := range entries {
		models[entry.Alias] = newModelPrice(entry)
	}

	c.mu.Lock()
	c.models = models
	c.mu.Unlock()
}

// Cost returns the USD cost for the given alias and usage parameters.
func (c *Cache) Cost(alias string, params Params) decimal.Decimal {
	c.mu.RLock()
	model, ok := c.models[alias]
	c.mu.RUnlock()
	if !ok {
		return decimal.Zero
	}
	return model.Cost(params)
}

// ModelPrice represents the pricing configuration for a single model alias.
type ModelPrice struct {
	Currency    string
	InputPrice  decimal.Decimal
	OutputPrice decimal.Decimal
	CachePrice  decimal.Decimal
	tiers       map[string][]tier
}

type tier struct {
	unit     Unit
	maxUnits *decimal.Decimal
	price    decimal.Decimal
	metadata map[string]string
}

func newModelPrice(entry config.ModelCatalogEntry) ModelPrice {
	mp := ModelPrice{
		Currency:    strings.ToUpper(strings.TrimSpace(entry.Currency)),
		InputPrice:  decimalFromFloat(entry.PriceInput),
		OutputPrice: decimalFromFloat(entry.PriceOutput),
		CachePrice:  decimal.Zero,
		tiers:       make(map[string][]tier),
	}
	if mp.Currency == "" {
		mp.Currency = "USD"
	}

	for bucket, tiers := range entry.PricingTiers {
		if len(tiers) == 0 {
			continue
		}
		normalized := make([]tier, 0, len(tiers))
		for _, t := range tiers {
			unit := Unit(strings.ToLower(strings.TrimSpace(t.Unit)))
			if unit == "" {
				unit = UnitTokensPerMillion
			}
			var maxPtr *decimal.Decimal
			if t.MaxUnits != nil {
				value := decimalFromFloat(*t.MaxUnits)
				maxPtr = &value
			}
			metadata := make(map[string]string, len(t.Metadata))
			for k, v := range t.Metadata {
				metadata[strings.ToLower(k)] = v
			}
			normalized = append(normalized, tier{
				unit:     unit,
				maxUnits: maxPtr,
				price:    decimalFromFloat(t.PricePerUnit),
				metadata: metadata,
			})
		}
		if len(normalized) == 0 {
			continue
		}
		sort.SliceStable(normalized, func(i, j int) bool {
			left := normalized[i].maxUnits
			right := normalized[j].maxUnits
			if left == nil {
				return false
			}
			if right == nil {
				return true
			}
			return left.LessThan(*right)
		})
		key := strings.ToLower(bucket)
		mp.tiers[key] = normalized
	}

	return mp
}

// Cost calculates the USD cost for the provided usage parameters.
func (m ModelPrice) Cost(params Params) decimal.Decimal {
	total := decimal.Zero

	if params.PromptTokens > 0 {
		total = total.Add(m.costTokens(TierInputKey, params.PromptTokens, m.InputPrice))
	}
	if params.CompletionTokens > 0 {
		total = total.Add(m.costTokens(TierOutputKey, params.CompletionTokens, m.OutputPrice))
	}
	if params.CachedPromptTokens > 0 {
		total = total.Add(m.costTokens(TierCachedInputKey, params.CachedPromptTokens, m.CachePrice))
	}

	return total
}

func (m ModelPrice) costTokens(bucket string, quantity int64, fallback decimal.Decimal) decimal.Decimal {
	if quantity <= 0 {
		return decimal.Zero
	}

	qty := decimal.NewFromInt(quantity)
	tiers := m.tiers[strings.ToLower(bucket)]

	var billed decimal.Decimal
	cost := decimal.Zero
	if len(tiers) > 0 {
		cost, billed = applyTiers(tiers, qty)
	}

	remaining := qty.Sub(billed)
	if remaining.Sign() > 0 && !fallback.IsZero() {
		cost = cost.Add(applyScalarPrice(UnitTokensPerMillion, fallback, remaining))
	} else if billed.IsZero() && cost.IsZero() && !fallback.IsZero() {
		cost = cost.Add(applyScalarPrice(UnitTokensPerMillion, fallback, qty))
	}

	return cost
}

func applyTiers(tiers []tier, quantity decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if quantity.Sign() <= 0 || len(tiers) == 0 {
		return decimal.Zero, decimal.Zero
	}

	remaining := quantity
	processed := decimal.Zero
	totalCost := decimal.Zero
	billed := decimal.Zero

	for _, t := range tiers {
		if remaining.Sign() <= 0 {
			break
		}

		var tierSpan decimal.Decimal
		if t.maxUnits != nil {
			// Skip tiers whose max is below the already processed amount.
			if processed.GreaterThanOrEqual(*t.maxUnits) {
				continue
			}
			tierSpan = t.maxUnits.Sub(processed)
		} else {
			tierSpan = remaining
		}

		if tierSpan.Sign() <= 0 {
			continue
		}

		billable := tierSpan
		if billable.GreaterThan(remaining) {
			billable = remaining
		}

		if billable.Sign() <= 0 {
			continue
		}

		cost := applyScalarPrice(t.unit, t.price, billable)
		totalCost = totalCost.Add(cost)
		billed = billed.Add(billable)
		remaining = remaining.Sub(billable)

		if t.maxUnits != nil {
			processed = *t.maxUnits
		} else {
			processed = processed.Add(billable)
		}
	}

	return totalCost, billed
}

func applyScalarPrice(unit Unit, price decimal.Decimal, quantity decimal.Decimal) decimal.Decimal {
	if price.IsZero() || quantity.Sign() <= 0 {
		return decimal.Zero
	}
	scale := unitDivisor(unit)
	if scale.IsZero() {
		return decimal.Zero
	}
	return price.Mul(quantity).Div(scale)
}

func unitDivisor(unit Unit) decimal.Decimal {
	switch unit {
	case UnitTokensPerMillion:
		return decimal.NewFromInt(1_000_000)
	case UnitTokensPerThousand:
		return decimal.NewFromInt(1_000)
	case UnitPerImage, UnitPerMegapixel, UnitPerSecond, UnitPerMinute:
		return decimal.NewFromInt(1)
	case UnitPerMillionCharacters:
		return decimal.NewFromInt(1_000_000)
	default:
		return decimal.Zero
	}
}

func decimalFromFloat(v float64) decimal.Decimal {
	if v == 0 {
		return decimal.Zero
	}
	return decimal.NewFromFloat(v)
}
