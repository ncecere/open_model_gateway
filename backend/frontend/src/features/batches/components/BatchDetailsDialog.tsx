import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { BatchRecord } from "@/api/batches";
import type { UserBatchRecord } from "@/api/user/batches";
import {
  dateFormatter,
  formatFinishedTimestamp,
  statusVariants,
} from "../utils";
import type { ReactNode } from "react";

type SharedBatchRecord = (BatchRecord | UserBatchRecord) & {
  tenant_name?: string;
  tenant_id: string;
  api_key_id?: string;
};

export type BatchDetailsDialogProps<T extends SharedBatchRecord = SharedBatchRecord> = {
  batch: T | null;
  tenantLabel?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function BatchDetailsDialog<T extends SharedBatchRecord>({
  batch,
  tenantLabel,
  open,
  onOpenChange,
}: BatchDetailsDialogProps<T>) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[640px]">
        {batch ? (
          <>
            <DialogHeader>
              <DialogTitle className="text-lg font-semibold">
                Batch {batch.id}
              </DialogTitle>
              <DialogDescription>
                Tenant:{" "}
                <span className="font-medium text-foreground">
                  {tenantLabel ?? batch.tenant_name ?? "—"}
                </span>
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-2 text-sm">
              <div className="grid gap-4 sm:grid-cols-2">
                <DetailItem label="Status">
                  <Badge
                    variant={statusVariants[batch.status] ?? "outline"}
                    className="capitalize"
                  >
                    {batch.status.replace(/_/g, " ")}
                  </Badge>
                </DetailItem>
                <DetailItem label="Endpoint">{batch.endpoint}</DetailItem>
                <DetailItem label="Completion window">
                  {batch.completion_window || "—"}
                </DetailItem>
                <DetailItem label="Max concurrency">
                  {batch.max_concurrency}
                </DetailItem>
                {batch.api_key_id ? (
                  <DetailItem label="API key ID">
                    <span className="font-mono text-xs text-muted-foreground">
                      {batch.api_key_id}
                    </span>
                  </DetailItem>
                ) : null}
                {batch.tenant_id ? (
                  <DetailItem label="Tenant ID">
                    <span className="font-mono text-xs text-muted-foreground">
                      {batch.tenant_id}
                    </span>
                  </DetailItem>
                ) : null}
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <DetailItem label="Created">
                  {dateFormatter.format(new Date(batch.created_at))}
                </DetailItem>
                <DetailItem label="Started">
                  {batch.in_progress_at
                    ? dateFormatter.format(new Date(batch.in_progress_at))
                    : "—"}
                </DetailItem>
                <DetailItem label="Finished">
                  {formatFinishedTimestamp(batch)}
                </DetailItem>
                <DetailItem label="Expires">
                  {batch.expires_at
                    ? dateFormatter.format(new Date(batch.expires_at))
                    : "—"}
                </DetailItem>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <DetailItem label="Requests">
                  {batch.counts.total.toLocaleString()}
                </DetailItem>
                <DetailItem label="Completed / Failed / Cancelled">
                  {batch.counts.completed} / {batch.counts.failed} /{" "}
                  {batch.counts.cancelled}
                </DetailItem>
              </div>
              <div>
                <p className="text-xs font-medium uppercase text-muted-foreground">
                  Metadata
                </p>
                {batch.metadata && Object.keys(batch.metadata).length > 0 ? (
                  <ul className="mt-2 space-y-1 text-sm">
                    {Object.entries(batch.metadata).map(([key, value]) => (
                      <li key={key} className="flex justify-between gap-4">
                        <span className="text-muted-foreground">{key}</span>
                        <span className="font-medium">{value}</span>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    No metadata provided.
                  </p>
                )}
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Close
              </Button>
            </DialogFooter>
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

type DetailItemProps = {
  label: string;
  className?: string;
  children: ReactNode;
};

function DetailItem({ label, className, children }: DetailItemProps) {
  return (
    <div className={className ?? ""}>
      <p className="text-xs uppercase text-muted-foreground">{label}</p>
      <div className="text-sm font-medium text-foreground">{children}</div>
    </div>
  );
}
