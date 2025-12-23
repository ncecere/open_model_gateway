# Getting Started

This guide orients you to the admin and user portals, the API docs, and the core
workflow for managing models, tenants, and keys.

## Where to Log In

- Admin portal: `http://<host>:<port>/admin`
- User portal: `http://<host>:<port>/`
- API docs (OpenAPI + Scalar UI): `http://<host>:<port>/docs`

The admin portal is for operators. The user portal is for end users who manage
their personal tenant and tenant memberships.

## Authentication Basics

- Local auth uses email + password (if enabled).
- SSO uses the configured OIDC provider.
- Admins can issue admin API tokens for automation.

If you are unsure which auth mode is enabled, check **Admin -> Settings**.

## Portal Layout

Admin portal common areas:

- **Dashboard**: platform health and high-level usage.
- **Models**: model catalog entries and routing config.
- **Tenants**: tenant records, memberships, and policies.
- **Keys**: API keys (tenant and admin tokens).
- **Usage**: spend, tokens, and exports.
- **Settings**: budgets, rate limits, alerts, defaults.

User portal common areas:

- **Dashboard**: usage summary and tenant scope.
- **Usage**: daily breakdowns and charts.
- **API Keys**: personal and tenant-scoped keys.
- **Tenants**: membership visibility and scopes.
- **Batches / Files**: if enabled in your deployment.

## First-Time Setup Checklist

1. Add one or more models to the catalog.
2. Create a tenant for your first team.
3. Invite users and assign roles.
4. Create API keys (per app or per team).
5. Configure budgets and rate limits.
6. Run a test request through `/v1`.

## Quick Test Commands

List models:

```bash
curl http://localhost:8090/v1/models \
  -H "Authorization: Bearer $API_KEY"
```

Chat request:

```bash
curl http://localhost:8090/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hello"}]}'
```

## What to Read Next

- Adding models: `user_docs/models.md`
- Tenants and users: `user_docs/tenants-and-users.md`
- API keys and admin tokens: `user_docs/api-keys.md`
- Budgets and rate limits: `user_docs/budgets-and-rate-limits.md`
