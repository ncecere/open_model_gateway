import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  ActionsMenu,
  EmptyState,
  FilterBar,
  TableSkeleton,
  type FilterOption,
} from "@/components/tables";
import type { UserFileRecord } from "@/api/user/files";
import type { UserTenant } from "@/api/user/tenants";
import { dateFormatter, formatBytes } from "../utils";
import { Download, Eye } from "lucide-react";
import { FileStatusBadge } from "./FileStatusBadge";
import { FileTypeIcon } from "./FileTypeIcon";

export type UserFilesTableProps = {
  tenants: UserTenant[];
  tenantsLoading: boolean;
  selectedTenantId?: string;
  onTenantChange: (tenantId: string) => void;
  searchTerm: string;
  onSearchChange: (value: string) => void;
  purposeFilter: string;
  onPurposeChange: (value: string) => void;
  purposeOptions: string[];
  files: UserFileRecord[];
  total: number;
  isLoading: boolean;
  onViewFile: (file: UserFileRecord) => void;
  onDownload: (file: UserFileRecord) => void;
  hasMore?: boolean;
  onLoadMore?: () => void;
  isFetchingMore?: boolean;
};

export function UserFilesTable({
  tenants,
  tenantsLoading,
  selectedTenantId,
  onTenantChange,
  searchTerm,
  onSearchChange,
  purposeFilter,
  onPurposeChange,
  purposeOptions,
  files,
  total,
  isLoading,
  onViewFile,
  onDownload,
  hasMore,
  onLoadMore,
  isFetchingMore,
}: UserFilesTableProps) {
  const formatTenantLabel = (tenant?: UserTenant) =>
    tenant?.is_personal ? "Personal" : tenant?.name ?? "—";

  const tenantFilterOptions: FilterOption[] = tenants.map((tenant) => ({
    value: tenant.tenant_id,
    label: formatTenantLabel(tenant),
  }));

  const purposeFilterOptions: FilterOption[] = [
    { value: "all", label: "All purposes" },
    ...purposeOptions.map((purpose) => ({ value: purpose, label: purpose })),
  ];

  const showNoTenantsMessage = !tenantsLoading && !tenants.length;

  return (
    <div className="space-y-4">
      {!showNoTenantsMessage && (
        <FilterBar
          layout="stacked"
          columns={3}
          filters={[
            {
              type: "select",
              key: "tenant",
              label: "Tenant",
              value: selectedTenantId ?? "",
              onChange: onTenantChange,
              options: tenantFilterOptions,
              placeholder: "Select tenant",
              loading: tenantsLoading,
              disabled: tenantsLoading,
            },
            {
              type: "select",
              key: "purpose",
              label: "Purpose",
              value: purposeFilter,
              onChange: onPurposeChange,
              options: purposeFilterOptions,
              placeholder: "All purposes",
              disabled: !purposeOptions.length,
            },
            {
              type: "search",
              key: "search",
              label: "Search",
              value: searchTerm,
              onChange: onSearchChange,
              placeholder: "Filter by filename",
              disabled: isLoading,
            },
          ]}
        />
      )}

      <Card>
        <CardHeader className="flex items-center justify-between">
          <div>
            <CardTitle>Files</CardTitle>
            <p className="text-sm text-muted-foreground">
              {showNoTenantsMessage
                ? "You are not part of any tenants yet."
                : `Showing ${files.length} of ${total} results`}
            </p>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <TableSkeleton rows={4} />
          ) : !files.length ? (
            <EmptyState
              message="No files uploaded"
              description="Upload files via the API to use with batches or fine-tuning."
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Filename</TableHead>
                  <TableHead>Purpose</TableHead>
                  <TableHead>Size</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Expires</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {files.map((file) => (
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
                            onClick: () => onViewFile(file),
                          },
                          {
                            label: "Download",
                            icon: <Download className="h-4 w-4" />,
                            onClick: () => onDownload(file),
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
        {hasMore && (
          <CardFooter className="flex justify-center border-t pt-4">
            <Button
              variant="outline"
              disabled={isFetchingMore}
              onClick={() => onLoadMore?.()}
            >
              {isFetchingMore ? "Loading..." : "Load more"}
            </Button>
          </CardFooter>
        )}
      </Card>
    </div>
  );
}
