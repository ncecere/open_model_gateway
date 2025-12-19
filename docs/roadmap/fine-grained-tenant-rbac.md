# Fine-Grained Tenant RBAC

## Summary
Upgrade the coarse owner/admin/viewer roles to a flexible RBAC system so large teams can delegate specific permissions (model management, budget edits, log access) without granting full admin rights. Support per-tenant roles plus custom policies for future growth.

## Implementation Plan

### Role Model
- Extend `membership_role` to include new standard roles (e.g., `owner`, `admin`, `operator`, `analyst`, `support`, `viewer`).
- Introduce a `permissions` table mapping roles to capabilities (CRUD flags for models, budgets, members, keys, usage exports, guardrails, etc.).
- Allow custom roles per tenant (optional) stored in `tenant_roles` with editable permission sets.

### Enforcement
1. Backend middleware checks role capabilities before executing admin/user endpoints.
2. UI hides/disables actions based on current role.
3. Audit logs capture role changes, permission grants, and sensitive actions.

### Admin UX
- Role management panel under Tenants → Members tab:
  - Assign built-in roles.
  - Define custom role (name + permissions checkboxes).
- Invite flows show capability descriptions for each role.

### API Changes
- Update `/admin/tenants/:id/members` and `/user/tenants/:id/members` endpoints to accept role IDs.
- Provide `/admin/roles` endpoints for listing/updating custom roles.

## Example Permissions
| Capability | Description |
| --- | --- |
| `models.manage` | Create/edit catalog entries for the tenant. |
| `budgets.update` | Change tenant budgets/rate limits. |
| `keys.manage` | Issue/rotate/revoke API keys. |
| `usage.view` | View usage/billing data. |
| `guardrails.manage` | Configure guardrail policies. |

## Components Needed
- DB migrations for roles/permissions.
- Middleware + helper functions to check permissions (`hasPermission(ctx, "models.manage")`).
- Portal updates (role selector, capability tooltips, guardrails around UI actions).
- Docs describing role definitions and how to configure custom roles.

## Risks
- Complexity creep → keep built-in roles simple and add custom roles as optional advanced feature.
- Backwards compatibility → map existing roles to new permission sets during migration.

## Next Steps
1. Design permission matrix and default roles.
2. Implement backend enforcement + migrations.
3. Update UIs + invite flows.
4. Document roles and capability mapping.
