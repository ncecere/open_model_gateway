import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  ActionsMenu,
  EmptyState,
  FilterBar,
  TableSkeleton,
  TablePagination,
  usePaginationFromOffset,
  type FilterOption,
} from "@/components/tables";
import type { AdminFileRecord } from "@/api/files";
import { dateFormatter, formatBytes } from "../utils";
import { Download, Eye, Trash2 } from "lucide-react";
import { FileStatusBadge } from "./FileStatusBadge";
import { FileTypeIcon } from "./FileTypeIcon";

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
  const tenantFilterOptions: FilterOption[] = [
    { value: "all", label: "All tenants" },
    ...tenants
      .filter((tenant) => !personalTenantIds.has(tenant.id))
      .map((tenant) => ({ value: tenant.id, label: tenant.name })),
  ];

  const rows = files.map((file) => {
    const isPersonal = personalTenantIds.has(file.tenant_id);
    return { file, isPersonal };
  });

  const { page, totalPages } = usePaginationFromOffset(offset, pageSize, total);

  return (
    <div className="space-y-4">
      <FilterBar
        layout="stacked"
        columns={4}
        filters={[
          {
            type: "select",
            key: "tenant",
            label: "Tenant",
            value: filters.tenant,
            onChange: (value) => onFiltersChange({ tenant: value }),
            options: tenantFilterOptions,
            placeholder: "All tenants",
          },
          {
            type: "select",
            key: "purpose",
            label: "Purpose",
            value: filters.purpose,
            onChange: (value) => onFiltersChange({ purpose: value }),
            options: PURPOSE_OPTIONS,
            placeholder: "All purposes",
          },
          {
            type: "select",
            key: "state",
            label: "State",
            value: filters.state,
            onChange: (value) => onFiltersChange({ state: value }),
            options: STATE_OPTIONS,
            placeholder: "Active",
          },
          {
            type: "search",
            key: "search",
            label: "Search",
            value: filters.search,
            onChange: onSearchChange,
            placeholder: "Filter by filename",
          },
        ]}
      />

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
            <TableSkeleton rows={4} />
          ) : !rows.length ? (
            <EmptyState
              message="No files found"
              description="Files uploaded by tenants will appear here. Try adjusting your filters."
            />
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
                  <TableHead>Expires</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map(({ file, isPersonal }) => (
                  <TableRow key={file.id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <FileTypeIcon filename={file.filename} contentType={file.content_type} />
                        <span className="font-medium" title={file.filename}>
                          {file.filename}
                        </span>
                      </div>
                    </TableCell>
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
                    <TableCell className="text-sm text-muted-foreground">
                      {file.expires_at
                        ? dateFormatter.format(new Date(file.expires_at))
                        : "—"}
                    </TableCell>
                    <TableCell className="text-right">
                      <ActionsMenu
                        label="Actions"
                        actions={[
                          {
                            label: "View details",
                            icon: <Eye className="h-4 w-4" />,
                            onClick: () => onViewDetails(file),
                          },
                          {
                            label: "Download",
                            icon: <Download className="h-4 w-4" />,
                            onClick: () => onDownload(file),
                          },
                          {
                            label: "Delete",
                            icon: <Trash2 className="h-4 w-4" />,
                            onClick: () => onDelete(file),
                            disabled: Boolean(file.deleted_at),
                            destructive: true,
                          },
                        ]}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
        <CardFooter>
          <TablePagination
            page={page}
            totalPages={totalPages}
            hasPrev={hasPrev}
            hasNext={hasNext}
            onPrev={() => onLoadMore("prev")}
            onNext={() => onLoadMore("next")}
            className="w-full"
          />
        </CardFooter>
      </Card>
    </div>
  );
}
