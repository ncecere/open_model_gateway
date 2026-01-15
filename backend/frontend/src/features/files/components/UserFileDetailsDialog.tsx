import { useState } from "react";
import { Copy, Check, Download } from "lucide-react";
import { DetailsDialog, DetailItem } from "@/components/dialogs";
import { Button } from "@/components/ui/button";
import type { UserFileRecord } from "@/api/user/files";
import { dateFormatter, formatBytes } from "../utils";
import { FileStatusBadge } from "./FileStatusBadge";
import { FileTypeIcon } from "./FileTypeIcon";

type UserFileDetailsDialogProps = {
  file: UserFileRecord | null;
  tenantLabel?: string;
  tenantId?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDownload?: (file: UserFileRecord) => void;
};

export function UserFileDetailsDialog({
  file,
  tenantLabel,
  open,
  onOpenChange,
  onDownload,
}: UserFileDetailsDialogProps) {
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
    onDownload?.(file);
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
        onDownload && (
          <Button variant="secondary" onClick={handleDownload}>
            <Download className="mr-2 h-4 w-4" />
            Download
          </Button>
        )
      }
    >
      <div className="space-y-4 text-sm">
        <div className="grid gap-4 sm:grid-cols-2">
          <DetailItem label="Tenant">{tenantLabel ?? "—"}</DetailItem>
          <DetailItem label="Purpose" className="capitalize">
            {file.purpose || "—"}
          </DetailItem>
          <DetailItem label="Size">{formatBytes(file.bytes)}</DetailItem>
          <DetailItem label="Content type">{file.content_type}</DetailItem>
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
    </DetailsDialog>
  );
}
