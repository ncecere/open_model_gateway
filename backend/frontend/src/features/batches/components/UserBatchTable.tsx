import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { AlertTriangle, Download, Eye, MoreHorizontal, Trash2 } from "lucide-react";
import type { UserBatchRecord } from "@/api/user/batches";
import { dateFormatter, statusVariants, formatFinishedTimestamp } from "../utils";

export type UserBatchTableProps = {
  batches: UserBatchRecord[];
  isLoading: boolean;
  tenantName?: string;
  canManage: boolean;
  downloadingKey: string | null;
  hasMore: boolean;
  canPageBackward: boolean;
  pageSize: number;
  onView: (batch: UserBatchRecord) => void;
  onDownload: (batch: UserBatchRecord, kind: "output" | "errors") => void;
  onCancel?: (batch: UserBatchRecord) => void;
  disableCancel?: boolean;
  onNextPage: () => void;
  onPrevPage: () => void;
};

export function UserBatchTable({
  batches,
  isLoading,
  tenantName,
  canManage,
  downloadingKey,
  hasMore,
  canPageBackward,
  pageSize,
  onView,
  onDownload,
  onCancel,
  disableCancel,
  onNextPage,
  onPrevPage,
}: UserBatchTableProps) {
  if (isLoading) {
    return <SkeletonTable />;
  }

  if (!batches.length) {
    return <EmptyState tenantName={tenantName} />;
  }

  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Status</TableHead>
            <TableHead>Endpoint</TableHead>
            <TableHead>Created</TableHead>
            <TableHead>Finished</TableHead>
            <TableHead>Progress</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {batches.map((batch) => {
            const outputDisabled =
              !batch.output_file_id || downloadingKey === `${batch.id}-output`;
            const errorsDisabled =
              !batch.error_file_id || downloadingKey === `${batch.id}-errors`;
            const cancelDisabled =
              disableCancel || !canCancel(batch.status) || !canManage;

            return (
              <TableRow key={batch.id}>
                <TableCell>
                  <Badge variant={statusVariants[batch.status] ?? "outline"}>
                    {batch.status.replace(/_/g, " ")}
                  </Badge>
                  {batch.errors?.data?.length ? (
                    <p className="mt-1 flex items-center gap-1 text-xs text-destructive">
                      <AlertTriangle className="h-3 w-3" />
                      {batch.errors.data.length} issue
                      {batch.errors.data.length > 1 ? "s" : ""}
                    </p>
                  ) : null}
                </TableCell>
                <TableCell>
                  <div className="text-sm font-medium">{batch.endpoint}</div>
                  <p className="text-xs text-muted-foreground">
                    Window {batch.completion_window || "24h"}
                  </p>
                </TableCell>
                <TableCell className="text-sm">
                  {dateFormatter.format(new Date(batch.created_at))}
                  {batch.cancelling_at ? (
                    <p className="text-xs text-muted-foreground">
                      Cancelling{" "}
                      {dateFormatter.format(new Date(batch.cancelling_at))}
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
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuLabel>Actions</DropdownMenuLabel>
                      <DropdownMenuItem onClick={() => onView(batch)}>
                        <Eye className="mr-2 h-4 w-4" />
                        View details
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        disabled={outputDisabled}
                        onClick={() => onDownload(batch, "output")}
                      >
                        <Download className="mr-2 h-4 w-4" />
                        Download output
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        disabled={errorsDisabled}
                        onClick={() => onDownload(batch, "errors")}
                      >
                        <Download className="mr-2 h-4 w-4" />
                        Download errors
                      </DropdownMenuItem>
                      {canManage && onCancel ? (
                        <>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            disabled={cancelDisabled}
                            className="text-destructive focus:text-destructive"
                            onClick={() => onCancel(batch)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Cancel batch
                          </DropdownMenuItem>
                        </>
                      ) : null}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
      <div className="mt-4 flex flex-col gap-2 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
        <p>
          Showing {batches.length} result{batches.length === 1 ? "" : "s"} (max{" "}
          {pageSize} per page)
        </p>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={!canPageBackward}
            onClick={onPrevPage}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={!hasMore}
            onClick={onNextPage}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
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

function EmptyState({ tenantName }: { tenantName?: string }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">
      <AlertTriangle className="h-8 w-8 text-muted-foreground" />
      <div>
        No batches queued
        {tenantName ? ` for ${tenantName}` : ""}. Submit requests via an API key
        belonging to this tenant.
      </div>
    </div>
  );
}

function canCancel(status: string) {
  return !["completed", "failed", "cancelled"].includes(status);
}
