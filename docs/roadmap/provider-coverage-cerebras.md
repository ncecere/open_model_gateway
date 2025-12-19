# Provider Coverage: Cerebras

## Summary
Integrate Cerebras Inference (cloud or on-prem cluster) so customers running private LLM hardware can expose their models through Open Model Gateway with the same OpenAI-compatible API.

## Implementation Plan

### Adapter Design
1. **Transport** – REST + optional gRPC client (depending on deployment). Provide TLS + token auth options.
2. **Chat/Responses** – convert OpenAI `content_parts` into Cerebras request schema, including support for long context windows and system prompts.
3. **Streaming** – implement chunk reader over Cerebras SSE/websocket stream (depending on API), emitting standard `chat.completion.chunk` payloads.
4. **Embeddings / Custom Ops** – map `/v1/embeddings` and optional `/v1/completions` if the API exposes classical completion endpoints.

### Deployment Modes
- **Hosted cloud**: base URL + API token.
- **On-prem cluster**: support custom CA certificates, node load balancing, and health checks hitting the local control plane.

### Routing & Config
- Add `providers.cerebras` config (base URL, auth token, ca_file, timeout).
- Catalog entries specify cluster/tenant metadata so usage/billing can differentiate between shared vs dedicated hardware.

### Observability
- Health checks per-cluster to monitor queue depth/availability.
- Telemetry tags `provider=cerebras` for dashboards; capture hardware-specific metrics (throughput, queue time) if available.

## Example Config
```yaml
providers:
  cerebras:
    base_url: https://cerebras-cluster.local/api
    api_key: ${CEREBRAS_TOKEN}
    ca_file: /etc/certs/cerebras-ca.pem
model_catalog:
  - alias: cerebras-gaudi-xl
    provider: cerebras
    provider_model: gaudi-xl
    context_window: 200000
    modalities: ["text"]
```

## Required Components
- Adapter implementation + health checks.
- Config builder + metadata docs (admin catalog section).
- Usage accounting to handle custom pricing (per-cluster hourly plus tokens).
- Optional scheduler integration if we need to queue requests when cluster is busy.

## Risks
- Proprietary API changes; keep adapter modular and versioned.
- Latency if clusters are self-hosted; consider async streaming to hide queue time.

## Next Steps
1. Obtain API access (cloud + on-prem) and record fixtures.
2. Implement adapter + streaming.
3. Document onboarding (cert management, config examples).
4. Update roadmap status once live.
