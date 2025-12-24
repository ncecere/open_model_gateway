# Fine-Grained Tenant RBAC Tasks

## Phase 1 - Role Mapping and Middleware
- [x] Define role capability map for owner/admin/member/viewer
- [x] Add middleware helper to enforce capability checks on user routes
- [x] Ensure existing memberships map cleanly to new capabilities
- [x] Add audit log entries for permission-gated changes

## Phase 2 - Member Budgets and Tenant Limits
- [x] Add schema for per-member budget fields (monthly cap, token cap, etc.)
- [x] Add schema for tenant limits (RPM, TPM, concurrency)
- [x] Implement `/user/tenants/:id/members` budget read/update endpoints
- [x] Implement `/user/tenants/:id/limits` read/update endpoints
- [x] Enforce limits within central-admin ceilings
- [x] Update user portal UI to edit member budgets and tenant limits

## Phase 3 - Model Attach/Detach
- [x] Add admin flag for tenant-assignable model aliases
- [x] Implement `/user/tenants/:id/models` list + attach/detach endpoints
- [x] Update routing to respect tenant-specific model lists
- [x] Update user portal UI to toggle attach/detach only (no edits)

## Phase 4 - Tenant Guardrails
- [ ] Define tenant guardrail storage and link to template identifiers (deferred: guardrails not implemented)
- [ ] Implement tenant-scoped guardrail list/create/update endpoints (deferred: guardrails not implemented)
- [ ] Ensure global guardrails remain read-only in tenant context (deferred: guardrails not implemented)
- [ ] Add user portal UI for tenant guardrail management (deferred: guardrails not implemented)

## Phase 5 - Documentation and Tests
- [x] Update user docs for role capabilities and restrictions
- [x] Update admin docs to clarify tenant-level changes
- [x] Add integration tests for each capability boundary
- [x] Add UI tests for visibility and disabled controls

## Phase 6 - QA and Rollout
- [x] Run end-to-end checks across user portal flows
- [ ] Verify auditing and error messaging for unauthorized actions
- [ ] Collect feedback from internal users and adjust tooltips
- [x] Run backend test targets for user RBAC changes (user http handlers, app request context)
- [x] Run frontend test targets (user tenant page, model catalog form)
- [x] Validate owner/admin/viewer flows for member budgets, tenant limits, and model attach/detach
