# Provider Coverage: vLLM / HF TGI

## Summary
Many customers host private Llama/Qwen/Mixtral deployments using vLLM or Hugging Face Text Generation Inference (TGI). Adding a generic inference-server adapter lets them expose those models through Open Model Gateway while reusing budgets, guardrails, and API compatibility.

## Implementation Plan

### Adapter Features
- **Transport**: configurable base URL + optional auth header. Support both vLLM’s OpenAI-compatible API and TGI’s text generation API.
- **Endpoints**: `/v1/chat/completions`, `/v1/completions`, `/v1/embeddings` (if available). Streaming support via SSE.
- **Model registry**: catalog entries specify `provider_model` = model name on the inference server; metadata includes context window, max output tokens, etc.
- **Health checks**: ping `/health` or run a lightweight completion.

### Configuration
```yaml
providers:
  vllm:
    base_url: https://llama-cluster.local/v1
    auth_header: "Authorization: Bearer ${VLLM_TOKEN}"
model_catalog:
  - alias: llama-3.1-70b-private
    provider: vllm
    provider_model: "llama3.1-70b"
    context_window: 128000
    modalities: ["text"]
```

### Routing & Limits
- Support multiple deployments by allowing `deployment` metadata (cluster name/region) and letting tenants attach/detach them.
- Expose telemetry tags (`provider=vllm`, `deployment=cluster-a`).

## Components
- Adapter package (`internal/adapters/vllm`) handling chat/completions/streaming.
- Provider builder + config validation.
- Updated catalog editor + docs with sample entries.
- Usage logging/telemetry integration.

## Risks
- API differences between vLLM versions → feature-detect streaming/completions.
- Resource contention on customer clusters → add configurable timeouts and circuit breaker thresholds.

## Next Steps
1. Prototype adapter against vLLM’s OpenAI-compatible API.
2. Add config schema + builder.
3. Document onboarding (including security considerations).
4. Provide sample catalog entries + tests.
