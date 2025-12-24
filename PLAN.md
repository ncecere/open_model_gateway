# Fine-Grained Tenant RBAC Plan

## Goal
Enable tenant owners/admins to manage day-to-day tenant operations without granting access to global admin controls.

## Scope
- Role capability mapping for owner/admin/member/viewer (no custom roles).
- Tenant-level permissions for member budgets, tenant limits, model attach/detach, and tenant guardrails.
- API and UI updates for the user portal to enforce capabilities.
- Data model changes for per-member budgets and tenant limits.
- Tests and documentation updates.

## Non-Goals
- No tenant control over tenant-wide budget caps.
- No tenant edits to model metadata (pricing, provider config, deployments).
- No changes to global guardrails beyond read-only visibility.

## Assumptions
- Central admin remains source of truth for tenant budgets and model metadata.
- Tenant attach/detach only applies to admin-approved model aliases.
- Guardrails are template-driven with tenant-level instances derived from global templates.

## Milestones
1. Role mapping + middleware enforcement
2. Member budget + tenant limits schema and APIs
3. Model attach/detach flow
4. Tenant guardrails scoped to templates
5. Docs + tests

## Dependencies
- Existing user portal auth and membership context.
- Admin model catalog and entitlement configuration.
- Guardrails framework and storage model (deferred until guardrails implementation lands).

## Open Questions
- Do tenant limits need a hard ceiling per tenant from central admins (global max caps)?
- Should member budget UI be scoped to personal vs shared tenant contexts?
- How should model attach/detach changes be audited for admin visibility?

## Acceptance Criteria
- Owners/admins can update per-member budgets and tenant limits via user APIs and UI.
- Owners/admins can attach/detach approved model aliases without editing metadata.
- Tenant guardrail management is limited to tenant-scoped policies derived from templates.
- Member/viewer roles cannot perform restricted actions; UI and API both enforce.
- Tests cover permission boundaries and negative cases.
