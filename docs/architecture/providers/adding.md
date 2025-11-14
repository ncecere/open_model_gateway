# Adding a Provider Adapter

This repo now treats each provider as a capability bundle. Adapters only need to implement the interfaces that make sense for their model deployments:

| Capability | Interface | Methods |
|------------|-----------|---------|
| Chat (non-stream) | `providers.ChatCompletions` | `Chat(ctx, models.ChatRequest)` |
| Chat streaming | `providers.ChatStreaming` | `ChatStream(ctx, models.ChatRequest)` |
| Embeddings | `providers.EmbeddingsProvider` | `Embed(ctx, models.EmbeddingsRequest)` |
| Images | `providers.ImagesProvider` | `Generate(ctx, models.ImageRequest)` |
| Model listing | `providers.ModelLister` | `Models(ctx)` |

## Implementation Checklist

1. **Adapter**
   - Live under `backend/internal/adapters/<provider>/`.
   - Fulfil the relevant capability interfaces. If you expose streaming, rely on `streamutil.Forward` to normalize channel lifecycle/usage-only chunks.
   - Provide conversion helpers that translate provider responses into `internal/models` structs.
2. **Builder**
   - Create `backend/internal/providers/builder_<provider>.go` with `build<Provider>Route(ctx, cfg, entry)`.
   - Parse `config.ModelCatalogEntry` metadata/environment defaults and populate `providers.Route` fields (`Chat`, `ChatStream`, `Embedding`, `Image`, etc.).
   - Call `providers.RegisterDefinition(providers.Definition{Name: "provider-key", Description: "Human readable", Builder: build<Provider>Route})` in `init()`.
3. **Registry Docs**
   - Update `docs/runtime/config.md` with metadata/env keys and add/modify a provider guide under `docs/architecture/providers/`.
4. **Tests**
   - Add fixture JSON under `backend/internal/providers/fixtures/testdata/` to capture sync/stream responses.
   - Write adapter tests that feed the fixtures through the conversion helpers to guarantee usage/tokens/responses match expectations.
5. **Tasks & Agents**
   - Check in updates to `TASKS.md` and `agents.md` explaining any new metadata or required follow-up work.

## Discovering Registered Providers

Gateway admins can call `GET /admin/providers` to see the current registry (powered by `providers.DefaultDefinitions`). The response includes the provider key, description, and declared capabilities, which matches the table exposed in this doc.

Following this checklist keeps the router factory untouched—the registry automatically exposes new builders via `providers.DefaultDefinitions()` and the factory clones that map on boot.
