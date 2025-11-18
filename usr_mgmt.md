# User Management & RBAC

This guide documents how identity, authentication, and role-based access control work inside Open Model Gateway. It captures the relationships between schema objects, runtime services, HTTP handlers, and bootstrap knobs so backend and frontend contributors can reason about the current system without spelunking through the entire tree.

## RBAC Overview

Roles are defined at two layers:

1. **Global (account scope)** – what the platform lets a user do across all tenants.
2. **Tenant (per-tenant scope)** – what a user may do inside a specific tenant they belong to.

### Global Roles (Account Scope)

| Role | Description | Admin Surface Access | User Surface Access |
| --- | --- | --- | --- |
| **Global User** (default) | Any authenticated account (local or OIDC). Upon first login we provision a personal tenant, assign the user as its owner, and seed it with global defaults. | May access `/admin/**` only when they also hold at least the required tenant role (owner/admin/viewer/user). No implicit cross-tenant rights. | Full access to their personal tenant plus any organization tenants they join. Capabilities are limited by the tenant role they hold. |
| **Global Admin** (“Super Admin”) | `users.is_super_admin = true`. Granted via bootstrap `admin_users[]` or mapped from OIDC `admin_roles`. | Bypasses tenant-level RBAC in admin handlers (`requireTenantRole`/`requireAnyRole` short-circuit), so they can manage every tenant, model, and setting. Still subject to global feature flags (e.g., local auth off). | Behaves like a global user in the user portal, but typically joins tenants only for auditability; personal tenant still exists for key management. |

Global admins inherit every capability of global users. The only distinction is the RBAC bypass inside the admin API; the user portal continues to rely on explicit memberships so personal tenants and audit trails remain intact.

### Tenant Roles (Per Tenant)

Tenant roles live in the `membership_role` enum (`owner > admin > viewer > user`). Each user can hold a different role per tenant.

| Role | Summary | Key Admin Portal Permissions | Key User Portal Permissions |
| --- | --- | --- | --- |
| **Owner** | Ultimate authority for a tenant. Every personal tenant auto-promotes its user to owner. | Rename tenant, change status, configure budgets/rate limits/models/routes, create/revoke API keys, and manage memberships (invite/remove, promote/demote). Can edit tenant-specific budget overrides and rate limits. | Can do everything admins can plus grant/revoke membership (including other owners) and configure tenant settings exposed through the user portal (API keys, membership, usage). |
| **Admin** | Day-to-day operator. Focused on API keys, usage, and operational settings. | Create/revoke API keys, list memberships, inspect usage/budgets/files/batches, tune rate limits where owner rights aren’t required. Cannot promote others to owner or delete tenants. | Create/revoke tenant API keys, invite/remove members (except promoting someone to owner or removing an owner), access usage dashboards, and manage non-destructive settings. |
| **Viewer** | Read-only auditor. | View tenant metadata, budgets, usage, API keys, and models but cannot mutate anything. Used by the admin UI to gate read-only pages. | View usage, API keys, and tenant dashboards without mutation rights. |
| **User** | Minimal role intended for callers who just need API access. | No admin-surface rights. Still allows issuing API requests via `/v1/*` when authenticated through an API key tied to the tenant. | Can see their own memberships and basic tenant context but cannot change keys or settings. |

The `admin/internal/rbac.AtLeast` helper enforces the `owner > admin > viewer > user` ordering, so requesting a `viewer` role automatically allows admins and owners.

## Data Model

- **Users** live in `backend/sql/schema/000_init.sql` with columns for `email`, `name`, `theme_preference`, `is_super_admin`, `personal_tenant_id`, and timestamps (`created_at`, `updated_at`, `last_login_at`). Every record may be linked to a personal tenant that exists solely for that user.
- **Tenants** carry `status` (`active` or `suspended`) and `kind` (`organization` vs `personal`). Personal tenants power the user portal and per-user API keys while organization tenants represent shared workspaces.
- **Memberships (`tenant_memberships`)** enforce a single role per `(tenant_id,user_id)` with the ordered enum `owner > admin > viewer > user`. Schema constraints guarantee uniqueness and cascading deletes.
- **API keys** belong to tenants and optionally point back to a user when `kind = personal`. A check constraint ensures personal keys always have an owner, while service keys may omit it.
- **Credentials (`user_credentials`)** store hashed Argon2id passwords for local auth as well as OIDC subjects. Each row records a `(provider, issuer, subject)` triple plus opaque metadata.

