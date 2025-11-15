import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { listPersonalTenants, listTenants } from "@/api/tenants";
import { cancelBatch, listBatches, type BatchRecord } from "@/api/batches";
import { useToast } from "@/hooks/use-toast";
import {
  AdminBatchTable,
  BatchDetailsDialog,
  BATCH_PAGE_SIZE,
} from "@/features/batches";
import { Separator } from "@/components/ui/separator";

const TENANTS_QUERY_KEY = ["tenants", "list"] as const;

export function BatchesPage() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const tenantsQuery = useQuery({
    queryKey: TENANTS_QUERY_KEY,
    queryFn: () => listTenants({ limit: 200 }),
    staleTime: 60_000,
  });
  const tenants = useMemo(
    () => tenantsQuery.data?.tenants ?? [],
    [tenantsQuery.data],
  );

  const personalTenantsQuery = useQuery({
    queryKey: ["tenants", "personal"],
    queryFn: () => listPersonalTenants({ limit: 500 }),
    staleTime: 60_000,
  });

  const personalTenantIds = useMemo(() => {
    const ids = new Set<string>();
    personalTenantsQuery.data?.personal_tenants.forEach((tenant) => {
      ids.add(tenant.tenant_id);
    });
    return ids;
  }, [personalTenantsQuery.data]);

  const [filters, setFilters] = useState({
    tenant: "all",
    status: "all",
    search: "",
  });
  const [searchQuery, setSearchQuery] = useState("");
  const [cursorAfter, setCursorAfter] = useState<string | undefined>(undefined);
  const [cursorStack, setCursorStack] = useState<(string | null)[]>([]);
  const [selectedBatch, setSelectedBatch] = useState<{
    batch: BatchRecord;
    tenantLabel: string;
  } | null>(null);

  useEffect(() => {
    const handle = setTimeout(() => {
      setSearchQuery(filters.search.trim());
      resetPagination();
    }, 300);
    return () => clearTimeout(handle);
  }, [filters.search]);

  const resetPagination = () => {
    setCursorAfter(undefined);
    setCursorStack([]);
  };

  const batchesQuery = useQuery({
    queryKey: [
      "admin-batches",
      filters.tenant,
      filters.status,
      searchQuery,
      cursorAfter ?? null,
      BATCH_PAGE_SIZE,
    ],
    queryFn: () =>
      listBatches({
        tenant_id: filters.tenant === "all" ? undefined : filters.tenant,
        status: filters.status === "all" ? undefined : filters.status,
        q: searchQuery || undefined,
        limit: BATCH_PAGE_SIZE,
        after: cursorAfter,
      }),
    refetchInterval: 15_000,
  });

  const cancelMutation = useMutation({
    mutationFn: (batchId: string) => cancelBatch(batchId),
    onSuccess: (_, batchId) => {
      queryClient.invalidateQueries({ queryKey: ["admin-batches"] });
      toast({
        title: "Batch cancelled",
        description: `Batch ${batchId} marked cancelled.`,
      });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to cancel batch",
        description: "Try again in a few seconds.",
      });
    },
  });

  const batches: BatchRecord[] = batchesQuery.data?.data ?? [];
  const tableIsLoading =
    tenantsQuery.isLoading ||
    personalTenantsQuery.isLoading ||
    batchesQuery.isLoading;

  const handleFiltersChange = (
    next: Partial<typeof filters>,
  ) => {
    setFilters((prev) => {
      const updated = { ...prev, ...next };
      if (next.tenant !== undefined || next.status !== undefined) {
        resetPagination();
      }
      return updated;
    });
  };

  const handleSearchChange = (value: string) => {
    setFilters((prev) => ({ ...prev, search: value }));
  };

  const hasMore = batchesQuery.data?.has_more ?? false;
  const lastId = batchesQuery.data?.last_id;
  const canPrev = cursorStack.length > 0;

  const handlePaginate = (direction: "next" | "prev") => {
    if (direction === "next") {
      if (!hasMore || !lastId) {
        return;
      }
      setCursorStack((prev) => [...prev, cursorAfter ?? null]);
      setCursorAfter(lastId);
      return;
    }
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
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Batches</h1>
        <p className="text-sm text-muted-foreground">
          Monitor asynchronous workloads submitted by tenant API keys and manage
          their lifecycle.
        </p>
      </header>

      <Separator />

      <AdminBatchTable
        tenants={tenants}
        personalTenantIds={personalTenantIds}
        batches={batches}
        pageSize={BATCH_PAGE_SIZE}
        hasMore={hasMore}
        canPageBackward={canPrev}
        isLoading={tableIsLoading}
        filters={filters}
        onFiltersChange={handleFiltersChange}
        onSearchChange={handleSearchChange}
        onPaginate={handlePaginate}
        onView={(batch, tenantLabel) => setSelectedBatch({ batch, tenantLabel })}
        onCancel={(batch) => cancelMutation.mutate(batch.id)}
      />

      <BatchDetailsDialog
        batch={selectedBatch?.batch ?? null}
        tenantLabel={selectedBatch?.tenantLabel ?? ""}
        open={Boolean(selectedBatch)}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedBatch(null);
          }
        }}
      />
    </div>
  );
}
