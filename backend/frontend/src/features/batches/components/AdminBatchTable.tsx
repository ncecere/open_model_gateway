import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  ActionsMenu,
  EmptyState,
  FilterBar,
  TableSkeleton,
  type FilterOption,
} from "@/components/tables";
import { AlertTriangle, Download, Eye, RouteIcon, Trash2 } from "lucide-react";
import type { BatchRecord } from "@/api/batches";
import {
  formatFinishedTimestamp,
  statusVariants,
  dateFormatter,
} from "../utils";

export type AdminBatchTableProps = {
  tenants: { id: string; name: string }[];
  personalTenantIds: Set<string>;
  batches: BatchRecord[];
  pageSize: number;
  hasMore: boolean;
  canPageBackward: boolean;
  isLoading: boolean;
  filters: {
    tenant: string;
    status: string;
    search: string;
  };
  downloadingKey?: string | null;
  onFiltersChange: (next: Partial<AdminBatchTableProps["filters"]>) => void;
  onSearchChange: (value: string) => void;
  onPaginate: (direction: "next" | "prev") => void;
  onView: (batch: BatchRecord, tenantLabel: string) => void;
  onDownload?: (batch: BatchRecord, kind: "output" | "errors") => void;
  onCancel: (batch: BatchRecord) => void;
};

