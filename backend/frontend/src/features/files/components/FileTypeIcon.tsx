import {
  File,
  FileJson,
  FileText,
  FileImage,
  FileCode,
  FileSpreadsheet,
  FileArchive,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

type FileTypeIconProps = {
  filename: string;
  contentType?: string;
  className?: string;
};

const EXTENSION_ICONS: Record<string, LucideIcon> = {
  // Data files
  json: FileJson,
  jsonl: FileJson,
  csv: FileSpreadsheet,
  tsv: FileSpreadsheet,
  xlsx: FileSpreadsheet,
  xls: FileSpreadsheet,
  // Text files
  txt: FileText,
  md: FileText,
  log: FileText,
  // Code files
  py: FileCode,
  js: FileCode,
  ts: FileCode,
  jsx: FileCode,
  tsx: FileCode,
  html: FileCode,
  css: FileCode,
  yaml: FileCode,
  yml: FileCode,
  // Image files
  png: FileImage,
  jpg: FileImage,
  jpeg: FileImage,
  gif: FileImage,
  svg: FileImage,
  webp: FileImage,
  // Archive files
  zip: FileArchive,
  tar: FileArchive,
  gz: FileArchive,
  rar: FileArchive,
};

const CONTENT_TYPE_ICONS: Record<string, LucideIcon> = {
  "application/json": FileJson,
  "application/jsonl": FileJson,
  "application/x-jsonlines": FileJson,
  "text/csv": FileSpreadsheet,
  "text/plain": FileText,
  "text/markdown": FileText,
  "image/png": FileImage,
  "image/jpeg": FileImage,
  "image/gif": FileImage,
  "image/svg+xml": FileImage,
  "image/webp": FileImage,
  "application/zip": FileArchive,
  "application/gzip": FileArchive,
};

function getExtension(filename: string): string {
  const parts = filename.split(".");
  return parts.length > 1 ? parts[parts.length - 1].toLowerCase() : "";
}

export function FileTypeIcon({ filename, contentType, className }: FileTypeIconProps) {
  // Try content type first
  if (contentType) {
    const Icon = CONTENT_TYPE_ICONS[contentType];
    if (Icon) {
      return <Icon className={cn("h-4 w-4 text-muted-foreground", className)} />;
    }
  }

  // Fall back to extension
  const ext = getExtension(filename);
  const Icon = EXTENSION_ICONS[ext] ?? File;

  return <Icon className={cn("h-4 w-4 text-muted-foreground", className)} />;
}
