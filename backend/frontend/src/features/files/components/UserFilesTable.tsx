import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
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

  return (
    <Card>
      <CardHeader>
        <CardTitle>Files</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <div className="space-y-1.5">
            <p className="text-sm font-medium text-muted-foreground">Tenant</p>
            {tenantsLoading ? (
              <Skeleton className="h-10 w-full" />
            ) : !tenants.length ? (
              <p className="text-sm text-muted-foreground">You are not part of any tenants yet.</p>
            ) : (
              <Select
                value={selectedTenantId}
                onValueChange={onTenantChange}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select tenant" />
                </SelectTrigger>
                <SelectContent>
                  {tenants.map((tenant) => (
                    <SelectItem key={tenant.tenant_id} value={tenant.tenant_id}>
                      {formatTenantLabel(tenant)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>
          <div className="space-y-1.5">
            <p className="text-sm font-medium text-muted-foreground">Search</p>
            <Input
              value={searchTerm}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder="Filter by filename or purpose"
              disabled={isLoading}
            />
          </div>
          <div className="space-y-1.5">
            <p className="text-sm font-medium text-muted-foreground">Purpose</p>
            <Select
              value={purposeFilter}
              onValueChange={onPurposeChange}
              disabled={!purposeOptions.length}
            >
              <SelectTrigger>
                <SelectValue placeholder="All purposes" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All purposes</SelectItem>
                {purposeOptions.map((purpose) => (
                  <SelectItem key={purpose} value={purpose}>
                    {purpose}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="mt-6">
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(4)].map((_, idx) => (
                <div key={idx} className="h-10 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : !files.length ? (
            <p className="text-sm text-muted-foreground">No files uploaded for this tenant.</p>
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