The authoritative SQL definitions live in `backend/sql/schema/000_init.sql` (core tables) and `backend/sql/schema/001_auth.sql` (credential store). SQLC-generated queries under `backend/sql/queries/*.sql` expose CRUD operations such as `GetUserByEmail`, `ListUserTenants`, `AddTenantMembership`, and `ListPersonalAPIKeysByUser`.

## Global Roles & Super Admins

Global privilege is represented by the boolean `users.is_super_admin`. The table above distinguishes *global users* (default) from *global admins* (“super admins”), but both are stored in the same `users` table. Super admins:

- Are seeded via `bootstrap.admin_users` (`docs/runtime/bootstrap.md:44-60`), which also hashes passwords and ensures personal tenants exist.
- Can be promoted automatically on every OIDC login when the IdP roles claim contains any of the configured `admin.admin_roles` (`docs/runtime/config.md:158-173`). The `AdminAuthService.syncUserAdminFlag` method (`backend/internal/auth/service.go:230-275`) keeps the DB flag in sync.
- Skip all tenant-level RBAC enforcement where `requireTenantRole` / `requireAnyRole` are used. The admin middleware checks `user.IsSuperAdmin` before consulting the `adminrbac` service (`backend/internal/httpserver/admin/admin_rbac.go:13-51`).

Despite the bypass, super admins still benefit from memberships for auditability and for the `/user` portal (see “User Portal Behaviour”).

## Authentication & Tokens

- `AdminAuthService` (`backend/internal/auth/service.go`) supports Argon2id-backed local login (`AuthenticateLocal`) and OIDC (`CompleteOIDCAuth`). Both flows invoke `PersonalService.EnsurePersonalTenant` before issuing tokens so every user has a tenant context.
- Access + refresh tokens come from a shared `TokenManager`. `/admin/**` and `/user/**` surfaces both call `AdminAuth.AuthorizeAccessToken` from their middleware (`backend/internal/httpserver/admin/admin_middleware.go`, `backend/internal/httpserver/user/middleware.go`), so a single login session powers both portals.
- Refresh tokens are rotated via `/admin/auth/refresh` using cookies or payload-provided tokens (`backend/internal/httpserver/admin/admin_routes.go:130-180`).
- Password management: `AdminAuthService.UpsertLocalPassword` writes to `user_credentials`, admin-side user creation accepts an optional password, and the user portal exposes `/user/profile/password` for self-service rotation (Argon2id verification + update).

## Personal Tenants & Default Models

`accounts.PersonalService` (`backend/internal/accounts/personal.go`) guarantees a `tenant_kind = personal` record with owner membership for every user (i.e., every global user automatically gets a personal tenant the first time they authenticate):

1. When `EnsurePersonalTenant` runs, it creates (or fetches) the tenant named `personal:<user-uuid>`, assigns the user as owner, writes `users.personal_tenant_id`, and seeds default models via `seedDefaultModels`.
2. The service registers callbacks for cache invalidation (`SetTenantModelUpdater`) so routing state stays consistent.
3. `SyncDefaultModels` reapplies updates whenever global defaults change via `/admin/settings/default-models` (`backend/internal/httpserver/admin/admin_default_models.go`).

This service is invoked by bootstrap seeding, local/OIDC login, admin user creation, and user portal API key generation to ensure the personal context always exists.

### Global Defaults & Tenant Inheritance

Administrators manage the “gateway defaults” that flow into new tenants (especially personal tenants) via the admin surface:

