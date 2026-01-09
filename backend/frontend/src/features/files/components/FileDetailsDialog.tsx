import { DetailsDialog, DetailItem } from "@/components/dialogs";
import type { AdminFileRecord } from "@/api/files";
import { dateFormatter, formatBytes } from "../utils";
import { FileStatusBadge } from "./FileStatusBadge";

type FileDetailsDialogProps = {
  file: AdminFileRecord | null;
  isPersonal: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};


export function FileDetailsDialog({ file, isPersonal, open, onOpenChange }: FileDetailsDialogProps) {
  if (!file) {
    return null;
  }

  return (
    <DetailsDialog
      open={open}
      onOpenChange={onOpenChange}
      title={file.filename}
      description={`File ID ${file.id}`}
      maxWidth="sm:max-w-[720px]"
    >
      <div className="space-y-4 text-sm">
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
            <FileStatusBadge status={file.status} />
          </DetailItem>
          <DetailItem label="Status details">
            {file.status_details ? (
              <span className="text-sm text-muted-foreground">{file.status_details}</span>
            ) : (
              "—"
            )}
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
    </DetailsDialog>
  );
}

// Re-export DetailItem for backwards compatibility
export { DetailItem } from "@/components/dialogs";
