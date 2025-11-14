# Bootstrap Configuration Reference

The router loads an optional `bootstrap` section from `router.yaml` to pre-seed tenants, admin users, memberships, and API keys. This document summarises the supported fields and how they map into runtime behaviour.

```yaml
bootstrap:
  tenants:
    - name: "Acme Corp"
      status: "active"        # or "suspended"
  admin_users:
    - email: "admin@example.com"
      name: "Demo Admin"
      password: "admin-password"
  memberships:
    - tenant: "Acme Corp"
      email: "admin@example.com"
      role: "owner"            # owner > admin > viewer > user
  api_keys:
    - tenant: "Acme Corp"
      prefix: "sk-demo"
      secret: "my-secret"
      name: "Demo Key"
      rate_limit:
        requests_per_minute: 3000
        tokens_per_minute: 60000
        parallel_requests: 8
  tenant_limits:
    - tenant: "Acme Corp"
      limits:
        requests_per_minute: 4000
        tokens_per_minute: 80000
        parallel_requests: 16
  tenant_budgets:
    - tenant: "Acme Corp"
      budget_usd: 250.0
      warning_threshold: 0.85
      refresh_schedule: "weekly"
      alert_emails:
        - "billing@example.com"
      alert_webhooks: []
      alert_cooldown: "2h"
```

## Behaviour by Section

- **tenants** – Creates tenants if they do not exist. `status` defaults to `active`.
- **admin_users** – Upserts users and Argon2id-hashes the provided password. Every bootstrapped admin is automatically promoted to `is_super_admin=true`, bypassing tenant RBAC checks at runtime.
- **memberships** – Ensures the user holds the specified role for the tenant. Roles are ordered `owner > admin > viewer > user`; invalid values trigger startup errors.
- **api_keys** – Generates hashed API key secrets. When the key already exists the bootstrap step leaves it untouched. Rate limit overrides are recorded for the limiter service.
- **tenant_limits** – Seeds per-tenant rate limit overrides that apply to all API keys inside the tenant.
- **tenant_budgets** – Creates/updates tenant-specific budget overrides, including refresh schedules (calendar_month, weekly, rolling_Xd) and alert channels (email/webhook) with per-tenant cooldowns.

## Operational Notes

- Bootstrapping is idempotent: re-running the router with the same config retains existing users/keys and only updates mutable fields (password hash, rate limits, etc.).
- Secrets are never logged; ensure real credentials come from environment-specific config or secret management tooling.
- Removing entries from `bootstrap` does **not** delete them from the database. Use the admin API/console for destructive operations.
- Admin passwords can be omitted when relying purely on OIDC.
- Super admins created via `admin_users` still benefit from memberships for tenant-specific dashboards, but they no longer require them to perform privileged actions.

See `docs/backend-status.md` for broader backend capabilities and `docs/runtime/router.example.yaml` for a fully annotated configuration template.
