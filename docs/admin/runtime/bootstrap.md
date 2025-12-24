---
title: bootstrap data
description: Seed tenants and quotas from router
---
[**Open Model Gateway**](/) can seed critical runtime data during startup via the `bootstrap` block.

---

#### Review sections
| Entry | Purpose |
| --- | --- |
| `tenants[]` | Create initial tenants with statuses so suspended tenants stay blocked by default. |
| `admin_users[]` | Upsert super admins with Argon2id-hashed passwords (or omit when relying solely on OIDC). |
| `memberships[]` | Map admins to tenants with `owner`, `admin`, `viewer`, or `user` roles for UI scoping. |
| `api_keys[]` | Provision hashed API keys with scopes, optional prefixes/secrets, rate limit overrides, and budget overrides. |
| `tenant_limits[]` | Clamp tenant RPM, TPM, and parallel ceilings before applying per-key overrides. |
| `tenant_budgets[]` | Override tenant budgets, refresh schedule, alert recipients, and cooldowns. |
| `default_models[]` | Seed default alias entitlements for new personal tenants so self-serve users inherit curated catalogs. |

---

#### Configure sample
Use this pattern to bootstrap a demo tenant plus guarded quotas; reapply the file whenever secrets or limits change because the seeder is idempotent.

```yaml
bootstrap:
  tenants:
    - name: "Acme"
      status: "active"
  admin_users:
    - email: "admin@example.com"
      name: "Acme Admin"
      password: "change-me"
  memberships:
    - tenant: "Acme"
      email: "admin@example.com"
      role: "owner"
  api_keys:
    - tenant: "Acme"
      name: "acme-shared"
      scopes: ["chat", "embeddings", "images", "files", "batches"]
      rate_limit:
        requests_per_minute: 3000
        tokens_per_minute: 60000
        parallel_requests: 8
      budget:
        usd_limit: 500
  tenant_limits:
    - tenant: "Acme"
      requests_per_minute: 4000
      tokens_per_minute: 80000
      parallel_requests: 16
  tenant_budgets:
    - tenant: "Acme"
      budget_usd: 250
      warning_threshold: 0.85
      refresh_schedule: "weekly"
      alert_emails:
        - "billing@example.com"
      alert_webhooks: []
      alert_cooldown: "2h"
  default_models:
    - alias: "gpt-4o-mini"
      scope: "tenant"
```

---

#### Operate safely
Bootstrap never deletes rows, so remove stale tenants or keys via the Admin UI or API to avoid orphaned records.
Keep sensitive values (admin passwords, API key secrets) in environment-specific config or secret managers so commits stay scrubbed.

---

#### Research
- Mirrored behavior from `docs/runtime/bootstrap.md` and cross-referenced rate limit + budget overrides with `AGENTS.md` release notes.
- Validated key structure against `backend/internal/bootstrap` services to ensure field names match current structs.
