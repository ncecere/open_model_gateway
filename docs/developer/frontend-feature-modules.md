# Frontend Feature Modules

This guide explains how the admin & user portals are now organized around domain-specific feature folders under `backend/frontend/src/features/`. Use it as the blueprint when extending existing surfaces or shipping new ones.

## Goals

- Keep page files lean (data orchestration + navigation only).
- Co-locate React Query hooks, dialogs, tables, and helper utilities per domain so admin and user portals stay in sync.
- Make it trivial to reuse UI/logic between portals (e.g., Tenants, Usage, API Keys).

## Directory Pattern

```
backend/frontend/src/features/<domain>/
  components/   # shadcn-based UI widgets (tables, dialogs, headers, sections)
  hooks/        # React Query + local-state helpers
  utils.ts      # Formatters/common helpers (optional)
  index.ts      # Barrel export for page imports
```

Existing domains:

- `features/models` – catalog filters/table/editor.
- `features/tenants` – summary header, directory card, create/edit dialogs, membership section & dialog, plus hooks (`useTenantDirectoryQuery`, `useTenantCreateDialog`, `useTenantEditDialog`, `useMembershipDialog`).
- `features/usage`, `features/api-keys`, `features/files`, `features/batches`.

## Adding or Extending a Feature

1. **Create/Update Hooks**
   - Put all React Query calls + derived state inside `hooks/`.
   - Example: `useTenantDirectoryFilters` returns search/status filters, active count, and sorted tenants; pages just consume the hook.

2. **Build Reusable Components**
   - Render UI in `components/` and accept props for data/mutations (e.g., `TenantEditDialog` takes the dialog state + callbacks for submit and model toggles).
   - Keep styling consistent with shadcn primitives already in use.

3. **Update the Page Shell**
   - Import the hook(s) + component(s) from `features/<domain>`.
   - The page should only:
     - Invoke hooks to fetch data/derive state.
     - Wire up mutation handlers (create/update/delete).
     - Compose the feature components.

4. **Document the Change**
   - Mention new components/hooks inside `docs/architecture/frontend.md` or the relevant feature doc.

## Example: Tenants

- `TenantSummaryHeader` shows counts + buttons.
- `TenantCreateDialog`, `TenantEditDialog`, `TenantMembershipSection`, and `TenantMembershipDialog` own all dialog/table UI.
- Hooks (`useTenantCreateDialog`, `useTenantEditDialog`, `useMembershipDialog`) encapsulate form state, defaulting logic, and suggestions.
- `TenantsPage.tsx` now imports these pieces, passes data/mutation callbacks, and stays under ~400 LOC.

When adding another tenant-specific view (e.g., audit history), extend `features/tenants` with a new component + hook and import it from the page.

## Testing & Future Enhancements

- Continue to run `bun run build` (or `bun run test` when we add component tests) after moving code.
- Prefer Storybook/Playwright coverage at the component level for shared dialogs/sections.
- To add a new feature (e.g., “Providers” admin screen):
  1. Create `features/providers/{hooks,components}/`.
  2. Build the table/dialog components there.
  3. Point `pages/ProvidersPage.tsx` and any user-facing equivalent to those exports.

Following this pattern keeps future work predictable and prevents drift between admin and user portals.
