# Fine-Grained Tenant RBAC

## Summary
Refine the existing role system (owner, admin, member/viewer) so tenants can manage day‑to‑day operations without touching global admin features. Requirements:
- Keep the role list simple—reuse existing roles; **no custom roles**.
- Tenant budgets remain central-admin only (tenants can’t change the tenant-wide spend cap).
- Tenant owners/admins can adjust **per-member budgets** (API key allocations) and **tenant rate limits/concurrency caps**.
- Tenant owners/admins can **attach or detach** model aliases that system admins have approved for tenant use, but **cannot modify any model metadata** (pricing, deployments, provider config, etc.).
- Tenant owners/admins can configure tenant-specific guardrails, but cannot override global guardrails.

## Role Model
| Role | Capabilities |
| --- | --- |
| `owner` | Full tenant-scoped control (invite/remove members, adjust member budgets, attach/detach models, configure tenant rate limits, manage tenant guardrails). Cannot change tenant budget or global guardrails. |
| `admin` | Same as owner except cannot transfer ownership. |
| `member` | Use assigned models/API keys; optional usage view. |
| `viewer` | Read-only usage (optional). |

## Permissions Matrix (Key Points)
- `tenant_budget.manage`: **false** for tenant roles (only central admins).
- `member_budget.manage`: true for owners/admins.
- `tenant_limits.manage` (TPM/RPM/concurrency): true for owners/admins.
- `models.attach`: true for owners/admins (attach/detach approved aliases). **No ability to edit model info.**
- `guardrails.manage`: true for owners/admins, scoped to tenant-level guardrails only (global guardrails remain read-only).
- `models.edit`: false (renaming, pricing, provider metadata remain central admin only).

## Enforcement & UI
- Backend middleware checks these capabilities before executing user-portal endpoints.
- UI hides/locks controls when the user lacks permission (e.g., disable model-edit buttons; only show attach/detach toggles).
- Tenant guardrail UI allows owners/admins to create/edit tenant-specific policies but only from templates defined by system admins; global guardrails remain view-only.

## API Changes
- `/user/tenants/:id/members` – expose per-member budgets; require owner/admin role for updates.
- `/user/tenants/:id/limits` – allow owners/admins to adjust TPM/RPM/concurrency caps.
- `/user/tenants/:id/models` – attach/detach endpoints; no edit operations.
- Guardrail endpoints scoped to tenant-level policy storage; global guardrails are admin-only.

## Implementation Plan
### Phase 1 – Role Mapping & Middleware
1. **Define capability map** for the four roles, removing any plans for custom roles.
2. **Add middleware helpers** (`requireCapability`) that consult the role map before hitting tenant-scoped APIs.
3. **Migrate existing memberships** so owners/admins automatically inherit the new capabilities.

### Phase 2 – Member Budgets & Tenant Limits
1. Extend tenant-member tables with budget fields (monthly spend cap, token cap, etc.).
2. Build `/user/tenants/:id/members/:member_id/budget` CRUD endpoints + UI guarded by owner/admin checks.
3. Add `/user/tenants/:id/limits` API + UI so owners/admins can tweak tenant TPM/RPM/concurrency caps (within central-admin ceilings).

### Phase 3 – Model Attach/Detach Flow
1. Admins flag catalog aliases as tenant-assignable.
2. Tenants list available aliases and toggle attach/detach; no edit forms, only on/off.
3. Update usage/routing to respect tenant-specific model lists.

### Phase 4 – Tenant Guardrails
1. Allow owners/admins to create tenant-level guardrail policies derived from global templates.
2. Enforce that tenants cannot modify global guardrails.
3. UI + API for listing/applying tenant guardrails.
> **Deferred:** Guardrails are not implemented yet. Revisit this phase once guardrails storage and templates land.

### Phase 5 – Documentation & Testing
1. Update admin/user guides describing role abilities and restrictions.
2. Add integration tests for each capability (budget edits, limits, model assignment, guardrail management).
3. Update invite flows/tooltips to clarify role powers.

## Task Checklist
- [ ] Role-capability map + middleware helpers
- [ ] Membership migration + backfill scripts
- [ ] Member budget schema + API/UI
- [ ] Tenant limits schema + API/UI
- [ ] Model attach/detach API/UI + catalog flag
- [ ] Tenant guardrail API/UI scoped to templates
- [ ] Documentation & tooltips
- [ ] E2E tests covering each permission boundary

## Risks
- Role confusion → provide clear tooltips/docs describing what each role can and cannot do.
- Legacy tenants → migration ensures existing owners/admins inherit new permissions automatically.
- Security → ensure API rejects any attempt to edit model metadata or tenant budgets from tenant-scoped endpoints.
