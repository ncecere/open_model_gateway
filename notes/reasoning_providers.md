# Reasoning Support Roadmap

## Goal
Expose provider reasoning traces (chain-of-thought / reasoning text streams) end-to-end across the Open Model Gateway so both streaming and non-streaming chat completions can optionally include provider-specific reasoning payloads when available.

## High-Level Requirements
1. **Schema Support**
   - Extend `internal/models` to capture reasoning metadata (`Reasoning` string, `ReasoningDetails` array of typed blobs) alongside the existing `role/content`.
   - Ensure JSON serialization for public + admin APIs includes these fields only when populated to maintain backward compatibility.
2. **Adapter Coverage**
   - Update every provider adapter to capture reasoning output when the upstream provider exposes it. Scope includes OpenRouter, OpenAI (Responses API / reasoning mode), Anthropic (Claude reasoning traces), Bedrock, Azure, Vertex, and any OpenAI-compatible endpoints that forward reasoning fields.
   - For providers that do not support reasoning, explicitly return empty metadata.
3. **Streaming Lifecycle**
   - Update `streamutil` consumers so SSE chunks can include reasoning segments (e.g., `choices[].delta.reasoning`). Decide whether reasoning arrives as separate event types or appended to the final message.
4. **HTTP Surface**
   - Modify `/v1/chat/completions` handlers (public/admin/user) to pass through the enriched `models.ChatResponse` objects, including reasoning metadata.
   - Document a request knob (`extra_body.reasoning.enabled` or provider-specific fields) that informs the router to enable reasoning when supported.
5. **Client Awareness**
   - Update frontend/admin UI (if needed) and SDK/API documentation so operators know how to request/consume reasoning.

## Detailed Implementation Plan

### 1. Schema & Contract Updates
1.1 **Model Changes**
   - `backend/internal/models/chat.go`
     - Add new optional fields to `ChatMessage`: `Reasoning string` and `ReasoningDetails []ReasoningDetail`.
     - Define `ReasoningDetail` struct (type, text, metadata) matching OpenRouter’s payload.
     - Ensure `ChatChunk` mirrors these fields for streaming (e.g., `ChunkDelta.Reasoning`).
1.2 **Marshalers**
   - Validate Fiber responses automatically include the new fields when `omitempty` is satisfied.
1.3 **Tests**
   - Add serialization unit tests confirming that legacy clients still receive the same payload when reasoning is absent.

### 2. Adapter Enhancements
2.1 **OpenRouter Adapter**
   - `convertChatResponse` / `convertChatChunk`: map `message.reasoning` and `reasoning_details` into the new structs.
   - Propagate `extra_body.reasoning.enabled` from requests (already supported) so upstream responses include traces.
2.2 **OpenAI Adapter**
   - Investigate reasoning availability (currently limited); if unsupported, set expectation to `nil`.
2.3 **Anthropic Adapter**
   - Claude 3 supports reasoning streams; map `message.metadata.stop_reasoning`, etc. Confirm API contract before implementation.
2.4 **Bedrock Vertex & OpenAI-Compatible**
   - Provide extension points for providers that adopt reasoning later; explicitly return `nil` for now.
2.5 **Interfaces**
   - Adapters should remain backwards compatible. No interface change is needed because reasoning lives entirely inside `models.ChatResponse`.
2.6 **Fixtures**
   - Add reasoning-specific fixtures to `backend/internal/providers/fixtures/testdata`.

### 3. Streaming Integration
3.1 **SSE Handling**
   - Update `streamutil.Forward` consumers (OpenRouter/Anthropic/others) so SSE chunks carrying reasoning deltas forward them in `ChunkDelta.Reasoning`.
3.2 **Keep-Alive Behavior**
   - Ensure keep-alive chunks without reasoning don’t break the parser.
3.3 **Usage Logging**
   - Confirm reasoning chunks don’t affect usage metrics; only content tokens count.

### 4. API Handlers
4.1 **Public `/v1/chat/completions`**
   - No structural change; new fields appear automatically once `models.ChatResponse` is enriched.
   - Document `extra_body.reasoning.enabled` usage in `docs/api/chat.md`.
4.2 **Admin/User APIs**
   - Ensure admin debug endpoints show reasoning when present.
4.3 **Validation**
   - When a client requests reasoning for a provider that doesn’t support it, either silently ignore (current behavior) or return a warning via response metadata—needs product decision.

### 5. Frontend & Docs
5.1 **Docs**
   - Add section to `docs/api/chat.md` and provider-specific guides describing how to request reasoning.
   - Update `docs/runtime/config.md` if new config toggles are introduced.
5.2 **UI (Optional)**
   - If admin UI will display reasoning for debugging, plan a toggle or expandable view.

### 6. Testing & Rollout
6.1 **Unit Tests**
   - Coverage for adapter conversions, JSON serialization, and SSE forwarding.
6.2 **Integration Tests**
   - End-to-end tests hitting mocked providers to ensure reasoning fields survive from request → adapter → response.
6.3 **Backward Compatibility**
   - Ensure clients that ignore the new fields continue working; update versioned API docs accordingly.

## Open Questions
1. Should reasoning be enabled globally or per-request? (Current assumption: per-request via `extra_body.reasoning.enabled`.)
2. Do we want to redact reasoning by default for security/compliance? (Might require config flag.)
3. How should streaming clients differentiate reasoning vs. standard content? Define event schema early.

## Next Steps
1. Align on API contract (fields + request knob) with stakeholders.
2. Create work tickets per section (schema, adapters, streaming, docs).
3. Implement OpenRouter path first and validate with live traffic.
4. Roll out to additional providers as they expose reasoning metadata.
