# Multimodal Chat Content Plan

## Goal
Enable the Open Model Gateway to accept OpenAI-style chat messages with mixed content parts (text + image URLs/file attachments, future audio/video) so we can faithfully mirror the upstream Chat Completions and Responses APIs. The plan focuses on server-side compatibility; SDK/UI work can follow once the backend path is stable.

## Constraints & Principles
- Maintain backward compatibility for existing text-only clients; old `content` strings should continue to work.
- Preserve provider-agnostic abstractions. `models.ChatMessage` is the canonical interface between handlers, services, and adapters.
- Ship feature flags or validation guards where a provider cannot handle certain part types.
- Keep streaming, batching, and logging behaviors consistent with current expectations.

## Phase 1 – Inventory & Gap Analysis
1. Trace every code path that reads or writes `ChatMessage.Content string`:
   - Request parsing (`backend/internal/httpserver/public/openai_routes.go`).
   - Service layer (`internal/executor`, `internal/pipeline`).
   - Provider adapters (OpenAI, OpenRouter, Anthropic, Azure, Vertex, etc.).
   - Batch worker and NDJSON writers.
   - Streaming chunk structs and SSE emitters.
2. Document assumptions (e.g., `content != ""`, `len(content) <= limit`) and note which ones break when `content` becomes a composite array.
3. Produce a short compatibility matrix showing which providers can forward multimodal payloads immediately and where we’ll initially reject unsupported part types.

## Phase 2 – Schema & Model Updates
1. Extend `models.ChatMessage` with a `ContentParts []MessageContentPart` while retaining the existing `Content string` for backward compatibility.
   - Define `MessageContentPart` struct with `Type` discriminator and payload unions (text, image URL/file ID, audio transcript placeholder, etc.).
   - Mirror OpenAI naming (`text`, `image_url`, `input_audio`, `output_audio`, etc.), even if we only enable text+image in the first iteration.
2. Update `models.ChatChunk` / `ChunkDelta` to support part-wise streaming updates (e.g., text deltas for a specific content index).
3. Add helper methods to:
   - Normalize legacy string content into a single text part.
   - Flatten content parts back into a legacy string (for providers that still demand text-only payloads).
4. Write serialization unit tests ensuring JSON marshaling yields OpenAI-compatible shapes and still honors old clients.

## Phase 3 – Handler & Validation Changes
1. Refactor the OpenAI HTTP DTOs so they parse `content` arrays alongside legacy strings.
2. Add validation rules:
   - Enforce allowed part combinations per role (e.g., assistant output cannot include `input_image`).
   - Cap number/size of image URLs or file references according to provider limits.
   - Reject unsupported types with explicit `400` errors.
3. Update request-to-model translation to populate `ContentParts` and fallback `Content` text as needed.
4. Ensure moderation/request logging/tokens look at the correct text fields.

## Phase 4 – Provider Adapter Propagation
1. **OpenAI Adapter**: map `ContentParts` to `[]openai.ChatCompletionMessageContentPartParam`. For providers that support images, forward the full array; otherwise, collapse to text or raise `unsupported_content_parts`.
2. **Other Providers**: evaluate each adapter:
   - OpenRouter may already support `content` arrays—mirror their schema.
   - Anthropic/Bedrock/Vertex might still be text-only; implement sanitizers.
3. Introduce capability flags per provider (`SupportsImageInput`, etc.) so routing decisions are explicit.
4. Update token counting/usage logic if adapters return per-part stats (e.g., OpenAI billing for images).

## Phase 5 – Streaming & Batch Alignment
1. Streaming (`chat_stream_pipeline.go`): ensure SSE payloads carry per-part deltas, matching OpenAI’s `chat.completion.chunk` schema for textual outputs and, eventually, image references.
2. Batch worker (`internal/batchworker`): when reading/writing NDJSON for `/v1/chat/completions` or `/v1/responses`, include the full content array so downloaded artifacts are drop-in replacements for OpenAI.
3. Admin/logging views: adjust any UIs or logs that assume a single `content` string.

## Phase 6 – Testing, Docs, and Rollout
1. Unit tests covering:
   - DTO parsing of text-only vs. mixed content requests.
   - Adapter conversion for OpenAI with and without image parts.
   - Legacy compatibility (old clients still succeed).
2. Integration tests hitting `/v1/chat/completions` with content arrays and verifying provider payloads via mocks/fixtures.
3. Documentation updates (`docs/api/chat.md`, release notes) highlighting new capabilities and limitations per provider.
4. Feature-flag rollout plan if we need to gate the behavior while adapters catch up.
