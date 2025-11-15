# API Key Self-Service Limits

## Plan
- Extend the admin + user API key creation flows so callers can optionally specify budget overrides and RPM/TPM/parallel caps at creation time (and update them later) instead of relying solely on bootstrap/time-of-day operator tweaks.
- Enforce hierarchical ceilings: personal keys may not exceed the global defaults assigned to that user, while tenant keys must sit at or below the tenant’s configured limits (falling back to global defaults when no tenant override exists).
- Update database schema + services so per-key rate limit overrides (currently bootstrap-only) are persisted via SQLC and exposed in responses, along with budget override metadata.
- Refresh both the admin and user portals with form controls for budgets + rate limits, inheriting and displaying the effective defaults so users know the maximum allowed values before submitting.
- Add validation/tests to cover limit inheritance, API error paths, and UI behaviors; document the new workflow so operators understand how tenant ceilings interact with API key overrides.

## Checklist
- [x] Schema & queries: add nullable budget/rate limit fields to `api_keys` (or dedicated tables) and regenerate SQLC so service layers can persist them outside bootstrap.
- [x] Backend services & handlers: update admin/user API key create/update endpoints to accept `{budget_usd, warning_threshold, requests_per_minute, tokens_per_minute, parallel_requests}` and enforce the correct ceilings (global default → tenant override → key override).
- [x] Runtime limiter integration: ensure `Container.KeyRateLimits` loads from the new storage so runtime enforcement reflects user-configured values immediately.
- [x] Admin portal UI: expose budget + RPM/TPM/parallel inputs in the tenant API key create/edit dialogs, showing inherited max values and validation errors.
- [x] User portal UI: allow personal-tenant users to set the same controls (bounded by their defaults) when creating/rotating keys.
- [x] Tests: add backend unit tests for validation/enforcement plus frontend form tests (or e2e coverage) to lock the new workflows.
- [x] Docs: update admin/user guides + changelog to describe the self-service API key limits and inheritance rules.
