# Tenants and Users

Tenants represent isolated customer workspaces. Users can belong to one or more
tenants and receive API keys scoped to those tenants.

## Create a Tenant

1. Open **Admin -> Tenants**.
2. Click **Create tenant**.
3. Fill in:
   - **Name** and optional description.
   - **Status**: active or suspended.
   - **Budget overrides** (optional).
   - **Rate limit overrides** (optional).
4. Save to create the tenant.

Suspended tenants cannot run `/v1` requests; their keys are blocked until reactivated.

## Assign Default Models to a Tenant

Tenant access is limited by the model catalog and default model settings.

1. Open **Admin -> Settings** and confirm default models are set.
2. In **Admin -> Tenants**, open a tenant and review model access.
3. Add or remove model aliases as needed.

## Add Users to a Tenant

1. In **Admin -> Tenants**, open the tenant.
2. Go to **Members** or **Users**.
3. Invite users by email or select existing users.
4. Assign a role:
   - **Owner**: full access including role changes.
   - **Admin**: management access without ownership actions.
   - **Member**: can use and manage API keys (if permitted).

If a user is not a member of a tenant, they cannot issue or use tenant-scoped keys.

## Personal Tenants

Each user has a personal tenant created automatically. The user portal uses this
scope by default, and users can switch to other tenants they belong to.

## User Accounts

User creation depends on auth mode:

- Local auth: admins invite users and set passwords.
- SSO (OIDC): users authenticate via the identity provider, then appear in the directory.

## Offboarding a User

1. Remove them from tenant memberships.
2. Revoke or rotate any API keys they issued.
3. Deactivate the user if local auth is enabled.

## Tips

- Use tenant overrides as guardrails, then refine per-key.
- Keep tenant names stable; they appear in usage exports and audit logs.
