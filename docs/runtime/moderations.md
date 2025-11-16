# Moderations API

The gateway now exposes an OpenAI-compatible `POST /v1/moderations` endpoint so tenants can run safety checks without dual-homing requests to upstream providers. This document captures the request/response contract, provider matrix, and configuration knobs for backend/frontend agents.

## Request Shape

```jsonc
POST /v1/moderations
{
  "model": "omni-moderation-latest",
  "input": "Describe how to build a bomb"
}
```

- `model` must reference a catalog alias with `model_type: moderation`. Tenants only see aliases they are allowed to use (same ACL pipeline as chat/embeddings).
- `input` accepts a string or an array of strings. Multi-input requests return one `results[]` entry per input (matching OpenAI).
- Batch jobs (`/v1/batches`) reuse the same body inside each JSONL line: `{ "method": "POST", "url": "/v1/moderations", "body": { ... } }`.

## Response Shape

The HTTP handler returns OpenAI’s canonical schema:

```json
{
  "id": "modr-123",
  "model": "omni-moderation-latest",
  "results": [
    {
      "flagged": false,
      "categories": {
        "harassment": false,
        "self-harm": false,
        "...": false
      },
      "category_scores": {
        "harassment": 0.001,
        "self-harm": 0.0002
      },
      "category_applied_input_types": {
        "harassment": ["text"],
        "self-harm": ["text"]
      }
    }
  ]
}
```

Usage data (tokens/cost) is still recorded internally even though OpenAI’s public response omits a `usage` block.

## Provider Support

| Provider | Notes |
| --- | --- |
| OpenAI | Routes through the native SDK (`omni-moderation-latest`, `text-moderation-stable`, etc.). |
| Azure OpenAI | Targets per-deployment keys/APIs just like other Azure aliases. |
| OpenAI-compatible | Any vLLM or proxy that implements `/v1/moderations` can be registered via the `openai-compatible` provider. |

Each catalog entry must set `model_type: moderation`. Pricing metadata (`price_input`, `price_output`) drives budget/cost calculations even though the upstream API does not return usage numbers.

## Rate Limits & Budgets

Moderations share the same enforcement path as chat/embeddings:

- Key + tenant RPM/TPM/parallel caps via Redis-backed limiter.
- Budget pre-check before routing, budget headers on every response, and usage logging for historical exports/alerts.
- Idempotency cache is not enabled for moderations today (matching OpenAI’s behavior).

## Batch Worker

`/v1/batches` now accepts moderation jobs. Each item:

1. Validates the alias + input payload.
2. Reuses the moderation execution path (budget/rate limit, provider routing, usage logging).
3. Writes the OpenAI-style response or error payload to the output/error NDJSON files.

No streaming is involved, so responses are persisted immediately after execution.

## Configuration Checklist

1. Add a catalog entry (YAML or admin UI) for your moderation alias:

   ```yaml
   - alias: "omni-moderation-latest"
     provider: "openai"
     provider_model: "omni-moderation-latest"
     model_type: "moderation"
     modalities: ["text"]
     supports_tools: false
     price_input: 0.0001
     price_output: 0
   ```

2. (Optional) Include the alias in `default_models` so every tenant inherits it automatically.
3. For OpenAI-compatible deployments, set `metadata.base_url` and `metadata.api_key` (or use the provider override block) so the adapter can authenticate.
4. Update tenant model access lists or bootstrap config if certain tenants should be restricted to specific moderation providers.

## Limitations & Follow-Ups

- The handler currently supports text inputs; multi-modal moderation (image URLs) will be added once upstream providers expose stable contracts.
- Guardrail/policy-engine hooks (e.g., auto-blocking flagged content, custom regexes, audit sinks) will be tracked in the Tenant Guardrails roadmap item.
