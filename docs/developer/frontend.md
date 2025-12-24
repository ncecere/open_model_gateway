# Frontend Guide

The Bun-powered React/Vite app in `backend/frontend` delivers the admin and user portals backed by the Go APIs.

## Understand the stack
Use this stack summary before editing modules.

| Aspect | Details |
| --- | --- |
| Framework | React 18 + Vite + TypeScript, bundled and tested with Bun. |
| Styling | Tailwind CSS, shadcn/ui primitives, and Lucide icons composed inside `src/ui`. |
| State & data | TanStack Query plus Axios clients (`src/api/`) with interceptors for auth, tenant scoping, and toast feedback. |
| Routing | React Router shells for `/apps/admin` and `/apps/user` with guarded routes. |
| Testing | Vitest + Testing Library for units, Playwright for smoke flows under `frontend/tests`. |

## Structure the repo
Review these directories before wiring features.

| Path | Purpose |
| --- | --- |
| `src/apps/admin` | Admin portal shell, routes, layouts, and shared providers. |
| `src/apps/user` | User portal shell with its own auth provider, sidebar, and dashboards. |
| `src/features/*` | Domain-specific hooks/components (models, tenants, usage, api-keys, files, batches). |
| `src/api` | Axios instances, DTO mappers, React Query hooks, and tenant-aware helpers. |
| `src/ui/kit` | Shared components such as `DataTable`, `ChartCard`, and status badges (see [UI kit](./ui-kit.md)). |
| `src/providers/DirectoryProvider.tsx` | Preloads tenants/users/models for reuse across portals. |

```
backend/frontend/
├── public
├── src/
│   ├── apps/{admin,user}
│   ├── api
│   ├── features
│   ├── ui/kit
│   └── main.tsx
└── bun.lock
```

## Build feature modules
Follow the [feature module guide](./frontend-feature-modules.md) to co-locate hooks, dialogs, and tables per domain so admin and user pages stay lean and share UI state.

## Ship the admin portal
These surfaces rely on the shared modules and admin APIs.

| Area | Highlights |
| --- | --- |
| Auth & shell | Local + OIDC login backed by `/admin/auth/*`, sidebar navigation, profile dropdown, and admin token helpers. |
| Dashboard | Health cards pulling `/healthz`, stats from `/admin/usage/summary`, and charts from `/admin/usage/breakdown` with timezone selectors. |
| Model catalog | Filterable table plus dialog reusing provider-specific panels (Azure, Bedrock, Vertex, OpenAI-compatible) to edit pricing, metadata, and overrides. |
| Tenants | Directory + edit modal tabs handling status, budgets, rate limits, memberships, and model entitlements. |
| API keys | Tenant switcher, key issuance/revoke dialog, budget + quota summaries, and copy-safe secret reveal flows. |
| Settings | Budget defaults, rate-limit defaults, default models, alert channels, and email test utilities. |
| Usage & exports | Dedicated usage views, CSV export helpers, trend comparisons, and timezone-aware breakdown tables. |
| Providers | `/admin/providers` table showing adapter capabilities, incidents, and routing weights. |

## Serve the user portal
The `/user` app gives non-admin members scoped management tools.

| Area | Highlights |
| --- | --- |
| Auth & shell | Separate auth context, sidebar, and “Open Admin Portal” link for super-admins. |
| Dashboard | Scoped stats (personal vs tenant), recent activity, and key summaries backed by `/user/dashboard`. |
| Usage | Same comparison + custom range controls, now hitting `/user/usage/*` with scope selectors and entity caps. |
| API keys | Tabs split personal vs tenant keys with rotate/reveal flows and inherited budget metadata. |
| Tenants | Tabbed modal for memberships, budgets, limits, and model toggles; personal tenants stay hidden. |
| Profile | Modal to edit name or rotation of local-auth passwords via `/user/profile`. |

## Run and embed
Install deps with `bun install`, develop via `bun run dev`, run unit tests with `bun run test`, and kick Playwright smoke tests through `bun run test:e2e` against a running backend. Build assets with `bun run build`; `make run-backend` copies `dist/` to `backend/internal/httpserver/ui/dist` so Fiber serves the SPA.

## Track ongoing work
Triage frontend tasks in `agents.md`, focusing on richer error states, expanded Playwright coverage for auth/catalog/key flows, reusable presets for custom usage ranges, and visibility for quota reset metadata.
