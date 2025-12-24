# Admin request reference
Automate `/admin/*` workflows with these curl-ready snippets.

## Prepare environment
Set the admin router base, login credentials, and reuse `GATEWAY_BASE_URL` when stripping `/v1`.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export GATEWAY_ADMIN_BASE_URL="http://localhost:8090"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="change-me"
```

## Authenticate first
Most files include a `TOKEN=$(...)` snippet using `POST /admin/auth/login`; capture the bearer token before chaining additional requests.

## Pick an operation
| Doc | Focus |
| --- | --- |
| [tenants.md](tenants.md) | Create, list, and update tenants plus metadata. |
| [api_keys.md](api_keys.md) | Issue, rotate, and revoke API keys with budgets and rate limits. |
| [models.md](models.md) | Attach or detach catalog models for a tenant. |
| [budgets.md](budgets.md) | Manage tenant budgets, key overrides, and alert settings. |
| [usage_exports.md](usage_exports.md) | Trigger exports and manage billing webhooks. |
| [providers.md](providers.md) | Query provider health incidents and routing weights. |
| [tokens.md](tokens.md) | Obtain admin API tokens for long-lived automation. |

## Helpful extras
`Code_Examples/curl/admin.sh` demonstrates a login → tenant → key flow that you can adapt for CI.
