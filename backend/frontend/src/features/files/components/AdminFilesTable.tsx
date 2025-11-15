import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import type { AdminFileRecord } from "@/api/files";
import { dateFormatter, formatBytes } from "../utils";
import { AlertTriangle, Download, Eye, MoreHorizontal, Search, Trash2 } from "lucide-react";
import { FileStatusBadge } from "./FileStatusBadge";

const PURPOSE_OPTIONS = [
  { value: "all", label: "All purposes" },
  { value: "batch", label: "Batch" },
  { value: "fine-tune", label: "Fine-tune" },
  { value: "assistants", label: "Assistants" },
];

const STATE_OPTIONS = [
  { value: "active", label: "Active" },
  { value: "deleted", label: "Deleted" },
  { value: "all", label: "All" },
];

type AdminFilesTableProps = {
  tenants: { id: string; name: string }[];
  personalTenantIds: Set<string>;
  files: AdminFileRecord[];
  total: number;
  isLoading: boolean;
  hasPrev: boolean;
  hasNext: boolean;
  filters: {
    tenant: string;
    purpose: string;
    state: string;
    search: string;
  };
  offset: number;
  pageSize: number;
  onFiltersChange: (next: Partial<AdminFilesTableProps["filters"]>) => void;
  onSearchChange: (value: string) => void;
  onLoadMore: (direction: "next" | "prev") => void;
  onViewDetails: (file: AdminFileRecord) => void;
  onDelete: (file: AdminFileRecord) => void;
  onDownload: (file: AdminFileRecord) => void;
};

export function AdminFilesTable({
  tenants,
  personalTenantIds,
  files,
  total,
  isLoading,
  hasPrev,
  hasNext,
  filters,
  offset,
  pageSize,
  onFiltersChange,
  onSearchChange,
  onLoadMore,
  onViewDetails,
  onDelete,
  onDownload,
}: AdminFilesTableProps) {
  const tenantOptions = tenants.filter((tenant) => !personalTenantIds.has(tenant.id));

  const rows = files.map((file) => {
    const isPersonal = personalTenantIds.has(file.tenant_id);
    return { file, isPersonal };
  });

  return (
    <div className="space-y-4">
      <section className="grid gap-4 lg:grid-cols-4">
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground">Tenant</p>
          <Select
            value={filters.tenant}
            onValueChange={(value) => onFiltersChange({ tenant: value })}
          >
            <SelectTrigger>
              <SelectValue placeholder="All tenants" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All tenants</SelectItem>
              {tenantOptions.map((tenant) => (
                <SelectItem key={tenant.id} value={tenant.id}>
                  {tenant.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground">Purpose</p>
          <Select
            value={filters.purpose}
            onValueChange={(value) => onFiltersChange({ purpose: value })}
          >
            <SelectTrigger>
              <SelectValue placeholder="All purposes" />
            </SelectTrigger>
            <SelectContent>
              {PURPOSE_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground">State</p>
          <Select
            value={filters.state}
            onValueChange={(value) => onFiltersChange({ state: value })}
          >
            <SelectTrigger>
              <SelectValue placeholder="Active" />
            </SelectTrigger>
            <SelectContent>
              {STATE_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground">Search</p>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-9"
              placeholder="Filter by filename"
              value={filters.search}
              onChange={(event) => onSearchChange(event.target.value)}
            />
          </div>
        </div>
      </section>

      <Card>
        <CardHeader className="flex items-center justify-between">
          <div>
            <CardTitle>Files</CardTitle>
            <p className="text-sm text-muted-foreground">
              Showing {rows.length} of {total} results
            </p>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <SkeletonTable />
          ) : !rows.length ? (
            <EmptyState />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Filename</TableHead>
                  <TableHead>Purpose</TableHead>
                  <TableHead>Size</TableHead>
                  <TableHead>Tenant</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map(({ file, isPersonal }) => (
                  <TableRow key={file.id}>
                    <TableCell className="font-medium">{file.filename}</TableCell>
                    <TableCell>
                      <Badge variant="secondary" className="capitalize">
                        {file.purpose || "unknown"}
                      </Badge>
                    </TableCell>
                    <TableCell>{formatBytes(file.bytes)}</TableCell>
                    <TableCell>
                      <div className="flex flex-col">
                        <span>{isPersonal ? "Personal" : file.tenant_name ?? "—"}</span>
                        <span className="text-xs font-mono text-muted-foreground">
                          {file.tenant_id}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="space-y-1">
                        <FileStatusBadge status={file.status} />
                        {file.status_details ? (
                          <p className="text-xs text-muted-foreground">{file.status_details}</p>
                        ) : null}
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {dateFormatter.format(new Date(file.created_at))}
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
                          <DropdownMenuSeparator />
                          <DropdownMenuItem onClick={() => onViewDetails(file)}>
                            <Eye className="mr-2 h-4 w-4" /> View details
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => onDownload(file)}>
                            <Download className="mr-2 h-4 w-4" /> Download
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            disabled={Boolean(file.deleted_at)}
                            className="text-destructive focus:text-destructive"
                            onClick={() => onDelete(file)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" /> Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
        <CardFooter className="flex items-center justify-between">
          <Button variant="outline" disabled={!hasPrev} onClick={() => onLoadMore("prev")}>
            Previous
          </Button>
          <p className="text-sm text-muted-foreground">
            Page {Math.floor(offset / pageSize) + 1} of {Math.ceil(total / pageSize) || 1}
          </p>
          <Button variant="outline" disabled={!hasNext} onClick={() => onLoadMore("next")}>
            Next
          </Button>
        </CardFooter>
      </Card>
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

function EmptyState() {
  return (
    <div className="flex flex-col items-center gap-3 rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">
      <AlertTriangle className="h-8 w-8 text-muted-foreground" />
      <div>No files found. Uploads will appear here in reverse chronological order.</div>
    </div>
  );
}

// formatFileStatus re-exported for dialog usage if needed
