import { useState } from "react";
import { Copy, Check, Download } from "lucide-react";
import { DetailsDialog, DetailItem } from "@/components/dialogs";
import { Button } from "@/components/ui/button";
import type { AdminFileRecord } from "@/api/files";
import { downloadAdminFileContent } from "@/api/files";
import { dateFormatter, formatBytes } from "../utils";
import { FileStatusBadge } from "./FileStatusBadge";
import { FileTypeIcon } from "./FileTypeIcon";

type FileDetailsDialogProps = {
  file: AdminFileRecord | null;
  isPersonal: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function FileDetailsDialog({ file, isPersonal, open, onOpenChange }: FileDetailsDialogProps) {
  const [copied, setCopied] = useState(false);

  if (!file) {
    return null;
  }

  const handleCopyId = async () => {
    await navigator.clipboard.writeText(file.id);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = () => {
    downloadAdminFileContent(file.id);
  };

  return (
    <DetailsDialog
      open={open}
      onOpenChange={onOpenChange}
      title={
        <div className="flex items-center gap-2">
          <FileTypeIcon filename={file.filename} contentType={file.content_type} className="h-5 w-5" />
          <span>{file.filename}</span>
        </div>
      }
      description={
        <div className="flex items-center gap-2">
          <span className="font-mono text-xs">{file.id}</span>
          <button
            onClick={handleCopyId}
            className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
            title="Copy file ID"
          >
            {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
          </button>
        </div>
      }
      maxWidth="sm:max-w-[720px]"
      extraFooter={
        <Button variant="secondary" onClick={handleDownload}>
          <Download className="mr-2 h-4 w-4" />
          Download
        </Button>
      }
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
            <span className="break-all font-mono text-xs">{file.checksum || "—"}</span>
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
