import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
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
    <Card>
      <CardHeader>
        <CardTitle>Files</CardTitle>
      </CardHeader>
      <CardContent>
        {showNoTenantsMessage ? (
          <p className="text-sm text-muted-foreground">You are not part of any tenants yet.</p>
        ) : (
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
                type: "search",
                key: "search",
                label: "Search",
                value: searchTerm,
                onChange: onSearchChange,
                placeholder: "Filter by filename or purpose",
                disabled: isLoading,
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
            ]}
          />
        )}

        <div className="mt-6">
          {isLoading ? (
            <TableSkeleton rows={4} rowHeight="h-10" />
          ) : !files.length ? (
            <EmptyState
              message="No files uploaded"
              description="Files uploaded for this tenant will appear here."
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Filename</TableHead>
                  <TableHead>Purpose</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Size</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Expires</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {files.map((file) => (
                  <TableRow key={file.id}>
                    <TableCell className="font-medium">{file.filename}</TableCell>
                    <TableCell>
                      <Badge variant="secondary" className="capitalize">
                        {file.purpose || "unknown"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="space-y-1">
                        <FileStatusBadge status={file.status} />
                        {file.status_details ? (
                          <p className="text-xs text-muted-foreground">{file.status_details}</p>
                        ) : null}
                      </div>
                    </TableCell>
                    <TableCell>{formatBytes(file.bytes)}</TableCell>
                    <TableCell>{dateFormatter.format(new Date(file.created_at))}</TableCell>
                    <TableCell>
                      {file.expires_at
                        ? dateFormatter.format(new Date(file.expires_at))
                        : "—"}
                    </TableCell>
                    <TableCell className="space-x-1 text-right">
                      <Button variant="ghost" size="icon" onClick={() => onViewFile(file)}>
                        <Eye className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => onDownload(file)}>
                        <Download className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
        {hasMore ? (
          <CardFooter className="flex justify-center">
            <Button
              variant="outline"
              disabled={isFetchingMore}
              onClick={() => onLoadMore?.()}
            >
              {isFetchingMore ? "Loading..." : "Load more"}
            </Button>
          </CardFooter>
        ) : null}
      </CardContent>
    </Card>
  );
}
