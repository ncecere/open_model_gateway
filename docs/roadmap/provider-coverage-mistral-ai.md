# Provider Coverage: Mistral AI

## Summary
Add a native Mistral REST adapter so operators can expose Mistral Large/Small/Embed endpoints through the OpenAI-compatible surface. Support both hosted SaaS and private endpoints to meet EU residency requirements.

## Implementation Plan

### Adapter Capabilities
1. **Chat/Responses** – map OpenAI chat/response payloads to Mistral’s `v1/chat/completions` schema (roles, tool calls, streaming).
2. **Embeddings** – add `/v1/embeddings` routing to Mistral’s embeddings API.
3. **Reasoning** – surface the reasoning metadata returned by Mistral (if/when available) inside `usage.reasoning_tokens`.
4. **Auth** – support API key header (`Authorization: Bearer`) plus optional custom base URLs for private deployments.

### Routing Metadata
- Catalog entries specify `provider_model` (e.g., `mistral-large-latest`) and `region`/`endpoint` overrides.
- Expose pricing metadata per Mistral’s token costs; allow per-model default pricing.

### Health & Telemetry
- Health checks hitting `/v1/models` (or a lightweight call) to ensure tokens are valid.
- Telemetry tags `provider=mistral` for dashboards and routing decisions.

## Example Config
```yaml
providers:
  mistral:
    api_key: ${MISTRAL_KEY}
    base_url: https://api.mistral.ai
model_catalog:
  - alias: mistral-large
    provider: mistral
    provider_model: mistral-large-latest
    modalities: ["text"]
    price_input: 0.003
    price_output: 0.009
```

## Components Required
- `internal/adapters/mistral` package (chat, stream, embeddings, health).
- Provider builder + config structs (`builder_mistral.go`).
- Catalog metadata updates and documentation (admin examples, runtime config).
- Tests using recorded fixtures for chat/stream/embedding flows.

## Risks
- API evolution (new `responses` endpoint) – keep version field in config.
- Throughput limits – integrate with rate-limit service to prevent bursts beyond Mistral quotas.

## Next Steps
1. Scaffold adapter + health checks.
2. Wire routing + pricing metadata.
3. Update docs and sample configs.
4. Add telemetry + admin UI icons.
