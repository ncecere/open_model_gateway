# Provider Coverage: Ollama

## Summary
Ollama is a lightweight inference server that developers use for local or on-prem experiments (e.g., `ollama run mistral`). Supporting Ollama through Open Model Gateway lets teams prototype quickly while still leveraging guardrails, budgets, and the OpenAI-compatible surface.

## Implementation Plan

### Adapter Features
- **Transport**: REST calls to `POST /api/generate` (sync) and `POST /api/chat` (chat history); streaming responses via chunked text. No auth by default, but allow optional Basic/Auth header for secured deployments.
- **Model mapping**: catalog entry’s `provider_model` is the Ollama model name (`mistral`, `llama3`, etc.). Provide metadata for context window, max tokens, etc.
- **Embeddings**: optional support via `POST /api/embeddings` if Ollama version exposes it.
- **Health checks**: call `/api/tags` or a lightweight generate.

### Configuration Example
```yaml
providers:
  ollama:
    base_url: http://ollama.local:11434
model_catalog:
  - alias: ollama-mistral
    provider: ollama
    provider_model: mistral
    modalities: ["text"]
```

### Tenant Flow
- System admin seeds catalog entries for the Ollama-hosted models they trust.
- Tenant owners/admins attach/detach those aliases like any other provider.

## Components
- Adapter package (`internal/adapters/ollama`) for sync/stream endpoints.
- Provider builder + config validation (base URL, optional auth).
- Docs showing how to run Ollama and register models.

## Risks
- Ollama is often run without auth → document security best practices (reverse proxy, tokens).
- Feature gaps (no embeddings in older versions) → handle missing endpoints gracefully.

## Next Steps
1. Build adapter hitting `POST /api/generate` for sync + streaming.
2. Add config schema + catalog examples.
3. Document onboarding (docker compose, sample curl).
4. Add telemetry tags + tests.