export function AdminBatchTable({
  tenants,
  personalTenantIds,
  batches,
  pageSize,
  hasMore,
  canPageBackward,
  isLoading,
  filters,
  downloadingKey,
  onFiltersChange,
  onSearchChange,
  onPaginate,
  onView,
  onDownload,
  onCancel,
}: AdminBatchTableProps) {
  const rows = batches.map((batch) => {
    const isPersonal = personalTenantIds.has(batch.tenant_id);
    const tenantName =
      tenants.find((tenant) => tenant.id === batch.tenant_id)?.name ??
      batch.tenant_name ??
      "—";
    return {
      batch,
      tenantLabel: isPersonal ? "Personal" : tenantName,
    };
  });

  const filtersActive =
    filters.tenant !== "all" ||
    filters.status !== "all" ||
    Boolean(filters.search.trim());
  const showErrorPill = (batch: BatchRecord) =>
    (batch.errors?.data?.length ?? 0) > 0;

  const tenantOptions: FilterOption[] = [
    { value: "all", label: "All tenants" },
    ...tenants
      .filter((tenant) => !personalTenantIds.has(tenant.id))
      .map((tenant) => ({ value: tenant.id, label: tenant.name })),
  ];

  const statusOptions: FilterOption[] = [
    { value: "all", label: "All statuses" },
    ...Object.keys(statusVariants).map((status) => ({
      value: status,
      label: status.replace(/_/g, " "),
    })),
  ];

  return (
    <Card>
      <CardHeader className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <CardTitle>Batch jobs</CardTitle>
          <p className="text-xs text-muted-foreground">
            Inspect async workloads, filter by tenant/status, and manage stuck batches.
          </p>
        </div>
        <FilterBar
          filters={[
            {
              type: "search",
              key: "search",
              value: filters.search,
              onChange: onSearchChange,
              placeholder: "Search batch id, tenant, or endpoint",
              className: "sm:w-64",
            },
            {
              type: "select",
              key: "tenant",
              value: filters.tenant,
              onChange: (value) => onFiltersChange({ tenant: value }),
              options: tenantOptions,
              placeholder: "All tenants",
              className: "sm:w-48",
            },
            {
              type: "select",
              key: "status",
              value: filters.status,
              onChange: (value) => onFiltersChange({ status: value }),
              options: statusOptions,
              placeholder: "All statuses",
              className: "sm:w-40",
            },
          ]}
        />
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <TableSkeleton rows={4} />
        ) : rows.length === 0 ? (
          <EmptyState
            message={filtersActive ? "No batches match the current filters" : "No batches submitted yet"}
            description={
              filtersActive
                ? "Try adjusting your filter criteria."
                : "Upload a JSONL file via the /v1/batches API to populate this feed."
            }
          />
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Batch ID</TableHead>
                  <TableHead>Tenant</TableHead>
                  <TableHead>Endpoint</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Finished</TableHead>
                  <TableHead>Progress</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map(({ batch, tenantLabel }) => (
                  <TableRow key={batch.id}>
                    <TableCell>
                      <Badge
                        variant={statusVariants[batch.status] ?? "outline"}
                        className="capitalize"
                      >
                        {batch.status.replace(/_/g, " ")}
                      </Badge>
                      {showErrorPill(batch) ? (
                        <p className="mt-1 flex items-center gap-1 text-xs text-destructive">
                          <AlertTriangle className="h-3 w-3" />
                          {batch.errors?.data.length} validation issue
                          {batch.errors && batch.errors.data.length > 1 ? "s" : ""}
                        </p>
                      ) : null}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {batch.id}
                    </TableCell>
                    <TableCell className="text-sm font-medium">
                      {tenantLabel}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2 text-sm font-medium">
                        <RouteIcon className="h-4 w-4 text-muted-foreground" />
                        {batch.endpoint}
                      </div>
                      <p className="text-xs text-muted-foreground">
                        Window: {batch.completion_window || "24h"}
                      </p>
                    </TableCell>
                    <TableCell className="text-sm">
                      {dateFormatter.format(new Date(batch.created_at))}
                      {batch.in_progress_at ? (
                        <p className="text-xs text-muted-foreground">
                          Started{" "}
                          {dateFormatter.format(new Date(batch.in_progress_at))}
                        </p>
                      ) : null}
                      {batch.cancelling_at ? (
                        <p className="text-xs text-muted-foreground">
                          Cancelling{" "}
                          {dateFormatter.format(new Date(batch.cancelling_at))}
                        </p>
                      ) : null}
                      {batch.expired_at ? (
                        <p className="text-xs text-muted-foreground">
                          Expired {dateFormatter.format(new Date(batch.expired_at))}
                        </p>
                      ) : null}
                    </TableCell>
                    <TableCell className="text-sm">
                      {formatFinishedTimestamp(batch)}
                    </TableCell>
                    <TableCell className="text-sm">
                      <div className="font-medium">
                        {batch.counts.completed}/{batch.counts.total} completed
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {batch.counts.failed} failed · {batch.counts.cancelled} cancelled
                      </p>
                    </TableCell>
                    <TableCell className="text-right">
                      <ActionsMenu
                        label="Actions"
                        actions={[
                          {
                            label: "View details",
                            icon: <Eye className="h-4 w-4" />,
                            onClick: () => onView(batch, tenantLabel),
                          },
                          ...(onDownload
                            ? [
                                {
                                  label: "Download output",
                                  icon: <Download className="h-4 w-4" />,
                                  onClick: () => onDownload(batch, "output"),
                                  disabled:
                                    !batch.output_file_id ||
                                    downloadingKey === `${batch.id}-output`,
                                },
                                {
                                  label: "Download errors",
                                  icon: <Download className="h-4 w-4" />,
                                  onClick: () => onDownload(batch, "errors"),
                                  disabled:
                                    !batch.error_file_id ||
                                    downloadingKey === `${batch.id}-errors`,
                                },
                              ]
                            : []),
                          {
                            label: "Cancel batch",
                            icon: <Trash2 className="h-4 w-4" />,
                            onClick: () => onCancel(batch),
                            disabled: !canCancel(batch.status),
                            destructive: true,
                          },
                        ]}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
      <CardFooter className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-muted-foreground">
          Showing {batches.length} result{batches.length === 1 ? "" : "s"} (max{" "}
          {pageSize} per page)
        </p>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={!canPageBackward}
            onClick={() => onPaginate("prev")}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={!hasMore}
            onClick={() => onPaginate("next")}
          >
            Next
          </Button>
        </div>
      </CardFooter>
    </Card>
  );
}

function canCancel(status: string) {
  return !["completed", "failed", "cancelled"].includes(status);
}
