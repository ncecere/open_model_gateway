# Frontend Status & Reference

This note tracks the current capabilities of the Open Model Gateway admin UI (`backend/frontend/`) and outlines how the React application integrates with the Go backend.

## Stack Overview

- **Framework**: React 19 + Vite, TypeScript, Bun (dev/build).
- **Design System**: Tailwind CSS + shadcn/ui primitives, Lucide icons.
- **State/Data**: React Query (TanStack Query v5) with dedicated API modules under `src/api/`.
- **Routing**: React Router 7 with authenticated routes gated behind the admin login flow.
- **Build**: `bun run build` (TS + Vite) emits static assets later embedded in `backend/internal/httpserver/ui/dist` via `make run-backend`.

### Directory Snapshot

```
backend/frontend/
├── features/                 # Domain-specific hooks/components (refactor target)
├── public/
├── src/
│   ├── apps/
│   │   ├── admin/             # Admin portal shell
│   │   └── user/              # Live user-facing portal (dashboard/usage/API keys/tenants)
│   ├── api/                   # Axios clients + DTO mappers
│   ├── components/            # Shared UI (AppLayout, forms, tables)
│   ├── hooks/                 # Auth/session helpers, toast wrapper
│   ├── pages/                 # Feature pages (Dashboard, Tenants, Keys, Models, Usage)
│   └── main.tsx               # App bootstrap (currently loads AdminApp)
└── components.json            # shadcn registry
```

`src/App.tsx` wires both `AdminApp` and `UserApp` entry points while the shared UI kit/hooks live in `src/ui` & `src/api`, so each portal can evolve independently without duplicating infrastructure.

### Feature Modules (`src/features/*`)

The refactor introduced a set of domain-specific “feature folders” so pages stay under ~500 LoC and reusable UI/logic lives in one place:

| Folder | Purpose / Key Components |
|--------|--------------------------|
| `features/models/` | `ModelFilters`, `ModelTable`, `ModelEditorDialog`, provider metadata registry, and helpers that translate SQLC models to form state. |
| `features/tenants/` | `TenantSummaryHeader`, `TenantDirectoryCard`, create/edit/membership dialogs, directory + dialog hooks. Both admin and user portals import these building blocks. |
| `features/usage/` | Shared comparison selectors, overview cards, tabs, and hooks (`useUsageSelections`, `useCustomRange`) that back both admin and user usage pages. |
| `features/api-keys/` | Reusable API key table, secret dialog, revoke dialog, and budget metadata helpers now consumed by `KeysPage` and `apps/user/pages/ApiKeysPage`. |
| `features/files/` & `features/batches/` | Filter cards, tables, detail dialogs, and hooks that keep the admin and user implementations in sync. |

Pages under `src/pages/` (admin) and `src/apps/user/pages/` (user) now render these components, pass in the required React Query data/mutation handlers, and let the feature folders own layout + local state. When adding a new surface, prefer this pattern:

1. Create a hook inside `features/<domain>/hooks/` that encapsulates the React Query call and any derived state (filters/search, dialog form state, etc.).
2. Build the UI in `features/<domain>/components/` so both admin and user portals can consume it.
3. Keep the page file responsible only for orchestrating hooks/mutations and navigation.

See `docs/developer/frontend-feature-modules.md` for a step-by-step guide and best practices for extending these modules or adding new ones.

## Implemented Features

