# Refactor Task List

1. Baseline check: confirm current build status, and note any failing tests before refactors; this is the comparison point for a "no behavior change" outcome.

2. Usage service split: create `types.go`, `window.go`, `user.go`, `admin.go`, `compare.go`, `daily.go`, and `helpers.go` under `backend/internal/services/usage/`; move existing types and functions into the new files; update imports; run gofmt.

3. Usage compile check: build or test the `backend` package to ensure no missing symbols or import cycles were introduced by the split.

4. OpenAI public routes split: create endpoint files (`openai_models.go`, `openai_chat.go`, `openai_responses.go`, `openai_embeddings.go`, `openai_images.go`, `openai_moderations.go`), plus `openai_types.go` and `openai_handler.go`; move DTOs and handlers with no logic changes; update imports.

5. OpenAI compile check: compile the `backend/internal/httpserver/public` package to confirm the route split builds cleanly.

6. Batch worker split: keep orchestration in `worker.go` and move per-endpoint execution into `worker_chat.go`, `worker_responses.go`, `worker_embeddings.go`, `worker_moderations.go`, and `worker_images.go`; add `worker_helpers.go` for shared logic; ensure image edit/variation handlers stay consistent.

7. Batch worker compile check: build or test the `backend/internal/batchworker` package to verify routing and helpers still wire correctly.

8. Executor helper extraction: add private helpers for route selection and capability filtering in `backend/internal/executor/`; replace duplicate logic in Chat/Image/Embed/Moderate with the helper calls; verify error messages remain identical.

9. Admin tenant handler split: create `admin_tenants_routes.go`, `admin_tenants_types.go`, and feature-specific handler files (core, budgets, rate limits, models, memberships, batches, keys); move code and DTOs without behavior changes; update router registration.

10. OpenAPI spec file: create `docs/openapi/openapi.yaml` and document `/v1/*`, `/admin/*`, and `/user/*` endpoints (start with core routes if needed); keep models consistent with existing request/response shapes.

11. Serve OpenAPI spec: embed the spec with `go:embed` and expose `GET /openapi.yaml` and `GET /openapi.json` from the backend.

12. Scalar UI: add `GET /docs` that serves Scalar and points to `/openapi.yaml` (or `/openapi.json`); keep this read-only with no auth for now.

13. Backend full build: run the backend test/compile suite to confirm the full backend still builds after all splits and OpenAPI additions.

14. Frontend dependency add: add `react-hook-form` to `backend/frontend/package.json` and update `bun.lock`.

15. Settings form refactor: create a `src/pages/settings/` folder (or `src/features/settings/` if preferred); split each Settings tab into its own component; create a form hook that loads defaults and calls `reset`; wrap the page in `FormProvider`; keep payloads identical.

16. Settings regression check: confirm default values, validation, and submit handlers match previous behavior; verify admin key creation and alert test flows still work.

17. Shared usage utilities: create `src/features/usage/hooks/useUsageRange.ts`, `src/features/usage/components/UsageFilters.tsx`, `src/features/usage/components/UsageDailyTable.tsx`, and `src/features/usage/formatters.ts`; move shared logic and formatting from admin/user usage pages.

18. Admin usage page update: replace duplicated date/selection logic and daily table rendering with the new shared helpers; verify query parameters and filters stay unchanged.

19. User usage page update: reuse the shared helpers and remove duplicate range/selection logic; keep the user scope behavior identical.

20. Shared selection hook: add `src/hooks/useDefaultSelection.ts` and apply it where repeated default-selection effects exist in usage pages.

21. Frontend build/test: run `bun test` and `bun run build` (or project equivalents) to confirm the refactor compiles and tests pass.

22. Manual smoke check: open Settings and Usage pages in both admin and user apps; verify data, filters, and actions behave the same as before.
