# UI Kit & Directory Provider Reference

This note captures the shared admin UI primitives that landed during the refactor so future contributors can see how they fit together without spinning up Storybook.

## DataTable (`src/ui/kit/DataTable.tsx`)

- Wraps the shadcn table with built-in loading + empty states.
- Accepts `columns` (header + cell renderers) and `getKey` for deterministic React keys.
- Supports `dense` rows for dialog tables and a custom `emptyState` node.

Usage example (from `UsersPage`):

```tsx
const columns: DataTableColumn<PersonalTenantRecord>[] = [
  { header: "User", cell: (record) => <UserCell record={record} /> },
  { header: "Status", cell: (record) => <span className="capitalize">{record.status}</span> },
  // ...
]

<DataTable
  data={filteredRecords}
  columns={columns}
  getKey={(record) => record.tenant_id}
  isLoading={personalTenantsQuery.isLoading}
  emptyState="No users match the current filters."
/>;
```

## ChartCard (`src/ui/kit/ChartCard.tsx`)

- Standard header + toolbar layout for charts/cards.
- Takes a `loadingHeight` prop for skeleton sizing.
- Keeps chart sections consistent across admin Dashboard/Usage pages.

Example:

```tsx
<ChartCard
  title="Top tenants"
  description="Daily usage for each tenant"
  toolbar={<MetricSelector metric={metric} onChange={setMetric} />}
  isLoading={breakdown.isLoading}
>
  <UsageComparisonChart ... />
  <TopUsageList ... />
</ChartCard>
```

## DirectoryProvider (`src/providers/DirectoryProvider.tsx`)

- Preloads tenants, users, and model catalogs via React Query, exposing a single `useDirectoryData()` hook.
- Keeps pages like Usage, Tenants, Keys from duplicating `listTenants/listUsers/listModelCatalog` calls.
- Wraps the admin portal shell so any nested component can grab the directory data or trigger a refetch.

```tsx
const { tenants, users, models, isLoading } = useDirectoryData();

// UsagePage now consumes the shared arrays instead of issuing its own queries.
```

## Testing

- Unit coverage for DataTable + ChartCard lives under `src/ui/kit/__tests__` (Vitest + Testing Library).
- E2E smoke tests (Playwright) hit the running frontend to ensure the login shell renders (`tests/ui-smoke.spec.ts`). Run `bun run dev` (or serve the built assets) then `bun run test:e2e` to execute the checks.

Keep this doc updated as new kit components or providers land so we always have a single reference outside the running app.
