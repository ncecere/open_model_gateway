# Pricing Tiers Plan

Notes capturing requirements and design ideas for tiered pricing support in the Open Model Gateway.

## Goals

- Support models whose unit price changes after a token threshold (e.g., Gemini 3 Pro: `< 200k` vs `> 200k` token brackets).
- Reuse the same structure across prompt, completion/output, and context cache pricing, plus future modalities (audio/images).
- Keep backward compatibility with existing single-price fields until the rollout is complete.

## Data Model

1. Extend `model_catalog` with a JSON column (`pricing_tiers_json`) or a child table storing ordered tiers.
2. Each tier entry contains:
   - `type`: `input`, `output`, `cache`, `image`, `audio`, `video` (extensible list that mirrors model modalities).
   - `max_tokens`: inclusive upper bound; `null` or missing means “no upper limit”. For non-token workloads this becomes the native unit threshold (e.g., image pixels, audio minutes).
   - `price_per_million`: USD cost per 1M tokens **or** per chosen unit.
   - `unit` (optional): overrides the default token meaning; use values like `per_image`, `per_megapixel`, `per_minute`, `per_character`. Lets us describe image/audio/video billing where vendors charge per asset/minute rather than tokens.
   - Optional `metadata` for dimension hints (e.g., image resolutions, audio sample rate).
3. Migration seeds a default tier using existing `price_input`/`price_output` values so current models keep working.

Example storage shape (JSON):

```json
{
  "input": [
    {"max_tokens": 200000, "price_per_million": 2.0},
    {"max_tokens": null, "price_per_million": 4.0}
  ],
  "output": [
    {"max_tokens": 200000, "price_per_million": 12.0},
    {"max_tokens": null, "price_per_million": 18.0}
  ],
  "cache": [
    {"max_tokens": 200000, "price_per_million": 0.2},
    {"max_tokens": null, "price_per_million": 0.4}
  ],
  "image": [
    {"max_tokens": null, "price_per_million": 40.0, "unit": "per_image"}
  ],
  "audio": [
    {"max_tokens": 60, "price_per_million": 0.0, "unit": "per_minute"},
    {"max_tokens": null, "price_per_million": 2.0, "unit": "per_minute"}
  ]
}
```

## Config + Admin API

- **YAML/bootstrap**: Add `pricing_tiers` block under each model. Example:

  ```yaml
  pricing_tiers:
    input:
      - max_tokens: 200000
        price_per_million: 2.00
      - max_tokens: null
        price_per_million: 4.00
    output:
      - max_tokens: 200000
        price_per_million: 12.00
      - max_tokens: null
        price_per_million: 18.00
    image:
      - max_tokens: null
        price_per_million: 40.00
        unit: per_image
    audio:
      - max_tokens: 60
        price_per_million: 0.00
        unit: per_minute
      - max_tokens: null
        price_per_million: 2.00
        unit: per_minute
  ```

- **Admin API payload**: Embed the same structure inside `ModelPayload`. Validation:
  - At least one tier per type if provided.
  - Ascending `max_tokens`.
  - Optional `max_tokens: null` for infinity; last tier must cover the tail.
  - If tiers missing, fall back to legacy scalar prices.
- **Admin UI**: 
  - Pricing section lists tiers in a table with “Add tier” action.
  - Catalog list surfaces whether tiered pricing is active (badge or tooltip).

## Billing Logic

- Centralize tier selection in a helper: `selectTier(priceType, tokens, tiers)`.
  - Default to first tier if no thresholds or tokens <= tier threshold.
  - For streaming/batch flows reuse the same helper so SSE or background jobs stay consistent.
- Update `usage recorder` to pass actual token counts and compute USD micros with the resolved `price_per_million`.
- Budget enforcement uses the same computed cost, so no additional changes needed beyond verifying new tests.

### Edge Cases

- Requests that land exactly on the threshold (e.g., 200,000 tokens) should use the tier where `tokens <= max_tokens`.
- Missing tokens (e.g., provider skips output token count) → fallback to default tier or mark cost as zero.
- Data migration ensures routes referencing models without tiers still behave the same.

## Testing

- Unit tests for tier selection covering:
  - Single-tier (null threshold) models (`gpt-5` example).
  - Multi-tier with prompt > threshold and completion < threshold.
  - Missing tier definitions (should use legacy price fields).
- Integration tests for `/v1/chat/completions` and `/v1/batches` verifying recorded `cost_usd_micros`.
- Admin UI tests for tier editor (add/remove/reorder) and config bootstrap parsing.

## Documentation

- Update `docs/runtime/config.md` with the new `pricing_tiers` block.
- Add migration notes + `CHANGELOG.md` entry once shipped.
- Provide sample configs for Gemini, Claude cache pricing, and simple single-tier models (`gpt-5`).

## Open Questions

- How many modality-specific helpers do we need? For example, image tiers might need pixel metadata while audio tiers may reference duration in seconds or transcribed token count.
- Do we allow tier reuse (e.g., same thresholds for input/output) to avoid duplication? For now, keep separate arrays for clarity.
- Need to surface tier info in the admin UI usage summary? Could add optional “tiered pricing” badge or display the active tier in cost detail views.
