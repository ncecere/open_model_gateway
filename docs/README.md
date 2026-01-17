# User documentation

Use this directory to onboard every persona that touches Open Model Gateway. Pick your role, read the matching guide, and keep the shared references nearby whenever you run workloads or support end users.

## Directory map

| Path | Audience | Description |
| --- | --- | --- |
| `admin/install.md` | System administrators | Install, configure, and upgrade the router via release bundles, Docker, or Kubernetes. |
| `admin/guide.md` | Platform administrators | Day-2 operations for tenants, catalogs, budgets, provider health, and incident response. |
| `admin/runtime/README.md` | Platform administrators | Runtime configuration map and annotated router.yaml references. |
| `tenant/guide.md` | Tenant owners / delegated admins | Invite members, attach tenant-assignable models, manage shared keys, configure tenant budgets/alerts, and coordinate change requests with platform admins. |
| `user/guide.md` | Builders / end users | Request keys, pick models, send `/v1/*` requests, interpret headers, and stay within budgets and limits. |
| `personal/guide.md` | Individual users | Personal tenant setup, personal keys, and self-serve usage tracking. |
| `reference/requests/` | All personas | Endpoint-specific curl guides plus header descriptions organized per `/v1/*` route. |
| `reference/troubleshooting.md` | All personas | Error matrix, diagnostics checklist, and escalation paths for admins, tenant owners, and users. |
| `admin/requests/` | Platform administrators | API-first workflows for tenants, keys, models, budgets, usage exports, providers, and admin tokens. |

## How to use these docs

1. Start with your role guide (`admin/`, `tenant/`, or `user/`) to understand responsibilities and day-to-day workflows.
2. Jump to the reference files whenever you need endpoint syntax, curl payloads, or troubleshooting steps.
3. Pivot to `docs/admin/runtime/README.md` and `docs/architecture/*` for deeper configuration or provider-level context.
4. Keep `Code_Examples/curl` open while testing so you can copy/paste scripts or adapt them to automation.

## Maintenance notes

- Update both the relevant role guide and the `reference/*` files whenever endpoints, UI labels, or workflows change.
- Call out multi-provider routing, tenant isolation, budgets, and rate limits in every new doc so readers understand the guardrails.
- Add links back into this README when you create new persona or reference docs so future contributors can find them quickly.
