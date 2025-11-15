# Files API Parity

## Plan
- Align with the latest OpenAI Files spec (`/v1/files` CRUD + `/v1/files/:id/content` and `/v1/uploads`) so tenants can upload/list/download/delete files using the same request/response shapes.
- Extend storage metadata so every file tracks purpose, status, status_details, deleted flags, TTL, checksum, and a cursor-friendly ID for pagination.
- Upgrade the HTTP handlers to support cursor pagination, filtering, soft deletes, and chunked upload sessions while continuing to stream large uploads/downloads through the existing blob layer.
- Add a background sweeper that enforces TTL (removes expired objects and marks them deleted) so lists stay accurate.
- Document the workflow (config, CLI examples) and add automated tests covering upload → list → download → delete flows plus error cases (max size, invalid purpose, expired file access).

## Checklist
- [x] Spec review: summarize the OpenAI Files contract (endpoints, fields like `status`, allowed `purpose` values, pagination via `limit/after`, and `deleted` behavior) and share it with the team.
- [x] Schema/storage: update the `files` table (status/status_details/deleted), ensure the blob layer exposes metadata needed for OpenAI responses, and add any missing indexes for cursor pagination or purpose filtering.
- [x] HTTP handlers: implement OpenAI-compatible responses for `GET/POST /v1/files`, `GET /v1/files/:id`, `GET /v1/files/:id/content`, `DELETE /v1/files/:id`, and add initial support for `/v1/uploads` (or return a structured 501 until chunking lands). Include cursor pagination and `purpose` filtering.
- [x] TTL sweeper + service logic: add a scheduled job that deletes/marks expired files, set `status=deleted` on removal, and ensure the service methods enforce the new validation rules (purpose allowlist, TTL, max size).
- [x] Tests: add unit tests for the files service + blob storage adapters, plus HTTP handler tests covering upload/list/download/delete, pagination, and error handling.
- [x] Docs + changelog: update runtime docs, admin/user guides, and `CHANGELOG.md` to describe the new Files API compatibility and how to configure/use it (include curl examples).
