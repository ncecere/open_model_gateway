import { useMemo, useState } from "react";
import { useQueries } from "@tanstack/react-query";
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
  KeyTable,
  IssuedSecretCard,
  CreateApiKeyDialog,
  RevokedKeysTable,
  useApiKeyMutations,
  type IssuedSecret,
  type BudgetMeta,
  type RevokedRow,
} from "../features/api-keys";

export function UserApiKeysPage() {
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

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">API Keys</h1>
          <p className="text-sm text-muted-foreground">
            Personal keys are always available. Tenant keys respect the role of each membership.
          </p>
        </div>
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
                />
              </section>
            </>
          )}
        </TabsContent>

        <TabsContent value="revoked" className="space-y-6">
          <RevokedKeysTable keys={revokedRows} loading={revokedLoading} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
