import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import type { TenantRecord, TenantStatus } from "@/api/tenants";
import { Separator } from "@/components/ui/separator";
import { listModelCatalog } from "@/api/model-catalog";
import { getBudgetDefaults } from "@/api/budgets";
import { getRateLimitDefaults } from "@/api/rate-limits";
import { useToast } from "@/hooks/use-toast";
import type { AdminUser } from "@/api/users";
import { listUsers } from "@/api/users";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  TenantDirectoryCard,
  TenantSummaryHeader,
  TenantCreateDialog,
  TenantEditDialog,
  TenantMembershipDialog,
  useTenantDirectoryQuery,
  useTenantDirectoryFilters,
  useTenantCreateDialog,
  useTenantEditDialog,
  useMembershipDialog,
  useAdminTenantMutations,
  useEditTenantData,
  useTenantHandlers,
} from "@/features/tenants";

const TENANT_STATUSES: TenantStatus[] = ["active", "suspended"];
const EMPTY_USERS: AdminUser[] = [];

export function TenantsPage() {
  const { toast } = useToast();
  const tenantsQuery = useTenantDirectoryQuery();
  const mutations = useAdminTenantMutations();

  const budgetDefaultsQuery = useQuery({
    queryKey: ["budget-defaults"],
    queryFn: getBudgetDefaults,
  });

  const rateLimitDefaultsQuery = useQuery({
    queryKey: ["rate-limit-defaults"],
    queryFn: getRateLimitDefaults,
  });

  const modelCatalogQuery = useQuery({
    queryKey: ["model-catalog"],
    queryFn: listModelCatalog,
  });
  const modelCatalog = modelCatalogQuery.data ?? [];

  const usersQuery = useQuery({
    queryKey: ["users", "directory"],
    queryFn: () => listUsers({ limit: 500 }),
  });
  const userDirectory = usersQuery.data?.users ?? EMPTY_USERS;

  const tenants = tenantsQuery.data?.tenants ?? [];
  const suspendedCount = tenants.filter((t) => t.status === "suspended").length;
  const {
    searchTerm: tenantSearch,
    setSearchTerm: setTenantSearch,
    statusFilter: tenantStatusFilter,
    setStatusFilter: setTenantStatusFilter,
    sortedTenants,
    activeCount,
  } = useTenantDirectoryFilters(tenants);
  const budgetDefaults = budgetDefaultsQuery.data;
  const rateLimitDefaults = rateLimitDefaultsQuery.data;

  const modelAliases = useMemo(
    () => modelCatalog.map((entry) => entry.alias),
    [modelCatalog],
  );
  const createDialog = useTenantCreateDialog(budgetDefaults, modelAliases);
  const membershipDialog = useMembershipDialog(userDirectory);
  const editDialog = useTenantEditDialog();
  const [editTab, setEditTab] = useState("overview");
  const [editSaving, setEditSaving] = useState(false);

  const {
    membershipsQuery,
    editModelsLoading,
    editBudgetLoading,
    editBudgetHadOverride,
    editRateLoading,
    editRateHadOverride,
    setEditRateHadOverride,
  } = useEditTenantData({ editDialog, budgetDefaults });

  const {
    handleCreateTenant,
    handleSaveTenantDetails,
    handleTenantStatusChange,
    handleDeleteTenant,
  } = useTenantHandlers({
    createDialog,
    editDialog,
    mutations,
    budgetDefaults,
    modelCatalog,
    modelCatalogLoading: modelCatalogQuery.isLoading,
    editBudgetHadOverride,
    editRateHadOverride,
    setEditRateHadOverride,
    setEditSaving,
  });

  const toggleEditModel = (alias: string, checked: boolean) => {
    editDialog.setSelectedModels((prev) =>
      checked ? (prev.includes(alias) ? prev : [...prev, alias]) : prev.filter((a) => a !== alias),
    );
  };

  const handleSelectAllEditModels = () => editDialog.setSelectedModels(modelAliases);
  const handleClearEditModels = () => editDialog.setSelectedModels([]);

  const handleInviteMember = async () => {
    if (!editDialog.tenant?.id) return;
    if (!membershipDialog.email.trim()) {
      toast({ variant: "destructive", title: "Email required" });
      return;
    }
    try {
      await mutations.upsertMembershipMutation.mutateAsync({
        tenantId: editDialog.tenant.id,
        payload: { email: membershipDialog.email.trim(), role: membershipDialog.role },
      });
      membershipDialog.setEmail("");
      membershipDialog.setRole("admin");
      membershipDialog.setOpen(false);
    } catch (err) {
      console.error(err);
    }
  };

  const openEditTenantDialog = (tenant: TenantRecord) => {
    editDialog.setTenant(tenant);
    setEditTab("overview");
    editDialog.setOpen(true);
  };

  return (
    <div className="space-y-6">
      <TenantSummaryHeader
        activeCount={activeCount}
        totalCount={tenants.length}
        onRefresh={() => tenantsQuery.refetch()}
        refreshing={tenantsQuery.isFetching}
        onCreate={() => createDialog.setOpen(true)}
      />
      <section className="grid gap-4 md:grid-cols-3">
        <OverviewCard label="Total tenants" value={tenants.length} help="Active and suspended tenants" />
        <OverviewCard label="Active" value={activeCount} help="Tenants currently active" />
        <OverviewCard label="Suspended" value={suspendedCount} help="Tenants paused for review" />
      </section>
      <TenantCreateDialog
        dialog={createDialog}
        statusOptions={TENANT_STATUSES}
        budgetDefaults={budgetDefaults}
        rateLimitDefaults={rateLimitDefaults}
        modelCatalog={modelCatalog}
        modelAliases={modelAliases}
        isModelCatalogLoading={modelCatalogQuery.isLoading}
        isSubmitting={mutations.createTenantMutation.isPending}
        onSubmit={handleCreateTenant}
      />
      <Separator />

      <TenantDirectoryCard
        activeCount={activeCount}
        totalCount={tenants.length}
        searchValue={tenantSearch}
        onSearchValueChange={setTenantSearch}
        statusFilter={tenantStatusFilter}
        onStatusFilterChange={setTenantStatusFilter}
        statusOptions={TENANT_STATUSES}
        isLoading={tenantsQuery.isLoading}
        tenants={tenants}
        displayTenants={sortedTenants}
        onStatusChange={(tenantId, status) => handleTenantStatusChange(tenantId, status, tenants)}
        isStatusUpdating={mutations.updateStatusMutation.isPending}
        onEditTenant={openEditTenantDialog}
        onDeleteTenant={(tenant) => handleDeleteTenant(tenant.id, tenant.status)}
        budgetDefaults={budgetDefaults}
      />

      <TenantMembershipDialog
        dialog={membershipDialog}
        isSubmitting={mutations.upsertMembershipMutation.isPending}
        usersLoading={usersQuery.isLoading}
        tenants={tenants}
        selectedTenantId={editDialog.tenant?.id}
        onSubmit={handleInviteMember}
      />

      <TenantEditDialog
        dialog={editDialog}
        statusOptions={TENANT_STATUSES}
        budgetDefaults={budgetDefaults}
        rateLimitDefaults={rateLimitDefaults}
        modelCatalog={modelCatalog}
        isModelCatalogLoading={modelCatalogQuery.isLoading}
        isSubmitting={editSaving}
        editModelsLoading={editModelsLoading}
        editBudgetLoading={editBudgetLoading}
        editRateLoading={editRateLoading}
        onToggleModel={toggleEditModel}
        onSelectAllModels={handleSelectAllEditModels}
        onClearModels={handleClearEditModels}
        onSubmit={handleSaveTenantDetails}
        memberships={membershipsQuery.data?.memberships ?? []}
        membershipsLoading={membershipsQuery.isLoading}
        onInviteMember={() => membershipDialog.setOpen(true)}
        onRemoveMember={(member) => {
          if (!editDialog.tenant?.id) return;
          mutations.removeMembershipMutation.mutate({ tenantId: editDialog.tenant.id, userId: member.user_id });
        }}
        isRemovingMember={mutations.removeMembershipMutation.isPending}
        activeTab={editTab}
        onTabChange={setEditTab}
      />
    </div>
  );
}

function OverviewCard({ label, value, help }: { label: string; value: number; help: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm text-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold">{value}</p>
        <p className="text-xs text-muted-foreground">{help}</p>
      </CardContent>
    </Card>
  );
}
