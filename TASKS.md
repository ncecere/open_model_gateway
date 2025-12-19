# Multimodal Content Tasks

1. **Catalog text-only assumptions**
   - Search for `ChatMessage.Content` usage across handlers, executors, adapters, batch worker, and SSE code.
   - Summarize the expectations/limits for each caller and highlight anything that would panic or misbehave with `nil`/empty content parts.

2. **Design message content structs**
   - Update `backend/internal/models/chat.go` with `MessageContentPart` definitions and `ContentParts` slices on `ChatMessage`, `ChatChunk`, and related types.
   - Add helpers for normalizing between legacy string content and the new part array.
   - Cover serialization with unit tests.

3. **Refactor HTTP request parsing**
   - Extend the OpenAI route DTOs to accept `content` arrays and convert them into `ContentParts`.
   - Implement validation for supported part types, limits, and role constraints.
   - Ensure moderation/logging/token counting keeps working with the new structure.

4. **Adapter propagation**
   - Update the OpenAI adapter to emit mixed content payloads and add capability flags for other providers.
   - Implement fallback/validation paths for adapters that remain text-only.
   - Adjust token accounting or metadata extraction if upstream responses now include per-part details.

5. **Streaming & batch alignment**
   - Modify SSE chunk emission so streaming clients receive per-part deltas mirroring OpenAI’s format.
   - Update the batch worker to read/write NDJSON lines containing the full content array.

6. **Testing & documentation**
   - Add table-driven tests for DTO parsing, adapter conversions, and legacy compatibility.
   - Create integration tests (mocking provider adapters) that confirm mixed content requests succeed.
   - Update API docs/release notes describing the new behavior and any provider-specific limitations.
