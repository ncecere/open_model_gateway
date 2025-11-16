# Images API Completion Plan

## Plan

1. **Spec & Handler Parity** – Update the `/v1/images/*` handlers (`backend/internal/httpserver/public/openai_routes.go`) so JSON and multipart flows accept OpenAI’s field aliases (`image[]`, `mask[]`), enforce size/count/mime limits, share validation helpers, and emit the same `invalid_request_error` payloads. Keep `runImageOperation` as the shared execution path with budget/idempotency handling so edits/variations inherit generations’ protections.
2. **Adapter Coverage** – Audit every provider registered with the `ImagesProvider` interface. Ensure OpenAI/OpenAI-compatible/Vertex/Bedrock adapters implement edits & variations with the latest metadata knobs (mask mode, guidance, variation prompts) while Azure & Titan explicitly return `ErrImageOperationUnsupported`. Document the knobs + limitations in `docs/runtime/config.md` and config samples.
3. **Routing, Pricing & Idempotency** – Extend `runImageOperation` to respect per-operation pricing metadata (`price_image_edit_cents`, `price_image_variation_cents`), derive usage when providers omit token counts, and persist successful responses in the idempotency cache so retries for edits/variations behave like generations. Normalize OpenAI-style error codes surfaced to clients.
4. **Batch & Admin Surfaces** – Teach the batch worker (`backend/internal/batchworker/worker.go`) how to execute `/v1/images/edits` and `/v1/images/variations` items (JSON schema validation, file reference resolution, result/error writing) and surface provider capability flags in the admin portal so operators know which aliases allow each operation.
5. **Tests & Docs** – Add handler/adaptor/batch tests covering happy paths and edge cases, then update `ROADMAP.md`, `CHANGELOG.md`, and user/admin/runtime docs to mark the endpoints complete, describe provider matrices, and show updated curl/config examples.

## Tasks

- Update multipart handlers in `backend/internal/httpserver/public/openai_routes.go` to accept OpenAI field aliases, enforce limits, and reuse shared validation/error helpers.
- Add per-operation pricing metadata parsing in `runImageOperation` so usage logging/budgets differentiate generations vs edits vs variations.
- Ensure every image-capable adapter implements or explicitly rejects edits/variations, wiring metadata knobs and documenting limitations in `docs/runtime/config.md`.
- Extend `backend/internal/batchworker/worker.go` to support `/v1/images/edits` and `/v1/images/variations`, including NDJSON schema validation and file reference handling.
- Create unit/integration tests for the updated handlers, adapters, and batch worker flows.
- Refresh `ROADMAP.md`, `CHANGELOG.md`, and the admin/user/runtime docs to reflect the completed Images API parity and configuration guidance.
