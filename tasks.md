# Moderations API Implementation Plan

## Goals
- Deliver `/v1/moderations` parity with the current OpenAI REST spec across sync requests and batch jobs.
- Wire native OpenAI/Azure adapters plus catalog/bootstrap defaults so tenants can target moderation aliases immediately.
- Update docs/tests/changelog to reflect the new capability (policy/guardrail hooks tracked separately).

## Tasks
1. **Spec & Docs**
   - Capture request/response schema (input validation, `results[].categories`, `category_scores`, `usage`) in `docs/runtime/moderations.md`.
   - Note supported providers, tenant enablement expectations, and batch compatibility; link from README/ROADMAP.
2. **Core Types & Routing**
   - Add `models/moderation.go` with typed structs + helper to parse string/array inputs.
   - Extend provider interfaces/route struct with `Moderations` support and allow catalog entries to mark moderation aliases/pricing.
   - Seed sample config (bootstrap/default models) with `omni-moderation-latest` so tenants can opt in.
3. **Adapter Plumbing**
   - Implement `Moderate` on `adapters/openai` and `adapters/azureopenai` using the official SDK, converting into the shared model types and populating usage.
   - Ensure provider definitions advertise the new capability so the factory registers routes correctly.
4. **HTTP Endpoint**
   - Register `POST /v1/moderations` in `public/router.go`.
   - Implement handler: parse OpenAI-compatible payload, enforce budgets/rate limits/model access, execute against moderation-capable routes, record usage, return OpenAI-formatted JSON, and cache via idempotency key when provided.
5. **Batch Worker Integration**
   - Allow `/v1/moderations` jobs inside the batch worker, reusing the same validation/execution logic and success/error logging.
   - Update `docs/runtime/batches.md` to reflect the added support.
6. **Validation & Docs**
   - Add unit tests for input parsing, adapter conversion, and handler happy/edge paths (budget/rate/error cases).
   - Update `CHANGELOG.md`, `ROADMAP.md`, and README surface matrix to call out the new endpoint.

> NOTE: Guardrail/policy engine work remains out of scope; a separate task list will track those requirements.