| Surface | Source of Truth | How Personal Tenants Inherit | How Tenants Override |
| --- | --- | --- | --- |
| **Model Allowlist** | `/admin/settings/default-models` backed by `default_models` + catalog entries | `PersonalService.seedDefaultModels` copies the alias list into every personal tenant upon creation (and re-syncs via `SyncDefaultModels`). | `/admin/tenants/:id/models` (owner role) or bootstrap `tenant_models` entries can replace the list. |
| **Budget Defaults** | `/admin/budgets/default` updating `budget_defaults` | `EffectiveTenantBudget` merges the default USD, warning threshold, refresh schedule, and alert channel config whenever a tenant lacks an override; the same values drive personal tenants. | `/admin/budgets/overrides` (owner role) or bootstrap `tenant_budgets[]` entries persist overrides in `tenant_budget_overrides`. |
| **Rate Limit Defaults** | `/admin/rate-limits/defaults` writing `rate_limit_defaults` | Redis limiter loads the defaults and applies them to every tenant/key unless overrides exist; personal tenants therefore honor the same RPM/TPM/parallel caps. | `/admin/tenants/:id/rate-limits` (owner role) stores overrides in `tenant_rate_limits`; per-key overrides ride along with API key creation. |
| **Alert Channels / Cooldowns** | Part of the budget default/override payload | Personal tenants inherit the alert email/webhook destinations and cooldowns from the budget defaults unless a tenant override is recorded. | Owners can set custom alert destinations/cooldowns via `/admin/budgets/overrides`. |
| **Rate Limit + Budget Bootstrap** | `bootstrap.tenant_limits[]` / `bootstrap.tenant_budgets[]` | Seeder populates the same tables the admin API uses, so new personal tenants continue to consume the merged view. | Same as above—overrides override defaults. |

This inheritance pattern ensures personal tenants “just work” after login while still letting operators fine-tune organization tenants through the admin UI or bootstrap.

## Tenant Memberships & Roles

- Enum order is defined near the top of `backend/sql/schema/000_init.sql`. The helper package `backend/internal/rbac` provides `ParseRole`, `AtLeast`, `Ensure`, and `EnsureAny`, giving everything a single source of truth for role comparisons (see the table in “RBAC Overview” for the qualitative mapping).
- Membership CRUD flows:
  - Admin API (`/admin/tenants/:id/memberships`): owner role required to upsert/remove, admin role required to list. Admin invites now require the user to exist already (create the account via `/admin/users` first) and record audit logs for both read + write operations.
  - User portal (`/user/tenants/:tenantID/memberships`): owner/admins can list, invite, and remove members for tenants they belong to. The handler reuses `admintenant.Service` for persistence but enforces additional safeguards such as “admins cannot mint new owners” and “you can’t remove yourself.”
- Bootstrap config supports declarative `memberships[]` entries that will upsert the requested role on startup; invalid role names abort startup (`docs/runtime/bootstrap.md:44-60`, `backend/internal/runtime/bootstrap/bootstrap.go:117-189`).

## Admin RBAC Enforcement

Every protected admin route uses one of two helpers defined in `backend/internal/httpserver/admin/admin_rbac.go`:

- `requireAnyRole` ensures the caller has at least a specific role across *any* tenant membership (`rbac.EnsureAny`). This pattern is used for global resources such as `/admin/users`, `/admin/settings/default-models`, `/admin/budgets/default`, and provider catalog endpoints; the minimum role varies between viewer (read-only) and admin (mutations).
- `requireTenantRole` enforces a role within a specific tenant (`rbac.Ensure`). It powers all tenant-specific endpoints (`/admin/tenants/:id/...`, `/admin/usage/:tenantID/...`, etc.). Owners can change memberships and models, admins can manage API keys and rate limits, and viewers can fetch read-only data (keys, budgets, usage).

If the `AdminRBAC` service is misconfigured the middleware returns `500 rbac service unavailable`, so wiring the service into the container (`backend/internal/app/container.go:61,245`) is essential.

## Admin Surface Overview

`backend/internal/httpserver/admin/router.go` wires all `/admin` routes. RBAC expectations per area:

| Area | Example Endpoints | Minimum Role |
| --- | --- | --- |
| Users | `GET/POST /admin/users`, `GET /admin/users/:id/tenants` | Admin on any tenant |
| Default Models | `GET /admin/settings/default-models` (viewer), `POST/DELETE` (admin) | Viewer/admin combos |
| Tenants (list/personal list) | `GET /admin/tenants`, `GET /admin/tenants/personal` | Viewer on any tenant (super admins bypass) |
| Tenant settings | `/admin/tenants/:id` (rename, status, models, budgets, rate limits, API keys, batches) | Viewer (read-only data), Admin (mutations), Owner (memberships/models) |
| Budgets | `/admin/budgets/default`, `/admin/budgets/overrides` | Admin on any tenant |
| Rate limits | `/admin/rate-limits/defaults`, `/admin/tenants/:id/rate-limits` | Viewer for reads, Admin for writes |
| Usage dashboards | `/admin/usage/*` | Viewer per tenant or any-role gate depending on scope |
| Provider catalog & audit logs | `/admin/providers`, `/admin/audit/logs` | Viewer on any tenant |

