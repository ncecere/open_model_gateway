# Batch API Parity Tasks

## 1. Spec & Requirements Alignment
- Pull Official OpenAI/Azure batch docs, capture request/response fields, statuses, pagination semantics.
- Define lifecycle mapping (validating, in_progress, finalizing, completed, failed, cancelling, cancelled, expired) and allowed completion windows.
- Record decisions in `docs/runtime` so backend/frontend agents stay synced.

## 2. Database & SQLC Updates
- Extend `batches` table with `errors JSONB`, `cancelling_at`, `expired_at`, plus index for cursor pagination.
- Update SQL queries (list/get/cancel/finalize) and regenerate SQLC models.
- Add cursor-based list query (`limit+1`, `after_id`) to support `has_more`/`first_id`/`last_id`.

## 3. Batch Service Enhancements
- Set initial status to `validating`, clamp completion windows, preserve max concurrency.
- Add helpers for each lifecycle transition (in-progress, finalizing, cancelling, expired, etc.).
- Store validation/runtime errors in the new JSON column so `/v1/batches/:id` mirrors OpenAI.
- Expand `sanitizeEndpoint` to cover all currently supported OpenAI endpoints.

## 4. Worker Runtime & Concurrency
- Honor `batch.MaxConcurrency` by introducing a goroutine pool for item processing.
- Update status transitions (`validating`→`in_progress`→`finalizing`→terminal) and write timestamps accordingly.
- Handle cancellations gracefully (switch to `cancelling`, finish in-flight items, then mark `cancelled`).
- Track provider status codes/request IDs, feed them into NDJSON writer, and adopt the OpenAI response/error line format.
- Add expiry sweep that marks overdue batches as `expired`.

## 5. HTTP Surface Parity
- Update `/v1/batches` JSON to include spec fields (`errors`, `has_more`, `first_id`, `last_id`, `cancelling_at`, `expired_at`, etc.).
- Accept `limit` + `after` query params and return cursor metadata.
- Ensure cancel route returns the new fields, and document `/files/:id/content` for outputs.
- Validate request bodies strictly (metadata limits, allowed completion windows, endpoint checks).

## 6. Testing & Docs
- Add unit tests for service transitions, concurrency enforcement, NDJSON formatting.
- Add HTTP integration tests covering create/list/get/cancel/output with spec-compliant assertions.
- Update `docs/user/guide.md`, `docs/admin/guide.md`, and `deploy/router.example.yaml` to describe the new behavior.
- Coordinate with frontend agent to consume the richer payload (status timeline, pagination, error display).

## 7. Follow-Up (Optional)
- Wire `/v1/responses` endpoint once base parity lands so batches can target it.
- Add SDK smoke tests (Node/Python) to ensure OpenAI clients accept the gateway responses.
