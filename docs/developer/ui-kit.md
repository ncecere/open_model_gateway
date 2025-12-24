# UI Kit Guide

Shared UI primitives live under `backend/frontend/src/ui/kit` so admin and user portals stay consistent without Storybook overhead.

## Catalog shared primitives
Use these components before building new ones.

| Component | Path | Purpose |
| --- | --- | --- |
| `DataTable` | `src/ui/kit/DataTable.tsx` | shadcn-based table with loading + empty states and dense mode. |
| `ChartCard` | `src/ui/kit/ChartCard.tsx` | Card shell for charts with title, description, toolbar, and skeleton height. |
| `StatusBadge` + helpers | `src/ui/kit/StatusBadge.tsx` | Consistent status/budget badges reused in admin + user portals. |
| `DirectoryProvider` | `src/providers/DirectoryProvider.tsx` | React context exposing tenants, users, and models for downstream hooks. |

## Render data tables
`DataTable` expects column definitions and a deterministic key function.

```tsx
const columns: DataTableColumn<ApiKey>[] = [
  { header: "Key", cell: (record) => <span>{record.label}</span> },
  { header: "Status", cell: (record) => <StatusBadge status={record.status} /> },
];

<DataTable
  data={apiKeys}
  columns={columns}
  getKey={(record) => record.id}
  isLoading={query.isLoading}
  emptyState="No API keys found."
/>
```

## Compose chart cards
Wrap usage charts or stats in `ChartCard` to keep layouts consistent across dashboard and usage pages.

```tsx
<ChartCard
  title="Top tenants"
  description="Last 7 days"
  toolbar={<MetricSelector metric={metric} onChange={setMetric} />}
  isLoading={breakdown.isLoading}
>
  <UsageComparisonChart data={breakdown.data} />
</ChartCard>
```

## Share directory data
Wrap the admin app in `DirectoryProvider` and consume the hook anywhere beneath it to avoid duplicate React Query calls.

```tsx
const { tenants, users, models, isLoading, refetch } = useDirectoryData();
```

Use the shared data to populate selectors (tenants in usage comparisons, model lists in dialogs) and call `refetch` whenever mutations alter directories.

## Test the kit
Vitest suites live beside the components (`src/ui/kit/__tests__`), and Playwright smoke tests under `frontend/tests` ensure the login shell renders with kit components present. Run `bun run test` for units and `bun run test:e2e` after starting the backend to cover end-to-end flows.
