# Tenant Owner Guide

Tenant owners (or delegated tenant admins) are responsible for modeling use cases inside a single tenant. This guide explains how to coordinate with platform admins, attach models, issue scoped API keys, and monitor spend.

## Know Your Role

- You control who can access the tenant (owners/admins/viewers/users) and which API keys exist.
- You can request new models from platform admins and toggle tenant-assignable aliases when permitted.
- You configure tenant-specific budgets, alerts, and rate limits (subject to platform defaults).
- You monitor usage for your org and keep workloads within budget/rate limits.

## Onboarding Checklist

1. Accept the invitation email from the platform admin (local credentials or SSO).
2. Sign in to the user portal at `/` and switch to the tenant you own.
3. If you have admin access, you can also use the admin portal at `/admin/ui` for platform-wide settings.
4. Review pre-provisioned models/keys/budgets seeded via bootstrap.
5. Capture the shared API key strategy (per application vs per user) and budget owners.

![TODO: User portal tenant switcher showing shared tenant](../assets/screenshots/user-portal-tenant-switcher.png)

## Manage Tenant Members

| Action | Path |
| --- | --- |
| Invite a teammate | **Tenants -> Members -> Invite** (choose role) |
| Change a role | **Tenants -> Members -> Edit** |
| Remove access | Delete membership from the same view |
| Automate | `POST /admin/tenants/{tenantID}/memberships` (requires platform admin token) |

![TODO: Tenant members tab with invite dialog](../assets/screenshots/tenant-members-invite.png)

Roles:
- **Owner** - full control over tenant settings, budgets, and keys.
- **Admin** - manage models, keys, and members but cannot delete the tenant.
- **Viewer** - read-only dashboards.
- **User** - scoped to API key visibility only.

## Models & Providers

1. Browse **Models** for tenant-assignable aliases. Each entry lists provider, deployment ID, supported modalities, rate limits, budgets, and health.
2. Enable/disable models for your tenant via **Models -> Assignments**. Requests to non-assigned aliases return `404 model_not_found`.
3. If you need a new model or provider:
   - Open an ops request with the alias/provider details.
   - Platform admins will attach the model globally or flag it tenant-assignable, then you can enable it.
4. Check `docs/reference/requests/README.md` for sample payloads and headers.

![TODO: Tenant model assignments panel](../assets/screenshots/tenant-model-assignments.png)

## API Keys

- Generate keys via **Tenants -> API Keys -> Create**. Provide a name, optional description, and overrides for budgets/rate limits.
- Secrets are only shown once; store them securely (vault, secret manager).
- Revoke keys when staff change roles or incidents occur.
- Rotate keys regularly and update applications with the new secrets before revoking the old ones.

![TODO: Tenant API key create modal](../assets/screenshots/tenant-api-key-create.png)

## Budgets, Rate Limits, and Alerts

1. Define tenant-level budgets under **Tenants -> Budgets**. Configure:
   - USD amount
   - Refresh schedule (rolling/weekly/calendar)
   - Warning threshold (percentage)
   - Alert channels (email/webhook)
2. Override per-key budgets when issuing or editing API keys. Per-key budgets never exceed the tenant budget.
3. Adjust rate limits (RPM/TPM/parallel) under the same screens. Platform defaults set the ceiling; tenant overrides can only go lower unless a platform admin raises them.
4. Test alerts with the **Send test alert** button to verify recipients and webhook endpoints.

![TODO: Tenant budget and limit overrides](../assets/screenshots/tenant-budgets-limits.png)

## Monitoring Usage

- **Usage dashboard**: filter by model, API key, or timeframe. Monitor token counts, latency, provider health, and 4xx/5xx rates.
- **Budget cards**: show remaining USD and next refresh. Investigate spikes early to avoid enforced `402 budget_exceeded` errors.
- **Response headers**: propagate to clients via `X-Budget-*` and `X-RateLimit-*` headers for client-side telemetry.

![TODO: Tenant usage dashboard filters](../assets/screenshots/tenant-usage-dashboard.png)

## Working with Users

- Provide onboarding instructions from `../user/guide.md` to every developer who needs access.
- Encourage users to reference the curl examples to self-serve request debugging before escalating.
- When budgets tighten, coordinate with the platform admin to adjust tenant or key caps.

## Troubleshooting

| Symptom | Likely Cause | Action |
| --- | --- | --- |
| Developers see `401 unauthorized` | Key revoked, rotated, or copied incorrectly | Reissue key or confirm secret storage practices |
| `402 budget_exceeded` | Tenant or key budgets depleted | Increase budgets (if approved) or wait for refresh |
| `404 model_not_found` | Alias not assigned | Enable the alias via **Models**, or request a new alias |
| `429 rate_limit_exceeded` | Hitting RPM/TPM caps | Increase per-key limits or shard traffic |
| `provider_unavailable` | Provider outage/failover exhaustion | Check administrator incident communications and fallback aliases |

If the issue requires platform intervention, capture tenant ID, key ID, request ID, and timestamps before escalating so admins can trace usage rows quickly.
