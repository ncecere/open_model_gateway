import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertTriangle,
  Eye,
  MoreHorizontal,
  RouteIcon,
  Search,
  Trash2,
} from "lucide-react";
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
  onFiltersChange: (next: Partial<AdminBatchTableProps["filters"]>) => void;
  onSearchChange: (value: string) => void;
  onPaginate: (direction: "next" | "prev") => void;
  onView: (batch: BatchRecord, tenantLabel: string) => void;
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
  onFiltersChange,
  onSearchChange,
  onPaginate,
  onView,
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

  return (
    <Card>
      <CardHeader className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <CardTitle>Batch jobs</CardTitle>
          <p className="text-xs text-muted-foreground">
            Inspect async workloads, filter by tenant/status, and manage stuck batches.
          </p>
        </div>
        <div className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
          <div className="relative sm:w-64">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={filters.search}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder="Search batch id, tenant, or endpoint"
              className="pl-9"
            />
          </div>
          <Select
            value={filters.tenant}
            onValueChange={(value) => onFiltersChange({ tenant: value })}
          >
            <SelectTrigger className="sm:w-48">
              <SelectValue placeholder="All tenants" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All tenants</SelectItem>
              {tenants
                .filter((tenant) => !personalTenantIds.has(tenant.id))
                .map((tenant) => (
                  <SelectItem key={tenant.id} value={tenant.id}>
                    {tenant.name}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
          <Select
            value={filters.status}
            onValueChange={(value) => onFiltersChange({ status: value })}
          >
            <SelectTrigger className="sm:w-40">
              <SelectValue placeholder="All statuses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              {Object.keys(statusVariants).map((status) => (
                <SelectItem key={status} value={status}>
                  {status.replace(/_/g, " ")}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <SkeletonTable />
        ) : rows.length === 0 ? (
          <EmptyState filtersActive={filtersActive} />
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
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                            <span className="sr-only">Open actions</span>
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuLabel>Actions</DropdownMenuLabel>
                          <DropdownMenuItem
                            onClick={() => onView(batch, tenantLabel)}
                          >
                            <Eye className="mr-2 h-4 w-4" />
                            View details
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            disabled={!canCancel(batch.status)}
                            className="text-destructive focus:text-destructive"
                            onClick={() => onCancel(batch)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Cancel batch
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
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

function SkeletonTable() {
  return (
    <div className="space-y-2">
      {[...Array(4)].map((_, idx) => (
        <Skeleton key={idx} className="h-12 w-full" />
      ))}
    </div>
  );
}

function EmptyState({ filtersActive }: { filtersActive: boolean }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">
      <AlertTriangle className="h-8 w-8 text-muted-foreground" />
      <div>
        {filtersActive
          ? "No batches match the current filters."
          : "No batches have been submitted yet. Upload a JSONL file via the /v1/batches API to populate this feed."}
      </div>
    </div>
  );
}

function canCancel(status: string) {
  return !["completed", "failed", "cancelled"].includes(status);
}
