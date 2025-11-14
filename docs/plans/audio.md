# Audio API Work Plan

## Goals
- Implement the full OpenAI audio surface so clients can talk to `/v1/audio/transcriptions`, `/v1/audio/translations`, `/v1/audio/speech`, and `/v1/audio/speech/:id` using the same payloads as the upstream spec.
- Keep the adapter pluggable: Azure, Bedrock, Vertex, OpenAI, and future providers should participate just like chat/embeddings today.
- Manage audio-capable models through the existing catalog (aliases, pricing, overrides, routing weights).
- Support streaming/chunked responses for text-to-speech while honoring the global 20 MB request body limit (controlled by `server.body_limit_mb`, default 20).

## API Surface
| Endpoint | Direction | Notes |
|----------|-----------|-------|
| `POST /v1/audio/transcriptions` | Speech → text | Accept multipart/form-data and JSON (per OpenAI) with `file`, `model`, `prompt`, `language`, `temperature`, etc. |
| `POST /v1/audio/translations` | Speech → text (English) | Same form handling as transcriptions. |
| `POST /v1/audio/speech` | Text → speech | Accept JSON payload (`model`, `input`, `voice`, `format`), return binary audio or Base64 depending on `response_format`. |
| `GET /v1/audio/speech/{id}` | Poll generated audio | Backed by storage for async jobs or passthrough for providers that expose IDs. |

## Provider Strategy
1. **Interfaces**: Extend `internal/providers` with `AudioTranscriber`, `AudioTranslator`, `TextToSpeech`, and optional `SpeechJobStore` interfaces. Builders register capabilities like other adapters.
2. **Adapters**:
   - **OpenAI**: wrap Whisper/TTS endpoints via the same HTTP client; reuse OpenAI-compatible credentials.
   - **Azure**: map to Azure Speech services (REST). Requires catalog metadata for voice locale, deployment, etc.
   - **Bedrock/Vertex**: evaluate available audio models; expose metadata toggles (`voice`, `format`, `language`).
   - **OpenAI-Compatible**: allow compatible gateways to surface audio by pointing `endpoint` at `/v1`.
3. **Model catalog**: add `modalities: ["audio"]` plus provider-specific metadata/overrides (e.g., `voice`, `sample_rate`, `encoding`). Persist pricing fields to charge per minute/token as needed.

## Request Handling
- **Upload parsing**: add helpers in `internal/httpserver/public/audio.go` to accept both multipart (file stream) and JSON (Base64) per OpenAI’s spec.
- **Streaming**: use Fiber’s `ctx.Context()` + the existing SSE/websocket utilities to forward chunked TTS responses (OpenAI `stream: true`). If providers return binary streams, wrap them as chunked HTTP responses.
- **Limits**: rely on `server.body_limit_mb` (currently 20 MB) for uploads; document a smaller recommended size in API docs if needed.
- **Idempotency**: honor `Idempotency-Key` for POSTs, caching transcriptions by hash of file+params when feasible.

## Admin & Docs
- **Docs**: update `API.md` once endpoints ship; add provider-specific metadata tables (Azure/Bedrock/Vertex) covering audio knobs, plus curl examples.
- **UI**: future iteration can expose audio models in the catalog dialog (new modality pill + fields), but backend is priority.

## Work Breakdown
1. Schema: ensure `model_catalog.modalities_json` accepts `"audio"`; add pricing fields if per-minute billing differs (optional in v1).
2. Config: extend provider docs (`docs/architecture/providers/*.md`) to describe audio metadata/overrides.
3. Providers: implement adapters + register capabilities.
4. HTTP layer: mount `/v1/audio/*` routes, parsing, response normalization, streaming.
5. Tests: unit tests for request parsing + provider adapters; integration tests using mock providers and sample audio fixtures.
6. Docs & examples: update `API.md`, PRD, and README smoke tests with audio curl snippets.

