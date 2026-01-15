import { useEffect, useMemo, useState } from "react";
import axios from "axios";

import { useToast } from "@/hooks/use-toast";
import { useDefaultSelection } from "@/hooks/useDefaultSelection";
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
  BATCH_PAGE_SIZE,
} from "@/features/batches";
import { PageHeader } from "@/components/layouts";

export function UserBatchesPage() {
  const { toast } = useToast();
  const { data: tenants, isLoading: tenantsLoading } = useUserTenantsQuery();
  const tenantOptions = useMemo(() => tenants ?? [], [tenants]);
  const personalTenant = useMemo(
    () => tenantOptions.find((tenant) => tenant.is_personal),
    [tenantOptions],
  );
  const [selectedTenantId, setSelectedTenantId] = useState<string>();
  const [downloading, setDownloading] = useState<string | null>(null);
  const [selectedBatch, setSelectedBatch] = useState<UserBatchRecord | null>(null);
  const [cursorAfter, setCursorAfter] = useState<string | undefined>(undefined);
  const [cursorStack, setCursorStack] = useState<(string | null)[]>([]);

  useDefaultSelection({
    items: tenantOptions,
    selected: selectedTenantId,
    onChange: setSelectedTenantId,
    getValue: (tenant) => tenant.tenant_id,
    getDefault: () => personalTenant?.tenant_id ?? tenantOptions[0]?.tenant_id,
  });

  const batchesQuery = useUserTenantBatchesQuery(selectedTenantId, {
    limit: BATCH_PAGE_SIZE,
    after: cursorAfter,
  });
  const cancelMutation = useCancelUserTenantBatchMutation();

  const selectedTenant = tenantOptions.find(
    (tenant) => tenant.tenant_id === selectedTenantId,
  );
  const tenantLabel = selectedTenant?.is_personal
    ? "Personal"
    : selectedTenant?.name ?? "";
  const canManage =
    selectedTenant?.role === "owner" || selectedTenant?.role === "admin";
  const batches = batchesQuery.data?.data ?? [];
  const hasMore = batchesQuery.data?.has_more ?? false;
  const lastId = batchesQuery.data?.last_id;
  const canPrev = cursorStack.length > 0;

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

  useEffect(() => {
    setCursorAfter(undefined);
    setCursorStack([]);
  }, [selectedTenantId]);

  const handleNextPage = () => {
    if (!hasMore || !lastId) {
      return;
    }
    setCursorStack((prev) => [...prev, cursorAfter ?? null]);
    setCursorAfter(lastId);
  };

  const handlePrevPage = () => {
    setCursorStack((prev) => {
      if (!prev.length) {
        return prev;
      }
      const next = [...prev];
      const previousCursor = next.pop();
      setCursorAfter(previousCursor ?? undefined);
      return next;
    });
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Batches"
        description="View the status of JSONL batch jobs you have access to and download their output files."
      />

      <UserBatchTable
        tenants={tenantOptions}
        tenantsLoading={tenantsLoading}
        selectedTenantId={selectedTenantId}
        onTenantChange={setSelectedTenantId}
        batches={batches}
        total={batches.length}
        isLoading={batchesQuery.isLoading}
        canManage={canManage}
        downloadingKey={downloading}
        hasMore={hasMore}
        canPageBackward={canPrev}
        pageSize={BATCH_PAGE_SIZE}
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
        onNextPage={handleNextPage}
        onPrevPage={handlePrevPage}
      />

      <BatchDetailsDialog
        batch={selectedBatch}
        tenantLabel={tenantLabel}
        open={Boolean(selectedBatch)}
        onOpenChange={(open) => {
          if (!open) setSelectedBatch(null);
        }}
        onDownload={handleDownload}
        downloadingKey={downloading}
      />
    </div>
  );
}
