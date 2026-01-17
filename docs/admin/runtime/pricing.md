---
title: pricing guide
description: Tiered model costs and metadata
---
[**Open Model Gateway**](/) bills every workload using tier metadata stored beside the model catalog.

---

#### Adopt tiers
Each alias defines `pricing_tiers` buckets (input, output, image, tts, etc.) alongside legacy `price_input` and `price_output` floats.
The runtime keeps scalars for backward compatibility but always prefers tier rows when calculating spend.

---

#### Define tier fields
| Field | Notes |
| --- | --- |
| `unit` | Required; defaults to `tokens_per_million` when omitted. |
| `price_per_unit` | USD-per-unit decimal recorded verbatim in `pricing_tiers_json`. |
| `max_units` | Inclusive ceiling for that row; `null` keeps the tier open-ended. |
| `metadata` | Free-form selectors (e.g., `quality`, `operation`, `channel`, `context_scope`) that dashboards echo and billing logic can match. |

---

#### Choose units
| Unit | When to use |
| --- | --- |
| `tokens_per_million` / `tokens_per_thousand` | Chat, embedding, or cache buckets depending on upstream pricing granularity. |
| `per_image` / `per_megapixel` | Image generations, edits, or variations; pair megapixel tiers with `approx_resolution`. |
| `per_minute` / `per_second` | Audio transcription or video streaming durations. |
| `per_million_characters` | Text-to-speech workloads where providers count output characters. |

---

#### Tag metadata
| Key | Used for billing? | Guidance |
| --- | --- | --- |
| `quality` | Yes (images) | Matches UI presets such as low/standard/hd when selecting an image tier. |
| `operation` | Yes (images) | Indicate generation vs edit vs variation to override the default `image` bucket. |
| `resolution` | Yes (images) | Match `size` strings or inferred edit dimensions for megapixel billing. |
| `context_scope` | No | Describe token ranges (<=200k, >200k) for LLM tiers. |
| `channel`, `voice`, `usage`, `sku` | No | Document TTS channels, default voices, policy hints, or upstream SKU labels for operators. |

---

#### Map usage buckets
| Bucket | Measurement |
| --- | --- |
| `input`, `output`, `cache` | Prompt/completion/cache tokens measured per request. |
| `embedding` | Embedding tokens per million or per thousand. |
| `image`, `image_generation`, `image_edit`, `image_variation` | Image counts or megapixels, split per operation when needed. |
| `audio`, `tts`, `video` | Minutes, characters, or seconds recorded by `/v1/audio/*` and future video routes. |

---

#### Configure workflows
Define tiers via YAML/bootstrap, the Admin UI, or the Admin API; every path persists JSON identical to the snippet below.

```yaml
model_catalog:
  - alias: "gpt-4o-mini"
    provider: "openai"
    pricing_tiers:
      input:
        - unit: tokens_per_million
          price_per_unit: 0.0005
      output:
        - unit: tokens_per_million
          price_per_unit: 0.0015
      cache:
        - unit: tokens_per_million
          price_per_unit: 0.00025
      image:
        - unit: per_image
          price_per_unit: 0.04
          metadata:
            quality: "standard"
            operation: "image_generation"
```

---

#### Troubleshoot caches
Ensure each bucket ends with a terminal tier (no `max_units`) so large requests never fall back to legacy floats.
After edits, call the Admin API to verify tiers arrive sorted because the router relies on that order before logging costs.

---

#### Research
- Collated tier behavior from the previous pricing guide plus `deploy/router.example.yaml` samples.
- Verified cache and unit handling via `backend/internal/pricing/cache.go` and the latest tier migration referenced in `AGENTS.md`.