| Area               | Highlights |
|--------------------|------------|
| **Auth**           | Login form that hits `/admin/auth/login`; stores tokens via React Query + Axios interceptors; automatic refresh + logout handling. |
| **App Shell**      | Sidebar navigation (Dashboard, Tenants, API Keys, Models, Usage), top bar showing logged-in email + account menu. |
| **Model Catalog**  | Full CRUD dialog powered by shadcn components: provider select, decimal pricing inputs, modality checkboxes (text/image/audio/video), toggle for enabled/tool-calling, metadata textarea, **plus provider-specific panels** (Azure, Bedrock, Vertex, OpenAI-compatible). Vertex editor now supports raw JSON upload/paste for service accounts and writes structured `provider_overrides`. List view mirrors backend data and offers edit/delete actions with confirmation dialogs. Disabled models are clearly labelled. |
| **Tenants**        | Table view with status filters, create/edit dialogs (status + budget + alert channels + allowed models), membership management panel, and spend progress indicators. |
| **API Keys**       | Tenant switcher, key issuance dialog (quota + budget overrides), revoke flow with confirmation, copy-to-clipboard helpers for prefix/secret, live budget + throttle summary. |
| **Budget Controls**| Settings tab exposes editable budget defaults (PUT `/admin/budgets/default`), shows inherited refresh/alert policies, and links back to tenant-specific overrides driven by `/admin/tenants/:id/budget`. |
| **Dashboard**      | Gateway health card hits `/healthz` for app/Postgres/Redis status, stat tiles highlight requests/tokens/spend, and a shadcn/Recharts area chart lets operators pick tenants vs models, metrics, and time windows using `/admin/usage/breakdown` (dates now rendered in the backend-configured timezone). |
| **Usage**          | Pure usage surface with gateway totals, tenant drill-down cards, CSV export helpers, and daily bucket table wired to `/admin/usage/summary`. The comparison card now includes tenant + model dropdowns, entity cap enforcement, and a “Custom range” picker that issues `start`/`end` requests against `/admin/usage/compare`. |
| **Users**          | Admin Users tab (sidebar) lists personal tenants per account, showing budget usage and linking back to invitations/password creation flows. |

### User Portal Status

| Area               | Highlights |
|--------------------|------------|
| **Shell/Auth**     | `/user` app has its own layout, sidebar (Dashboard, Usage, Tenants, API Keys), and protected routing via `useUserAuth`. Super-admins get an “Open Admin Portal” button in the sidebar plus return link. |
| **Dashboard**      | Stat tiles + cards driven by `/user/dashboard` with scope selector (Personal vs tenant keys) plus recent usage/API key tables and tenant activity list (personal tenants filtered out). |
| **Usage**          | Shares the same scope + period controls, shows per-scope API-keys table and daily series (tenant breakdown removed because the scope selector covers that use case). Comparison overlays now match admin (tenant + model selectors, 6-entity cap, and optional custom date ranges before hitting `/user/usage/compare`). |
| **API Keys**       | Tabs split personal vs tenant keys with issuance/revoke flows; tenant create/revoke actions are gated by membership role. Issued secrets show in a dedicated “Save this secret” card. |
| **Tenants**        | Read-only membership list (personal tenants hidden) plus budget summary card fed by `/user/tenants/:id/summary`. |
| **Profile**        | Accessible from the user dropdown; modal shows account metadata and allows name edits. Local-auth accounts can change passwords (old/new/confirm); OIDC accounts see read-only messaging. |

All forms surface toast feedback on success/failure and leverage React Query invalidations to keep data fresh.

## Build & Embed Workflow

1. Run `bun install` after dependency changes (e.g., new shadcn components).
2. `bun run dev` for hot reloading during UI work.
3. `bun run build` generates `dist/`; `make run-backend` automatically copies the bundle into `backend/internal/httpserver/ui/dist/` before launching the Go binary.
4. Production assets are served by Fiber via `/` with SPA fallback.

## Configuration Touchpoints

- API base URL defaults to `/admin`; Axios client handles cookie-supplied refresh tokens and attaches bearer tokens when available.
- Tenant-scoped requests support `X-Tenant-ID` header via `setTenantId()` helper (Keys, Memberships pages).
- Admin users seeded via backend bootstrap appear in the login dropdown if local auth is enabled.

## Outstanding Frontend Tasks

- Expand error/empty states with dedicated components and copy.
- Add Playwright end-to-end coverage for auth, catalog edits, and API key issuance.
- Add saved custom-range presets + shareable URLs for the comparison tabs so operators can bookmark billing windows.
- Surface API key quota reset windows and inherited refresh info alongside each key.
- Show “last updated” metadata for budget defaults in the Settings tab (and history once backend exposes it).

Refer to `FE_BE_BUILD.md` for the combined build/packaging checklist and to `agents.md` for day-to-day progress updates.
