import { useMemo, useState } from "react";
import {
  useTenantSummaryQuery,
  useUserTenantsQuery,
} from "../hooks/useUserData";
import {
  OverviewCard,
  TenantsList,
  TenantDetailDialog,
  type TenantMembership,
} from "../features/tenants";
import { PageHeader } from "@/components/layouts";

export function UserTenantsPage() {
  const { data: tenants, isLoading } = useUserTenantsQuery();
  const allTenants = tenants ?? [];

  const filteredTenants = useMemo<TenantMembership[]>(
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

  const totalMemberships = filteredTenants.length;
  const activeMemberships = filteredTenants.filter((t) => t.status === "active").length;
  const managedMemberships = filteredTenants.filter(
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

      <TenantsList tenants={filteredTenants} isLoading={isLoading} onManage={openTenantDetails} />

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