All handlers share middleware for JWT validation (`admin_middleware.go`) which places the user + ID into the request context for downstream audit logging and RBAC checks.

## User Portal Behaviour

The `/user` API surface (`backend/internal/httpserver/user/router.go`) consumes the same auth tokens but focuses on per-user workflows:

- **Profile endpoints** (`GET/PATCH /user/profile`, `POST /user/profile/password`) provide personal metadata, `is_super_admin`, personal tenant ID, and whether password updates are allowed (presence of local credentials).
- **Tenant views** (`GET /user/tenants`, `GET /user/tenants/:tenantID/summary`) rely on `tenant.Service` to read memberships and budgets. Every caller must already be a member; super admins do not bypass these checks, so they must join a tenant before they can manage it through the user portal.
- **API keys** come in two flavours:
  - Personal keys (`/user/api-keys`) always target the personal tenant, auto-creating it if needed, and return secret/token material for the caller.
  - Shared tenant keys (`/user/tenants/:tenantID/api-keys`) require owner/admin role (`canManageTenantKeys`). Admins can create standard service keys, revoke them, and fetch usage summaries.
- **Membership management** uses similar owner/admin guards. Admins can invite/remove viewers/users/admins, but only owners can promote someone to owner or remove another owner.
- **Usage + Files + Batches** reuse the same services as the admin surface but scope everything to the caller’s memberships.

Frontend components (`backend/frontend/src/components/AppLayout.tsx`, `backend/frontend/src/apps/user/components/UserLayout.tsx`) read `user.is_super_admin` from the auth payload to toggle “Return to User Portal”/“Open Admin Portal” buttons, but they still rely on backend RBAC for real enforcement.

## Bootstrap & Configuration Hooks

Key configuration surfaces that influence user management:

- `bootstrap.*` (`docs/runtime/bootstrap.md`, `backend/internal/runtime/bootstrap/bootstrap.go`) can declare tenants, admin users, memberships, API keys, tenant budgets, and rate limits. The seeder is idempotent and errors when supplied with invalid roles.
- `admin.*` config toggles local auth, OIDC issuers, JWT secret/TTL, cookie names, and role-based admission controls. Environment variables follow the `ROUTER_ADMIN_*` pattern.
- `budgets.*`, `rate_limits.*`, and model defaults indirectly affect user-facing workflows (validating API key quotas, calculating usage windows, and seeding personal tenants).

## Open Questions & Observations

- `GET /admin/tenants` and `/admin/tenants/personal` currently do **not** call `requireAnyRole`, which means any authenticated admin token (even without memberships) can list every tenant. Decide whether that’s intentional (super-admin-like default) or if we should gate it.
- `/user` routes do not grant implicit authority to super admins; they must either belong to a tenant or go through the admin UI/API to add themselves. This keeps the user portal simple but may surprise operators expecting global access everywhere.
- **Audit coverage:** Mutating admin endpoints already call `recordAudit`, but read-heavy operations (listing tenants, listing API keys, fetching budgets) currently do not emit audit events. If compliance requires “read” trails, we should add lightweight audit entries or structured logs for those code paths.

Keep this document updated whenever the RBAC matrix or authentication flows change so future contributors can understand the intent behind each check.

## Follow-Up Actions

The discussion surfaced several concrete improvements to tackle next:

- **Gate tenant listings** – Update `/admin/tenants` + `/admin/tenants/personal` handlers to call `requireAnyRole(..., viewer)` (super admins will still bypass). Document that the user portal intentionally shows only tenants the user belongs to.
- **Audit read paths** – Decide which admin/user endpoints should emit read events (e.g., listing API keys, fetching budgets) and instrument them with `recordAudit` or structured logs referencing the actor/tenant.
- **Admin UX enhancements** – Surface password fields on the Users page (backend already supports it) and change tenant membership invites to select from existing users rather than auto-creating accounts. Owners/Admins will continue to manage memberships in both admin and user portals, but the UX needs to reflect the new constraints.
