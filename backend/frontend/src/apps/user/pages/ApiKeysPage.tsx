import { useMemo, useState, useCallback } from "react";
import { useQueries, useQueryClient } from "@tanstack/react-query";
import { Key } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useDefaultSelection } from "@/hooks/useDefaultSelection";
import { useToast } from "@/hooks/use-toast";
import { PageHeader } from "@/components/layouts";
import { computeNextResetDate, formatScheduleLabel } from "@/features/api-keys";
import { shortDateFormatter as dateFormatter } from "@/lib/formatters";
import { getTenantSummary, type TenantBudgetSummary } from "@/api/user/tenants";
import type { UserAPIKey } from "@/api/user/api-keys";
import {
  useTenantAPIKeysQuery,
  useUserAPIKeysQuery,
  useUserTenantsQuery,
  useAllTenantAPIKeysQueries,
} from "../hooks/useUserData";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  KeyTable,
  IssuedSecretCard,
  CreateApiKeyDialog,
  RevokedKeysTable,
  BulkUserKeyActionBar,
  useApiKeyMutations,
  type IssuedSecret,
  type BudgetMeta,
  type RevokedRow,
} from "../features/api-keys";
import { revokeUserAPIKey } from "@/api/user/api-keys";

export function UserApiKeysPage() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<"personal" | "tenant" | "revoked">("personal");
  const { data: personalKeys, isLoading: personalLoading } = useUserAPIKeysQuery();
  const { data: tenants } = useUserTenantsQuery();

  const personalTenantId = useMemo(
    () => tenants?.find((t) => t.is_personal)?.tenant_id,
    [tenants],
  );
  const tenantOptions = useMemo(
    () => (tenants ?? []).filter((t) => !t.is_personal),
    [tenants],
  );
  const tenantIds = tenantOptions.map((t) => t.tenant_id);

  const [selectedTenantId, setSelectedTenantId] = useState<string>();
  useDefaultSelection({
    items: tenantOptions,
    selected: selectedTenantId,
    onChange: setSelectedTenantId,
    getValue: (t) => t.tenant_id,
  });

  const { data: tenantKeyData, isFetching: tenantKeysLoading } =
    useTenantAPIKeysQuery(selectedTenantId);
  const allTenantKeyQueries = useAllTenantAPIKeysQueries(tenantIds);

  // Budget data
  const uniqueBudgetTenantIds = useMemo(() => {
    const ids = new Set<string>();
    (personalKeys ?? []).forEach((k) => ids.add(k.tenant_id));
    tenantOptions.forEach((t) => ids.add(t.tenant_id));
    return Array.from(ids);
  }, [personalKeys, tenantOptions]);

  const tenantSummaryQueries = useQueries({
    queries: uniqueBudgetTenantIds.map((tid) => ({
      queryKey: ["user-tenant-summary", tid],
      queryFn: () => getTenantSummary(tid),
      enabled: Boolean(tid),
    })),
  });

  const tenantBudgetMap = useMemo(() => {
    const map = new Map<string, TenantBudgetSummary>();
    tenantSummaryQueries.forEach((q, i) => {
      const tid = uniqueBudgetTenantIds[i];
      if (tid && q.data?.budget) {
        map.set(tid, q.data.budget);
      }
    });
    return map;
  }, [tenantSummaryQueries, uniqueBudgetTenantIds]);

  const personalBudgetLimit: number | null = personalTenantId
    ? (tenantBudgetMap.get(personalTenantId)?.limit_usd ?? null)
    : null;
  const selectedTenantBudget = selectedTenantId
    ? tenantBudgetMap.get(selectedTenantId)
    : null;

  // Tenant data
  const tenantRole = tenantKeyData?.role;
  const tenantKeys = tenantKeyData?.api_keys ?? [];
  const tenantActiveKeys = tenantKeys.filter((k) => !k.revoked);
  const selectedTenant = tenantOptions.find((t) => t.tenant_id === selectedTenantId);
  const canManageTenant = tenantRole === "owner" || tenantRole === "admin";

  // Dialog state
  const [createOpen, setCreateOpen] = useState(false);
  const [tenantCreateOpen, setTenantCreateOpen] = useState(false);
  const [issuedSecret, setIssuedSecret] = useState<IssuedSecret | null>(null);

  // Bulk selection state
  const [personalSelectedIds, setPersonalSelectedIds] = useState<Set<string>>(new Set());
  const [tenantSelectedIds, setTenantSelectedIds] = useState<Set<string>>(new Set());
  const [bulkRevokeLoading, setBulkRevokeLoading] = useState(false);
  const [pendingBulkRevoke, setPendingBulkRevoke] = useState<"personal" | "tenant" | null>(null);

  // Mutations
  const {
    handleCopy,
    handlePersonalCreate,
    handleTenantCreate,
    handleRevoke,
    handleTenantRevoke,
    handleRotate,
    handleTenantRotate,
    createMutation,
    tenantCreateMutation,
  } = useApiKeyMutations({
    selectedTenantId,
    selectedTenantName: selectedTenant?.name,
    tenantOptions,
    onIssued: setIssuedSecret,
    onPersonalDialogClose: () => setCreateOpen(false),
    onTenantDialogClose: () => setTenantCreateOpen(false),
  });

  // Personal keys
  const personalOnlyKeys =
    personalTenantId && personalKeys
      ? personalKeys.filter((k) => k.tenant_id === personalTenantId)
      : [];
  const activeKeys = personalOnlyKeys.filter((k) => !k.revoked);
  const revokedKeys = personalOnlyKeys.filter((k) => k.revoked);

  // Revoked keys from all tenants
  const revokedRows: RevokedRow[] = useMemo(() => {
    const tenantRevoked = allTenantKeyQueries.flatMap((q, i) => {
      const meta = tenantOptions[i];
      if (!meta || !q.data) return [];
      return q.data.api_keys
        .filter((k) => k.revoked)
        .map((k) => ({ ...k, tenantLabel: meta.name }));
    });
    return [
      ...revokedKeys.map((k) => ({ ...k, tenantLabel: "Personal" })),
      ...tenantRevoked,
    ];
  }, [revokedKeys, allTenantKeyQueries, tenantOptions]);

  const revokedLoading =
    personalLoading ||
    allTenantKeyQueries.some((q) => q.isLoading || q.isFetching);

  // Stats calculation - all keys across personal and tenant
  const allActiveKeys = useMemo(() => {
    const tenantActiveCount = allTenantKeyQueries.reduce((sum, q) => {
      if (!q.data) return sum;
      return sum + q.data.api_keys.filter((k) => !k.revoked).length;
    }, 0);
    return activeKeys.length + tenantActiveCount;
  }, [activeKeys.length, allTenantKeyQueries]);

  const totalKeys = allActiveKeys + revokedRows.length;

  // Budget helpers
  const resolveBudgetMeta = (key: UserAPIKey): BudgetMeta => {
    const tenantBudget = tenantBudgetMap.get(key.tenant_id);
    const fallbackLimit = tenantBudget?.limit_usd ?? null;
    const limit =
      key.quota?.budget_usd ??
      (typeof key.quota?.budget_cents === "number"
        ? key.quota.budget_cents / 100
        : fallbackLimit);
    const used = tenantBudget?.used_usd ?? 0;
    const warning = key.quota?.warning_threshold ?? tenantBudget?.warning_threshold ?? 0.8;
    const schedule =
      key.budget_refresh_schedule || tenantBudget?.refresh_schedule || "calendar_month";
    return { limit, used, warning, schedule };
  };

  const formatResetValue = (key: UserAPIKey) => {
    const { schedule } = resolveBudgetMeta(key);
    const label = formatScheduleLabel(schedule);
    const next = computeNextResetDate(schedule);
    return next ? `${label} · ${dateFormatter.format(next)}` : label;
  };

  const formatRole = (role?: string) =>
    role ? role.charAt(0).toUpperCase() + role.slice(1) : "—";

  // Bulk revoke handlers
  const handleBulkPersonalRevoke = useCallback(async () => {
    setBulkRevokeLoading(true);
    const keysToRevoke = activeKeys.filter((k) => personalSelectedIds.has(k.id));
    try {
      for (const key of keysToRevoke) {
        await revokeUserAPIKey(key.id);
      }
      toast({ title: `${keysToRevoke.length} key(s) revoked` });
      setPersonalSelectedIds(new Set());
      setPendingBulkRevoke(null);
      void queryClient.invalidateQueries({ queryKey: ["user-api-keys"] });
    } catch {
      toast({ variant: "destructive", title: "Failed to revoke keys" });
    } finally {
      setBulkRevokeLoading(false);
    }
  }, [activeKeys, personalSelectedIds, toast, queryClient]);

  const currentSelectedIds = activeTab === "personal" ? personalSelectedIds : tenantSelectedIds;
  const currentSetSelectedIds = activeTab === "personal" ? setPersonalSelectedIds : setTenantSelectedIds;

  return (
    <div className="space-y-6">
      <PageHeader
        title="API Keys"
        description="Personal keys are always available. Tenant keys respect the role of each membership."
      />

      {/* Summary Stats */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Keys</CardTitle>
            <Key className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalKeys}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active</CardTitle>
            <div className="h-2 w-2 rounded-full bg-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{allActiveKeys}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Revoked</CardTitle>
            <div className="h-2 w-2 rounded-full bg-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{revokedRows.length}</div>
          </CardContent>
        </Card>
      </div>

      {issuedSecret ? (
        <IssuedSecretCard issued={issuedSecret} onCopy={handleCopy} />
      ) : null}

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as "personal" | "tenant" | "revoked")}
        className="space-y-6"
      >
        <TabsList>
          <TabsTrigger value="personal">Personal keys</TabsTrigger>
          <TabsTrigger value="tenant" disabled={!tenantOptions.length}>
            Tenant keys
          </TabsTrigger>
          <TabsTrigger value="revoked">Revoked keys</TabsTrigger>
        </TabsList>

        <TabsContent value="personal" className="space-y-6">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-xl font-semibold">Personal tenant</h2>
              <p className="text-sm text-muted-foreground">
                Keys scoped to your personal tenant inherit default model/budget controls.
              </p>
            </div>
            <CreateApiKeyDialog
              mode="personal"
              open={createOpen}
              onOpenChange={setCreateOpen}
              onSubmit={handlePersonalCreate}
              isSubmitting={createMutation.isPending}
              budgetLimit={personalBudgetLimit}
            />
          </div>
          <section className="grid gap-6">
            <KeyTable
              title="Active keys"
              loading={personalLoading}
              keys={activeKeys}
              variant="active"
              onRevoke={handleRevoke}
              onRotate={handleRotate}
              allowRevoke
              allowRotate
              getBudgetMeta={resolveBudgetMeta}
              selectedIds={personalSelectedIds}
              onSelectionChange={setPersonalSelectedIds}
              formatResetValue={formatResetValue}
            />
          </section>
        </TabsContent>

        <TabsContent value="tenant" className="space-y-6">
          {tenantOptions.length === 0 ? (
            <Card>
              <CardHeader>
                <CardTitle>No shared tenants</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  Join a tenant as an owner or admin to manage shared API keys.
                </p>
              </CardContent>
            </Card>
          ) : (
            <>
              <div className="flex flex-col gap-4 lg:flex-row lg:items-end">
                <div className="flex-1 space-y-2">
                  <Label>Select tenant</Label>
                  <Select value={selectedTenantId} onValueChange={setSelectedTenantId}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select tenant" />
                    </SelectTrigger>
                    <SelectContent>
                      {tenantOptions.map((t) => (
                        <SelectItem key={t.tenant_id} value={t.tenant_id}>
                          {t.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex-1 space-y-1">
                  <Label>Role</Label>
                  <p className="rounded-md border px-3 py-2 text-sm">
                    {formatRole(tenantRole)}{" "}
                    {!canManageTenant ? "(read-only)" : "(manage keys allowed)"}
                  </p>
                </div>
                <CreateApiKeyDialog
                  mode="tenant"
                  open={tenantCreateOpen}
                  onOpenChange={setTenantCreateOpen}
                  onSubmit={handleTenantCreate}
                  isSubmitting={tenantCreateMutation.isPending}
                  budgetLimit={selectedTenantBudget?.limit_usd}
                  tenantName={selectedTenant?.name}
                  disabled={!canManageTenant}
                  disabledReason="Only tenant owners and admins can create keys."
                />
              </div>
              <section className="grid gap-6">
                <KeyTable
                  title="Active keys"
                  loading={tenantKeysLoading}
                  keys={tenantActiveKeys}
                  variant="active"
                  allowRevoke={canManageTenant}
                  allowRotate={canManageTenant}
                  onRevoke={handleTenantRevoke}
                  onRotate={handleTenantRotate}
                  getBudgetMeta={resolveBudgetMeta}
                  formatResetValue={formatResetValue}
                  selectedIds={canManageTenant ? tenantSelectedIds : undefined}
                  onSelectionChange={canManageTenant ? setTenantSelectedIds : undefined}
                />
              </section>
            </>
          )}
        </TabsContent>

        <TabsContent value="revoked" className="space-y-6">
          <RevokedKeysTable keys={revokedRows} loading={revokedLoading} />
        </TabsContent>
      </Tabs>

      {/* Bulk Action Bar */}
      <BulkUserKeyActionBar
        selectedCount={currentSelectedIds.size}
        onRevoke={() => setPendingBulkRevoke(activeTab === "personal" ? "personal" : "tenant")}
        onClear={() => currentSetSelectedIds(new Set())}
        isLoading={bulkRevokeLoading}
        disabled={activeTab === "tenant" && !canManageTenant}
      />

      {/* Bulk Revoke Confirmation Dialog */}
      <AlertDialog
        open={pendingBulkRevoke !== null}
        onOpenChange={(open) => !open && setPendingBulkRevoke(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Revoke {currentSelectedIds.size} keys?
            </AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. All selected keys will stop working
              immediately.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={bulkRevokeLoading}
              onClick={handleBulkPersonalRevoke}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Revoke all
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
