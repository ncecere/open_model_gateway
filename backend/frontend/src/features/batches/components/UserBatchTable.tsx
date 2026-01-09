import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ActionsMenu, EmptyState, TableSkeleton, type ActionItem } from "@/components/tables";
import { AlertTriangle, Download, Eye, Trash2 } from "lucide-react";
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
    return <TableSkeleton rows={4} />;
  }

  if (!batches.length) {
    return (
      <EmptyState
        message={`No batches queued${tenantName ? ` for ${tenantName}` : ""}`}
        description="Submit requests via an API key belonging to this tenant."
      />
    );
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
                  <ActionsMenu
                    label="Actions"
                    actions={[
                      {
                        label: "View details",
                        icon: <Eye className="h-4 w-4" />,
                        onClick: () => onView(batch),
                      },
                      {
                        label: "Download output",
                        icon: <Download className="h-4 w-4" />,
                        onClick: () => onDownload(batch, "output"),
                        disabled: outputDisabled,
                      },
                      {
                        label: "Download errors",
                        icon: <Download className="h-4 w-4" />,
                        onClick: () => onDownload(batch, "errors"),
                        disabled: errorsDisabled,
                      },
                      ...(canManage && onCancel
                        ? [
                            {
                              label: "Cancel batch",
                              icon: <Trash2 className="h-4 w-4" />,
                              onClick: () => onCancel(batch),
                              disabled: cancelDisabled,
                              destructive: true,
                            } as ActionItem,
                          ]
                        : []),
                    ]}
                  />
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

function canCancel(status: string) {
  return !["completed", "failed", "cancelled"].includes(status);
}
