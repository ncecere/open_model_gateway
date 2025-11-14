# Fine-Grained Tenant RBAC Roadmap

## Summary
Move beyond the current owner/admin/viewer roles and introduce granular permissions so enterprises can enforce least-privilege access. Roles should govern both admin APIs and user portal actions.

## Implementation Overview

1. **Role Model**
   - Define a `permissions` table mapping role names to capability flags (manage tenants, view usage, rotate keys, change budgets, etc.).
   - Allow custom roles per tenant or reuse platform defaults.

2. **Enforcement**
   - Update middleware to check capabilities instead of hardcoded role checks (e.g., `requirePermission(c, "tenants.update")`).
   - Ensure both admin and user APIs respect these permissions.

3. **UI & APIs**
   - Admin UI gets a “Roles & Permissions” section to review, edit, and assign roles.
   - Invitation workflow lets admins choose a role when adding members.

## Usage Examples

### Creating a “Billing Analyst” Role
```bash
curl -X POST https://router.example.com/admin/roles \
  -H "Authorization: Bearer sk-admin" \
  -H "Content-Type: application/json" \
  -d '{
        "name": "billing_analyst",
        "permissions": ["usage.view", "budget.view"]
      }'
```
- Assign to a user so they can view spend dashboards but cannot modify budgets or keys.

### Assign Role to Member
```bash
curl -X PATCH https://router.example.com/admin/tenants/{tenant_id}/members/{user_id} \
  -H "Authorization: Bearer sk-admin" \
  -H "Content-Type: application/json" \
  -d '{"role": "billing_analyst"}'
```

## Implementation Details

| Area | Notes |
|------|-------|
| Schema | `roles` table + `role_permissions` join table. Tenants can override or inherit defaults. |
| Migration | Backfill existing members to new roles (owner/admin/viewer) to preserve behavior. |
| UI | Role picker in tenant member table; tooltip listing permissions. |
| Audit | Log role changes for compliance review. |

## Next Steps
1. Finalize permission taxonomy.
2. Implement schema + service layer.
3. Update middleware to check capability flags.
4. Build UI + API flows for managing roles.
