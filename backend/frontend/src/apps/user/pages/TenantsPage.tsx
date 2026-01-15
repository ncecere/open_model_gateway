import { useMemo, useState } from "react";
import {
  useTenantSummaryQuery,
  useUserTenantsQuery,
} from "../hooks/useUserData";
import {
  OverviewCard,
  TenantsList,
  TenantDetailDialog,
  TenantFilters,
  type TenantMembership,
  type StatusFilter,
  type RoleFilter,
} from "../features/tenants";
import { PageHeader } from "@/components/layouts";
import { Card, CardContent } from "@/components/ui/card";

export function UserTenantsPage() {
  const { data: tenants, isLoading } = useUserTenantsQuery();
  const allTenants = tenants ?? [];

  // Filter state
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [roleFilter, setRoleFilter] = useState<RoleFilter>("all");

  // All shared tenants (excluding personal)
  const sharedTenants = useMemo<TenantMembership[]>(
    () =>
      allTenants
        .filter((tenant) => !tenant.is_personal)
        .map((tenant) => ({
          tenant_id: tenant.tenant_id,
          name: tenant.name,
          status: tenant.status,
          role: tenant.role,
          budget_used_usd: tenant.budget_used_usd,
          budget_limit_usd: tenant.budget_limit_usd,
          warning_threshold: tenant.warning_threshold,
          is_personal: tenant.is_personal,
        })),
    [allTenants],
  );

  // Apply filters
  const filteredTenants = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return sharedTenants.filter((tenant) => {
      const matchesTerm = !term || tenant.name.toLowerCase().includes(term);
      const matchesStatus = statusFilter === "all" || tenant.status === statusFilter;
      const matchesRole = roleFilter === "all" || tenant.role === roleFilter;
      return matchesTerm && matchesStatus && matchesRole;
    });
  }, [sharedTenants, searchTerm, statusFilter, roleFilter]);

  const filtersActive = searchTerm !== "" || statusFilter !== "all" || roleFilter !== "all";

  const clearFilters = () => {
    setSearchTerm("");
    setStatusFilter("all");
    setRoleFilter("all");
  };

  const [selectedTenant, setSelectedTenant] = useState<string | undefined>(undefined);
  const [detailOpen, setDetailOpen] = useState(false);

  const summaryQuery = useTenantSummaryQuery(detailOpen ? selectedTenant : undefined);

  const openTenantDetails = (tenantId: string) => {
    setSelectedTenant(tenantId);
    setDetailOpen(true);
  };

  const handleDialogOpenChange = (open: boolean) => {
    setDetailOpen(open);
    if (!open) {
      setSelectedTenant(undefined);
    }
  };

  // Stats based on all shared tenants (not filtered)
  const totalMemberships = sharedTenants.length;
  const activeMemberships = sharedTenants.filter((t) => t.status === "active").length;
  const managedMemberships = sharedTenants.filter(
    (t) => t.role === "owner" || t.role === "admin",
  ).length;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Tenants"
        description="Owners and admins can manage memberships, invite teammates, and review tenant budgets from here."
      />

      <section className="grid gap-4 md:grid-cols-3">
        <OverviewCard label="Memberships" value={totalMemberships} help="Total tenants you belong to" />
        <OverviewCard label="Active" value={activeMemberships} help="Tenants currently active" />
        <OverviewCard label="Managed" value={managedMemberships} help="Tenants where you are owner/admin" />
      </section>

      <Card>
        <CardContent className="pt-4">
          <TenantFilters
            searchTerm={searchTerm}
            onSearchTermChange={setSearchTerm}
            statusFilter={statusFilter}
            onStatusFilterChange={setStatusFilter}
            roleFilter={roleFilter}
            onRoleFilterChange={setRoleFilter}
            filteredCount={filteredTenants.length}
            totalCount={sharedTenants.length}
          />
        </CardContent>
      </Card>

      <TenantsList
        tenants={filteredTenants}
        isLoading={isLoading}
        onManage={openTenantDetails}
        hasAnyTenants={sharedTenants.length > 0}
        filtersActive={filtersActive}
        onClearFilters={clearFilters}
      />

      <TenantDetailDialog
        open={detailOpen}
        onOpenChange={handleDialogOpenChange}
        tenantId={selectedTenant}
        summary={summaryQuery.data ?? undefined}
        summaryLoading={summaryQuery.isLoading}
      />
    </div>
  );
}
