import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { AdminFileRecord } from "@/api/files";
import { Badge } from "@/components/ui/badge";
import { dateFormatter, formatBytes } from "../utils";
import type { ReactNode } from "react";

type FileRecord = Pick<
  AdminFileRecord,
  | "id"
  | "tenant_id"
  | "tenant_name"
  | "filename"
  | "purpose"
  | "content_type"
  | "bytes"
  | "checksum"
  | "encrypted"
  | "storage_backend"
  | "deleted_at"
  | "created_at"
  | "expires_at"
> & { [key: string]: any };

type FileDetailsDialogProps = {
  file: FileRecord | null;
  isPersonal: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function FileDetailsDialog({ file, isPersonal, open, onOpenChange }: FileDetailsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[720px]">
        {file ? (
          <>
            <DialogHeader>
              <DialogTitle>{file.filename}</DialogTitle>
              <DialogDescription>File ID {file.id}</DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-2 text-sm">
              <div className="grid gap-4 sm:grid-cols-2">
                <DetailItem label="Tenant">
                  {isPersonal ? "Personal" : file.tenant_name || "—"}
                </DetailItem>
                <DetailItem label="Tenant ID">
                  <span className="font-mono text-xs text-muted-foreground">
                    {file.tenant_id}
                  </span>
                </DetailItem>
                <DetailItem label="Purpose" className="capitalize">
                  {file.purpose || "—"}
                </DetailItem>
                <DetailItem label="Content type">{file.content_type}</DetailItem>
                <DetailItem label="Size">{formatBytes(file.bytes)}</DetailItem>
                <DetailItem label="Checksum">
                  <span className="break-all">{file.checksum || "—"}</span>
                </DetailItem>
                <DetailItem label="Encrypted">{file.encrypted ? "Yes" : "No"}</DetailItem>
                <DetailItem label="Storage backend">{file.storage_backend}</DetailItem>
                <DetailItem label="Status" className="capitalize">
                  <Badge variant={file.deleted_at ? "destructive" : "secondary"}>
                    {file.deleted_at ? "deleted" : "active"}
                  </Badge>
                </DetailItem>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <DetailItem label="Created">
                  {dateFormatter.format(new Date(file.created_at))}
                </DetailItem>
                <DetailItem label="Expires">
                  {file.expires_at
                    ? dateFormatter.format(new Date(file.expires_at))
                    : "—"}
                </DetailItem>
                <DetailItem label="Deleted">
                  {file.deleted_at
                    ? dateFormatter.format(new Date(file.deleted_at))
                    : "—"}
                </DetailItem>
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

export function DetailItem({ label, className, children }: DetailItemProps) {
  return (
    <div className={className ?? ""}>
      <p className="text-xs uppercase text-muted-foreground">{label}</p>
      <div className="text-sm font-medium text-foreground">{children}</div>
    </div>
  );
}
