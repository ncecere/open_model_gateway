import { useEffect, useMemo, useState } from "react";
import axios from "axios";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/hooks/use-toast";
import {
  useCancelUserTenantBatchMutation,
  useUserTenantBatchesQuery,
  useUserTenantsQuery,
} from "../hooks/useUserData";
import { userApi } from "@/api/userClient";
import type { UserBatchRecord } from "@/api/user/batches";
import {
  BatchDetailsDialog,
  UserBatchTable,
} from "@/features/batches";

export function UserBatchesPage() {
  const { toast } = useToast();
  const { data: tenants, isLoading } = useUserTenantsQuery();
  const tenantOptions = useMemo(
    () =>
      (tenants ?? [])
        .map((tenant) => ({
          ...tenant,
          displayName: tenant.is_personal
            ? "Personal"
            : tenant.name?.trim() || "Unnamed tenant",
        }))
        .sort((a, b) => {
          if (a.is_personal && !b.is_personal) return -1;
          if (!a.is_personal && b.is_personal) return 1;
          return a.displayName.localeCompare(b.displayName);
        }),
    [tenants],
  );
  const personalTenant = useMemo(
    () => tenantOptions.find((tenant) => tenant.is_personal),
    [tenantOptions],
  );
  const [selectedTenantId, setSelectedTenantId] = useState<string>();
  const [downloading, setDownloading] = useState<string | null>(null);
  const [selectedBatch, setSelectedBatch] = useState<UserBatchRecord | null>(null);

  useEffect(() => {
    if (!tenantOptions.length) {
      setSelectedTenantId(undefined);
      return;
    }
    if (selectedTenantId && tenantOptions.some((t) => t.tenant_id === selectedTenantId)) {
      return;
    }
    if (personalTenant) {
      setSelectedTenantId(personalTenant.tenant_id);
      return;
    }
    setSelectedTenantId(tenantOptions[0]?.tenant_id);
  }, [selectedTenantId, tenantOptions, personalTenant]);

  const selectValue = selectedTenantId ?? "";

  const batchesQuery = useUserTenantBatchesQuery(selectedTenantId);
  const cancelMutation = useCancelUserTenantBatchMutation();

  const selectedTenant = tenantOptions.find(
    (tenant) => tenant.tenant_id === selectedTenantId,
  );
  const canManage =
    selectedTenant?.role === "owner" || selectedTenant?.role === "admin";
  const batches = batchesQuery.data?.data ?? [];

  const extractFilename = (header?: string) => {
    if (!header) return undefined;
    return header
      .split(";")
      .map((segment) => segment.trim())
      .find((segment) => segment.startsWith("filename="))
      ?.replace("filename=", "")
      ?.replace(/^"+|"+$/g, "");
  };

  const handleDownload = async (batch: UserBatchRecord, kind: "output" | "errors") => {
    if (!selectedTenantId) return;
    const key = `${batch.id}-${kind}`;
    setDownloading(key);
    try {
      const response = await userApi.get(
        `/tenants/${selectedTenantId}/batches/${batch.id}/${kind}`,
        {
          responseType: "blob",
          headers: { Accept: "application/x-ndjson" },
        },
      );
      const blob = new Blob([response.data], {
        type: response.headers["content-type"] ?? "application/x-ndjson",
      });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      const parsedFilename =
        extractFilename(response.headers["content-disposition"]) ||
        `${kind === "output" ? "batch" : "batch_errors"}_${batch.id}.jsonl`;
      anchor.download =
        parsedFilename ||
        `${kind === "output" ? "batch" : "batch_errors"}_${batch.id}.jsonl`;
      document.body.appendChild(anchor);
      anchor.click();
      document.body.removeChild(anchor);
      setTimeout(() => URL.revokeObjectURL(url), 250);
    } catch (error) {
      console.error(`download ${kind} failed`, error);
      const description = axios.isAxiosError(error)
        ? error.response?.data?.error || error.message
        : (error as Error).message;
      toast({
        variant: "destructive",
        title: `Failed to fetch batch ${kind}`,
        description: description || "Please retry in a moment.",
      });
    } finally {
      setDownloading(null);
    }
  };

  const skeletonRows = (
    <div className="space-y-2">
      {[...Array(4)].map((_, idx) => (
        <Skeleton key={idx} className="h-12 w-full" />
      ))}
    </div>
  );

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Tenant Batches</h1>
        <p className="text-sm text-muted-foreground">
          View the status of JSONL batch jobs you have access to and download their
          output files.
        </p>
      </header>

      <Card>
        <CardHeader className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <CardTitle>Scope</CardTitle>
            <p className="text-xs text-muted-foreground">
              Choose a tenant to inspect outstanding batches.
            </p>
          </div>
          <Select
            value={selectValue}
            onValueChange={(value) => setSelectedTenantId(value || undefined)}
            disabled={!tenantOptions.length}
          >
            <SelectTrigger className="w-64">
              <SelectValue placeholder="Select tenant" />
            </SelectTrigger>
            <SelectContent>
              {tenantOptions.map((tenant) => (
                <SelectItem key={tenant.tenant_id} value={tenant.tenant_id}>
                  {tenant.displayName}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            skeletonRows
          ) : !selectedTenantId ? (
            <p className="text-sm text-muted-foreground">
              Join a tenant to submit batches.
            </p>
          ) : (
            <UserBatchTable
              batches={batches}
              isLoading={batchesQuery.isLoading}
              tenantName={selectedTenant?.displayName}
              canManage={canManage}
              downloadingKey={downloading}
              onView={(batch) => setSelectedBatch(batch)}
              onDownload={handleDownload}
              onCancel={
                canManage && selectedTenantId
                  ? (batch) => {
                      cancelMutation.mutate(
                        { tenantId: selectedTenantId, batchId: batch.id },
                        {
                          onSuccess: () =>
                            toast({
                              title: "Batch cancelled",
                              description: `Batch ${batch.id} marked as cancelled`,
                            }),
                          onError: () =>
                            toast({
                              variant: "destructive",
                              title: "Failed to cancel batch",
                            }),
                        },
                      );
                    }
                  : undefined
              }
              disableCancel={cancelMutation.isPending || !selectedTenantId}
            />
          )}
        </CardContent>
      </Card>

      <BatchDetailsDialog
        batch={selectedBatch}
        tenantLabel={selectedTenant?.displayName ?? ""}
        open={Boolean(selectedBatch)}
        onOpenChange={(open) => {
          if (!open) setSelectedBatch(null);
        }}
      />
    </div>
  );
}
