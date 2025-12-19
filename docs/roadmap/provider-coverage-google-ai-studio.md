# Provider Coverage: Google AI Studio

## Summary
Extend the existing Vertex adapter to support Google AI Studio’s public REST endpoints (Gemini 2.0, Imagen) so operators can onboard Google-hosted models without provisioning full Vertex projects. Focus on simple API key auth, rapid experimentation, and regional compliance.

## Implementation Outline

### Adapter Layer
1. **Client Config** – new provider block `providers.google_ai_studio` (API key, base URL, project). Metadata fields per model for safety settings, system instructions, and response MIME types.
2. **Chat/Responses** – reuse the multimodal message translation to map OpenAI-style `content_parts` to Gemini’s request schema (`contents`, `system_instruction`). Support both text-only and image/audio parts.
3. **Streaming** – implement the SSE stream reader for Gemini’s `responses:streamGenerate` endpoint.
4. **Images** – map Imagen generation options (sizes, background, style) to `/images/generations`.

### Routing & Catalog
- Add sample catalog entries for `gemini-3.0-flash`, `gemini-2.0-pro`, `imagen-3` with pricing metadata.
- Include capability flags (vision/audio) pulled from the catalog metadata.

### Auth & Quotas
- Each route uses an API key header (`x-goog-api-key`); optionally support service-account JSON for teams that need OAuth.
- Enforce per-provider rate limits using the existing key limiter buckets.

## Example Workflow
1. Operator drops a config snippet:
   ```yaml
   providers:
     google_ai_studio:
       api_key: ${GOOGLE_AI_STUDIO_KEY}
       project: my-project
   model_catalog:
     - alias: gemini-3.0-flash
       provider: google_ai_studio
       provider_model: models/gemini-3.0-flash
       modalities: ["text","image","audio"]
   ```
2. Tenant calls `/v1/responses` with `model=gemini-3.0-flash` and mixed text/image content; adapter translates to Gemini schema and returns the standard OpenAI payload.
3. Admin monitors latency/error metrics via the existing telemetry pipeline (provider=`google_ai_studio`).

## Components Needed
- New adapter package under `internal/adapters/googleai`.
- Provider registry entry + builder (`builder_googleai.go`).
- Config validation + docs (runtime config + admin catalog examples).
- Tests covering sync/streaming chat + image flows via recorded fixtures.

## Risks
- API stability (AI Studio still evolving) → keep versioning knobs in config.
- Quota enforcement (per-key quotas) → surface provider-specific error codes for circuit breaking.

## Next Steps
1. Prototype chat adapter using Google sample key.
2. Implement streaming + image support.
3. Document onboarding and add catalog samples.
4. Wire telemetry + health checks.
