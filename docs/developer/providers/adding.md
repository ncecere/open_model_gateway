# Add Provider Adapters

Adapters expose provider capabilities through shared interfaces so the router can compose routes without special cases.

## Understand capability interfaces
Consult this table before adding code.

| Capability | Interface | Method |
| --- | --- | --- |
| Chat (sync) | `providers.ChatCompletions` | `Chat(ctx, models.ChatRequest)` |
| Chat streaming | `providers.ChatStreaming` | `ChatStream(ctx, models.ChatRequest)` |
| Embeddings | `providers.EmbeddingsProvider` | `Embed(ctx, models.EmbeddingsRequest)` |
| Images | `providers.ImagesProvider` | `Generate(ctx, models.ImageRequest)` |
| Model listing | `providers.ModelLister` | `Models(ctx)` |

## Wire the adapter
Place the adapter in `backend/internal/adapters/<provider>` and implement the needed interfaces, leaning on shared helpers (`streamutil.Forward`, `executor` retry helpers, metadata parsers) so SSE, usage, and errors follow the same semantics.

## Register the builder
Create `backend/internal/providers/builder_<provider>.go`, parse `config.ModelCatalogEntry` metadata, populate a `providers.Route`, and register it via `providers.RegisterDefinition` inside `init()` so `/admin/providers` and the factory see the new builder.

## Document metadata
Update `docs/runtime/config.md` and the provider guide under `docs/developer/providers/` with any new metadata/env keys, then call out bootstrap impacts in `agents.md` if operators need to seed secrets or overrides.

## Add fixtures and tests
Capture sample payloads under `backend/internal/providers/fixtures/testdata`, add adapter tests to guarantee chat/stream/usage conversions, and extend contract tests if the provider exposes new modalities.

## Surface the registry
`GET /admin/providers` enumerates `providers.DefaultDefinitions()` so operators can confirm the adapter landed without reading code; use it when validating migrations or docs.
