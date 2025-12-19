# Pricing Flexibility Plan

## Current State Recap
- Catalog entries (`backend/internal/config/config.go:204-225`, `backend/internal/services/admincatalog/service.go:38-221`) expose only `price_input`/`price_output` floats (USD per 1M tokens). Usage logging (`backend/internal/services/usagepipeline/logger.go:121-367`) loads these two numbers into an in-memory map to compute every request cost.
- Image endpoints bolt on metadata overrides (`price_image_cents`, `price_image_edit_cents`, `price_image_variation_cents`) which executor threads through via `OverrideCostCents` (`backend/internal/executor/executor.go:349-365`). Audio metadata (`price_audio_minute_cents`, `price_tts_million_chars_cents`) exists in config docs but is not consumed anywhere yet.
- Admin UI/editor (`backend/frontend/src/features/models/components/ModelEditorDialog.tsx:245-468`, `ModelTable.tsx:128-142`) only supports scalar input/output pricing. Usage dashboards only display the computed cents/USD returned from the backend; they have no concept of tiers or modality-specific units.
- Notes (`notes/pricing_tiers.md`, `notes/audio_pricing.md`) already describe desired tiered pricing and alternative units, but the database schema (`backend/sql/schema/000_init.sql:100-139`, `model_catalog` columns in `backend/sql/schema/000_init.sql` via generated queries) lacks any structure to store them.

## Problems To Solve
1. LLMs like Gemini 1.5/3 charge different rates above token brackets (e.g., >200k tokens). Single scalar prices cannot represent these, so budget enforcement becomes inaccurate.
2. Image/audio/video workloads bill on different units (per image, per megapixel, per minute, per character, per frame quality). Encoding them as ad-hoc metadata strings makes executor logic brittle and spreads billing rules across handlers.
3. Operators need a clear way to configure these prices (YAML + Admin UI) and audit them (usage dashboards), otherwise multi-modal routing is untrustworthy.

## Proposed Architecture

### 1. Unified Pricing Schema
- Extend catalog entries with a `pricing_tiers` map (`input`, `output`, `cache`, `image`, `audio`, `tts`, etc.). Each tier item includes `max_units` (tokens/minutes/etc), `unit` (defaults to `tokens` per 1M), `price_per_unit` (USD per million tokens or natural unit), optional `metadata` (quality/resolution tags). JSONB column `pricing_tiers_json` (and generated struct) stores this.
- Migration seeds backwards-compatible tiers from existing `price_input`/`price_output` and any `price_image*_cents` metadata. Keep legacy columns for rollout; mark them deprecated once UI/API no longer require them.

### 2. Config + Admin API + UI
- YAML/bootstrap: allow `pricing_tiers` under each model (mirrors structure from `notes/pricing_tiers.md`). Loader populates both new structure and legacy floats (for compatibility) but warns when both conflict.
- Admin API payload gains optional `pricing_tiers`; server normalizes tier ordering/units, validates ascending thresholds, and persists JSON. Admin UI adds a tier editor (table form) supporting multiple tier types plus metadata for things like image quality/resolution or audio billing source. Provide presets for common providers.
- For short term, show both scalar fallback fields and the tier builder with a banner explaining precedence (tiers override scalars).

### 3. Cost Computation
- Add helper in `usagepipeline` (e.g., `pricing.SelectPrice(alias, usage, metadata)`) that:
  - Chooses tier based on usage metrics (prompt/output tokens, cached tokens, audio duration, image dimensions, etc.).
  - Supports per-request overrides (quality/resolution) by matching tier metadata keys.
  - Returns USD w/ micros + cents for persistence. Falls back to scalar prices if no tiers exist.
- Executors/handlers populate new usage metrics (e.g., audio duration, generated image resolution, streaming token counts) so tier selection has the necessary inputs. Image/audio overrides no longer parse ad-hoc metadata; they ask the pricing helper instead.
- Update batch worker to reuse the same helper so offline jobs stay consistent.

### 4. Usage + Reporting
- Store the resolved tier + unit alongside request/usage rows (extra columns like `billing_unit`, `billing_tier`). Expose them via admin/user usage APIs so dashboards can show "Billed @ $0.04 per image (HD)" or "Tier 2 (>200k tokens)".
- Update React dashboards to surface tier info and highlight non-token billing, reducing operator confusion.

### 5. Docs + Tooling
- Refresh `docs/runtime/config.md`, `deploy/router.example.yaml`, and admin documentation to highlight tier syntax and migration guidance.
- Add changelog entry plus migration/runbook instructions (export DB, run migration, verify seeded tiers, optionally delete legacy metadata overrides).

### Metadata Reference
| Metadata key | Purpose | Classification |
| --- | --- | --- |
| `cached` | Distinguishes cached-input traffic so discounted tiers apply | Price-determining |
| `modality` | Identifies which token/channel (text vs image tokens, etc.) the tier covers | Price-determining |
| `quality` | Maps UI quality presets (low/med/high image renders) to the correct tier | Price-determining |
| `operation` | Labels the workload such as `stt`, `tts`, or `video` so handlers pick the right tier | Price-determining |
| `channel` | Specifies the billed output channel (e.g., `text_transcript`, `audio_output`) | Price-determining |
| `model` | Points to a sub-model/SKU when an alias fronts multiple provider SKUs | Price-determining |
| `resolution` | Selects pricing based on requested pixels (720p vs 4K) | Price-determining |
| `context_scope` | Human-readable explanation of when the tier applies (e.g., <=200k tokens) | Context-only |
| `approx_resolution` | Describes implied size for UI/docs (“square ~1024px”) | Context-only |
| `includes` | Notes what is bundled in the tier (“text-to-image & editing”) | Context-only |
| `usage` | Highlights policy constraints (“dev only”) | Context-only |
| `sku` | Display/reference label when providers rename SKUs | Context-only |

`channel` is especially helpful on multimodal workloads so dashboards can show whether costs landed on `text_transcript`, `subtitle_caption`, `video_only`, or `video_plus_audio`. Keep `operation` values consistent—`stt`, `tts`, and `video` are the current canonical tags—to ensure the pricing helper resolves tiers deterministically.

## Phased Delivery
1. **Schema & loader foundation**: add JSON column/models, config parsing, migration that seeds tiers from existing fields, keep runtime behavior identical (still reading scalar values) while flagging entries lacking tiers.
2. **Usage pipeline + executor refactor**: introduce pricing helper, support image/audio overrides via tiers, make logger consume new structure while honoring fallback.
3. **Admin/API/UI upgrades**: expose tier editor, extend API, add usage reporting metadata, adjust tests.
4. **Cleanup**: remove deprecated metadata keys, document new flow, add integration tests for multi-tier costing, revisit budgets to ensure warnings remain accurate.

## Risks / Open Questions
- Need deterministic ordering + validation for tiers to avoid ambiguous billing (tie-breaking when metadata tags overlap).
- Some providers return usage mid-stream (SSE). Ensure streaming completions capture full token counts for tier selection.
- Backward compatibility for automation hitting Admin API expecting scalar fields; consider versioned API or transitional period where server accepts both but always responds with normalized tiers.
