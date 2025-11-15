import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { UserFileRecord } from "@/api/user/files";
import type { ReactNode } from "react";
import { dateFormatter, formatBytes } from "../utils";
import { formatFileStatus } from "./FileStatusBadge";

type UserFileDetailsDialogProps = {
  file: UserFileRecord | null;
  tenantLabel?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function UserFileDetailsDialog({ file, tenantLabel, open, onOpenChange }: UserFileDetailsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        {file ? (
          <>
            <DialogHeader>
              <DialogTitle>{file.filename}</DialogTitle>
              <DialogDescription>File ID {file.id}</DialogDescription>
            </DialogHeader>
            <div className="space-y-3 py-2 text-sm">
              <DetailItem label="Tenant">{tenantLabel ?? "—"}</DetailItem>
              <div className="grid gap-3 sm:grid-cols-2">
                <DetailItem label="Purpose" className="capitalize">
                  {file.purpose || "—"}
                </DetailItem>
                <DetailItem label="Size">{formatBytes(file.bytes)}</DetailItem>
                <DetailItem label="Content type">{file.content_type}</DetailItem>
                <DetailItem label="Status" className="capitalize">
                  {formatFileStatus(file.status)}
                </DetailItem>
                <DetailItem label="Status details">
                  {file.status_details ? (
                    <span className="text-sm text-muted-foreground">{file.status_details}</span>
                  ) : (
                    "—"
                  )}
                </DetailItem>
                <DetailItem label="Created">
                  {dateFormatter.format(new Date(file.created_at))}
                </DetailItem>
                <DetailItem label="Expires">
                  {file.expires_at
                    ? dateFormatter.format(new Date(file.expires_at))
                    : "—"}
                </DetailItem>
              </div>
            </div>
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
