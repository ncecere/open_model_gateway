import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  MembershipRecord,
  MembershipRole,
  TenantRecord,
  TenantStatus,
} from "@/api/tenants";
import {
  createTenant,
  listTenantMemberships,
  listTenantModels,
  removeTenantMembership,
  updateTenant,
  upsertTenantMembership,
  upsertTenantModels,
  updateTenantStatus,
} from "@/api/tenants";
import {
  deleteBudgetOverride,
  getBudgetDefaults,
  getTenantBudget,
  upsertBudgetOverride,
} from "@/api/budgets";
import type { UpsertBudgetOverrideRequest } from "@/api/budgets";
import { Separator } from "@/components/ui/separator";
import { listModelCatalog } from "@/api/model-catalog";
import { useToast } from "@/hooks/use-toast";
import type { AdminUser } from "@/api/users";
import { listUsers } from "@/api/users";
import {
  TenantDirectoryCard,
  TenantSummaryHeader,
  TenantCreateDialog,
  TenantEditDialog,
  TenantMembershipDialog,
  TenantMembershipSection,
  TENANTS_QUERY_KEY,
  TENANTS_DASHBOARD_KEY,
  useTenantDirectoryQuery,
  useTenantDirectoryFilters,
  useTenantCreateDialog,
  useTenantEditDialog,
  useMembershipDialog,
  INHERIT_SCHEDULE,
} from "@/features/tenants";
const MEMBERSHIPS_QUERY_KEY = (tenantId?: string) =>
  ["tenant-memberships", tenantId] as const;

const TENANT_STATUSES: TenantStatus[] = ["active", "suspended"];
const EMPTY_USERS: AdminUser[] = [];

const normalizeAliases = (aliases: string[]) =>
  [...new Set(aliases)].sort((a, b) => a.localeCompare(b));

const aliasSelectionsEqual = (a: string[], b: string[]) => {
  const left = normalizeAliases(a);
  const right = normalizeAliases(b);
  if (left.length !== right.length) {
    return false;
  }
  return left.every((value, index) => value === right[index]);
};

export function TenantsPage() {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const tenantsQuery = useTenantDirectoryQuery();

  const budgetDefaultsQuery = useQuery({
    queryKey: ["budget-defaults"],
    queryFn: getBudgetDefaults,
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

  const createTenantMutation = useMutation({
    mutationFn: createTenant,
    onSuccess: (tenant) => {
      toast({
        title: "Tenant created",
        description: `${tenant.name} is now active`,
      });
      queryClient.invalidateQueries({ queryKey: TENANTS_QUERY_KEY });
      queryClient.invalidateQueries({ queryKey: TENANTS_DASHBOARD_KEY });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to create tenant",
        description: "Please retry in a moment.",
      });
    },
  });

  const updateStatusMutation = useMutation({
    mutationFn: updateTenantStatus,
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: TENANTS_QUERY_KEY });
      queryClient.invalidateQueries({ queryKey: TENANTS_DASHBOARD_KEY });
      queryClient.invalidateQueries({
        queryKey: MEMBERSHIPS_QUERY_KEY(variables.tenantId),
      });
      toast({ title: "Tenant status updated" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to update status",
        description: "Check your permissions and try again.",
      });
    },
  });

  const tenants = tenantsQuery.data?.tenants ?? [];
  const {
    searchTerm: tenantSearch,
    setSearchTerm: setTenantSearch,
    statusFilter: tenantStatusFilter,
    setStatusFilter: setTenantStatusFilter,
    sortedTenants,
    activeCount,
  } = useTenantDirectoryFilters(tenants);
  const budgetDefaults = budgetDefaultsQuery.data;

  const modelAliases = useMemo(
    () => modelCatalog.map((entry) => entry.alias),
    [modelCatalog],
  );
  const createDialog = useTenantCreateDialog(budgetDefaults, modelAliases);
  const handleTenantStatusChange = async (
    tenantId: string,
    status: TenantStatus,
  ) => {
    const tenant = tenants.find((entry) => entry.id === tenantId);
    if (!tenant || tenant.status === status) {
      return;
    }
    try {
      await updateStatusMutation.mutateAsync({ tenantId, status });
    } catch (error) {
      console.error(error);
    }
  };

  const handleCreateTenant = async () => {
    if (!createDialog.name.trim()) {
      toast({
        variant: "destructive",
        title: "Name is required",
      });
      return;
    }

    const trimmedBudget = createDialog.budgetUsd.trim();
    const trimmedThreshold = createDialog.warningThreshold.trim();
    const scheduleSelection =
      createDialog.refreshSchedule === INHERIT_SCHEDULE
        ? undefined
        : createDialog.refreshSchedule;
    const trimmedCooldown = createDialog.alertCooldown.trim();
    const budgetValue = Number.parseFloat(trimmedBudget);
    const thresholdValue = Number.parseFloat(trimmedThreshold);
    const cooldownValue = Number.parseInt(trimmedCooldown, 10);
    const defaults = budgetDefaultsQuery.data;

    if (trimmedBudget && (!Number.isFinite(budgetValue) || budgetValue <= 0)) {
      toast({
        variant: "destructive",
        title: "Budget override must be a positive number",
      });
      return;
    }

    if (
      trimmedBudget &&
      trimmedThreshold &&
      (!Number.isFinite(thresholdValue) ||
        thresholdValue <= 0 ||
        thresholdValue > 1)
    ) {
      toast({
        variant: "destructive",
        title: "Warning threshold must be between 0 and 1",
      });
      return;
    }

    if (
      trimmedBudget &&
      trimmedCooldown &&
      (!Number.isFinite(cooldownValue) || cooldownValue <= 0)
    ) {
      toast({
        variant: "destructive",
        title: "Cooldown must be a positive integer (seconds)",
      });
      return;
    }

    if (modelCatalog.length === 0) {
      toast({
        variant: "destructive",
        title: modelCatalogQuery.isLoading
          ? "Model catalog is still loading"
          : "Add at least one model to the catalog first",
        description: modelCatalogQuery.isLoading
          ? "Please wait a moment and try again."
          : undefined,
      });
      return;
    }

    if (createDialog.selectedModels.length === 0) {
      toast({
        variant: "destructive",
        title: "Select at least one model",
      });
      return;
    }

    try {
      const tenant = await createTenantMutation.mutateAsync({
        name: createDialog.name.trim(),
        status: createDialog.status,
      });
      if (trimmedBudget) {
        const emailList = parseListInput(createDialog.alertEmails);
        const webhookList = parseListInput(createDialog.alertWebhooks);
        const payload: UpsertBudgetOverrideRequest = {
          budget_usd: budgetValue,
          warning_threshold:
            Number.isFinite(thresholdValue) &&
            thresholdValue > 0 &&
            thresholdValue <= 1
              ? thresholdValue
              : (defaults?.warning_threshold_perc ?? 0.8),
          refresh_schedule:
            scheduleSelection || defaults?.refresh_schedule || "calendar_month",
          alert_emails: emailList.length ? emailList : undefined,
          alert_webhooks: webhookList.length ? webhookList : undefined,
          alert_cooldown_seconds:
            trimmedCooldown && Number.isFinite(cooldownValue)
              ? cooldownValue
              : defaults?.alert?.cooldown_seconds,
        };

        try {
          await upsertBudgetOverride(tenant.id, payload);
        } catch (error) {
          console.error(error);
          toast({
            variant: "destructive",
            title: "Tenant created, but budget override failed",
            description: "Update the budget from the Usage tab.",
          });
        }
      }

      try {
        await upsertTenantModels(tenant.id, createDialog.selectedModels);
      } catch (error) {
        console.error(error);
        toast({
          variant: "destructive",
          title: "Tenant created, but model assignment failed",
          description: "Reopen the tenant dialog to retry.",
        });
      }

      createDialog.setName("");
      createDialog.setStatus("active");
      createDialog.setBudgetUsd("");
      createDialog.setWarningThreshold("");
      createDialog.setRefreshSchedule(INHERIT_SCHEDULE);
      createDialog.setAlertEmails("");
      createDialog.setAlertWebhooks("");
      createDialog.setAlertCooldown("");
      createDialog.setSelectedModels(modelAliases);
      createDialog.setOpen(false);
    } catch (error) {
      console.error(error);
    }
  };

  const [membershipTenantId, setMembershipTenantId] = useState<
    string | undefined
  >(undefined);
  useEffect(() => {
    if (!membershipTenantId && tenants.length > 0) {
      setMembershipTenantId(tenants[0].id);
    }
  }, [membershipTenantId, tenants]);

  const membershipsQuery = useQuery({
    queryKey: MEMBERSHIPS_QUERY_KEY(membershipTenantId),
    queryFn: () => listTenantMemberships(membershipTenantId as string),
    enabled: Boolean(membershipTenantId),
  });

  const upsertMembershipMutation = useMutation({
    mutationFn: ({
      tenantId,
      payload,
    }: {
      tenantId: string;
      payload: { email: string; role: MembershipRole };
    }) => upsertTenantMembership(tenantId, payload),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: MEMBERSHIPS_QUERY_KEY(variables.tenantId),
      });
      toast({ title: "Membership updated" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to update membership",
        description: "Check the email and try again.",
      });
    },
  });

  const removeMembershipMutation = useMutation({
    mutationFn: ({ tenantId, userId }: { tenantId: string; userId: string }) =>
      removeTenantMembership(tenantId, userId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: MEMBERSHIPS_QUERY_KEY(variables.tenantId),
      });
      toast({ title: "Membership removed" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to remove membership",
        description: "Try again shortly.",
      });
    },
  });

  const membershipDialog = useMembershipDialog(userDirectory);

  const editDialog = useTenantEditDialog();
  const [editModelsLoading, setEditModelsLoading] = useState(false);
  const [editBudgetLoading, setEditBudgetLoading] = useState(false);
  const [editHadOverride, setEditHadOverride] = useState(false);
  const [editSaving, setEditSaving] = useState(false);

  useEffect(() => {
    if (!editDialog.open || !editDialog.tenant) {
      if (!editDialog.open) {
        setEditSaving(false);
        setEditBudgetLoading(false);
        setEditModelsLoading(false);
      }
      return;
    }

    editDialog.setName(editDialog.tenant.name);
    editDialog.setStatus(editDialog.tenant.status as TenantStatus);
    const defaults = budgetDefaultsQuery.data;
    editDialog.setBudgetUsd("");
    editDialog.setWarningThreshold(
      defaults?.warning_threshold_perc != null
        ? defaults.warning_threshold_perc.toString()
        : "",
    );
    editDialog.setRefreshSchedule(INHERIT_SCHEDULE);
    editDialog.setAlertEmails((defaults?.alert?.emails ?? []).join(", "));
    editDialog.setAlertWebhooks((defaults?.alert?.webhooks ?? []).join(", "));
    const cooldownSeconds = defaults?.alert?.cooldown_seconds;
    editDialog.setAlertCooldown(
      cooldownSeconds != null ? cooldownSeconds.toString() : "",
    );
    setEditHadOverride(false);

    setEditBudgetLoading(true);
    getTenantBudget(editDialog.tenant.id)
      .then((override) => {
        if (override) {
          setEditHadOverride(true);
          editDialog.setBudgetUsd(override.budget_usd.toString());
          editDialog.setWarningThreshold(override.warning_threshold.toString());
          editDialog.setRefreshSchedule(override.refresh_schedule);
          editDialog.setAlertEmails((override.alert_emails ?? []).join(", "));
          editDialog.setAlertWebhooks((override.alert_webhooks ?? []).join(", "));
          editDialog.setAlertCooldown(
            override.alert_cooldown_seconds
              ? override.alert_cooldown_seconds.toString()
              : "",
          );
        }
      })
      .catch(() => {
        toast({
          variant: "destructive",
          title: "Failed to load budget override",
          description: "Try reopening the dialog.",
        });
      })
      .finally(() => setEditBudgetLoading(false));

    setEditModelsLoading(true);
    listTenantModels(editDialog.tenant.id)
      .then((models) => {
        editDialog.setSelectedModels(models);
        editDialog.setOriginalModels(models);
      })
      .catch(() => {
        toast({
          variant: "destructive",
          title: "Failed to load model access",
          description: "Try reopening the dialog.",
        });
      })
      .finally(() => setEditModelsLoading(false));
  }, [editDialog.open, editDialog.tenant, budgetDefaultsQuery.data, toast]);

  const toggleEditModel = (alias: string, checked: boolean) => {
    editDialog.setSelectedModels((prev) => {
      if (checked) {
        if (prev.includes(alias)) {
          return prev;
        }
        return [...prev, alias];
      }
      return prev.filter((item) => item !== alias);
    });
  };

  const handleSelectAllEditModels = () => {
    editDialog.setSelectedModels(modelAliases);
  };

  const handleClearEditModels = () => {
    editDialog.setSelectedModels([]);
  };

  const selectedMemberships: MembershipRecord[] =
    membershipsQuery.data?.memberships ?? [];

  const handleInviteMember = async () => {
    if (!membershipTenantId) return;
    if (!membershipDialog.email.trim()) {
      toast({ variant: "destructive", title: "Email required" });
      return;
    }
    try {
      await upsertMembershipMutation.mutateAsync({
        tenantId: membershipTenantId,
        payload: { email: membershipDialog.email.trim(), role: membershipDialog.role },
      });
      membershipDialog.setEmail("");
      membershipDialog.setRole("admin");
      membershipDialog.setOpen(false);
    } catch (error) {
      console.error(error);
    }
  };

  const openEditTenantDialog = (tenant: TenantRecord) => {
    editDialog.setTenant(tenant);
    editDialog.setOpen(true);
  };

  const handleManageMembers = (tenantId: string) => {
    setMembershipTenantId(tenantId);
    const section = document.getElementById("tenant-memberships");
    section?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  const handleSaveTenantDetails = async () => {
    if (!editDialog.tenant || editSaving) {
      return;
    }
    const tenantId = editDialog.tenant.id;
    const trimmedName = editDialog.name.trim();
    const trimmedBudget = editDialog.budgetUsd.trim();
    const trimmedThreshold = editDialog.warningThreshold.trim();
    const trimmedCooldown = editDialog.alertCooldown.trim();
    const scheduleSelection =
      editDialog.refreshSchedule === INHERIT_SCHEDULE
        ? undefined
        : editDialog.refreshSchedule;

    if (!trimmedName) {
      toast({ variant: "destructive", title: "Name is required" });
      return;
    }

    if (editDialog.selectedModels.length === 0) {
      toast({
        variant: "destructive",
        title: "Select at least one model",
      });
      return;
    }

    const budgetValue = Number.parseFloat(trimmedBudget);
    if (trimmedBudget && (!Number.isFinite(budgetValue) || budgetValue <= 0)) {
      toast({
        variant: "destructive",
        title: "Budget override must be a positive number",
      });
      return;
    }

    const thresholdValue = Number.parseFloat(trimmedThreshold);
    if (
      trimmedBudget &&
      trimmedThreshold &&
      (!Number.isFinite(thresholdValue) || thresholdValue <= 0 || thresholdValue > 1)
    ) {
      toast({
        variant: "destructive",
        title: "Warning threshold must be between 0 and 1",
      });
      return;
    }

    const cooldownValue = Number.parseInt(trimmedCooldown, 10);
    if (
      trimmedBudget &&
      trimmedCooldown &&
      (!Number.isFinite(cooldownValue) || cooldownValue <= 0)
    ) {
      toast({
        variant: "destructive",
        title: "Cooldown must be a positive integer (seconds)",
      });
      return;
    }

    setEditSaving(true);
    try {
      if (trimmedName !== editDialog.tenant.name) {
        await updateTenant(tenantId, { name: trimmedName });
      }

      if (editDialog.status !== editDialog.tenant.status) {
        await updateStatusMutation.mutateAsync({
          tenantId,
          status: editDialog.status,
        });
      }

      if (trimmedBudget) {
        const defaults = budgetDefaultsQuery.data;
        const payload: UpsertBudgetOverrideRequest = {
          budget_usd: budgetValue,
          warning_threshold:
            Number.isFinite(thresholdValue) && thresholdValue > 0 && thresholdValue <= 1
              ? thresholdValue
              : defaults?.warning_threshold_perc ?? 0.8,
          refresh_schedule:
            scheduleSelection || defaults?.refresh_schedule || "calendar_month",
          alert_emails: parseListInput(editDialog.alertEmails),
          alert_webhooks: parseListInput(editDialog.alertWebhooks),
          alert_cooldown_seconds:
            trimmedCooldown && Number.isFinite(cooldownValue)
              ? cooldownValue
              : defaults?.alert?.cooldown_seconds,
        };
        await upsertBudgetOverride(tenantId, payload);
      } else if (editHadOverride) {
        await deleteBudgetOverride(tenantId);
      }

      if (!aliasSelectionsEqual(editDialog.selectedModels, editDialog.originalModels)) {
        await upsertTenantModels(tenantId, editDialog.selectedModels);
        editDialog.setOriginalModels(editDialog.selectedModels);
      }

      toast({ title: "Tenant updated" });
      editDialog.setOpen(false);
      queryClient.invalidateQueries({ queryKey: TENANTS_QUERY_KEY });
      queryClient.invalidateQueries({ queryKey: TENANTS_DASHBOARD_KEY });
    } catch (error) {
      console.error(error);
      toast({
        variant: "destructive",
        title: "Failed to update tenant",
        description: "Check the form and try again.",
      });
    } finally {
      setEditSaving(false);
    }
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
      <TenantCreateDialog
        dialog={createDialog}
        statusOptions={TENANT_STATUSES}
        budgetDefaults={budgetDefaults}
        modelCatalog={modelCatalog}
        modelAliases={modelAliases}
        isModelCatalogLoading={modelCatalogQuery.isLoading}
        isSubmitting={createTenantMutation.isPending}
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
        onStatusChange={handleTenantStatusChange}
        isStatusUpdating={updateStatusMutation.isPending}
        onEditTenant={openEditTenantDialog}
        onManageMembers={handleManageMembers}
        budgetDefaults={budgetDefaults}
      />

      <div id="tenant-memberships">
        <TenantMembershipSection
          tenants={tenants}
          selectedTenantId={membershipTenantId}
          onTenantChange={setMembershipTenantId}
          memberships={selectedMemberships}
          isLoading={membershipsQuery.isLoading}
          onInviteClick={() =>
            membershipTenantId && membershipDialog.setOpen(true)
          }
          onRemoveMember={(member) => {
            if (!membershipTenantId) return;
            removeMembershipMutation.mutate({
              tenantId: membershipTenantId,
              userId: member.user_id,
            });
          }}
          isRemoving={removeMembershipMutation.isPending}
        />
      </div>

      <TenantMembershipDialog
        dialog={membershipDialog}
        isSubmitting={upsertMembershipMutation.isPending}
        usersLoading={usersQuery.isLoading}
        tenants={tenants}
        selectedTenantId={membershipTenantId}
        onSubmit={handleInviteMember}
      />

      <TenantEditDialog
        dialog={editDialog}
        statusOptions={TENANT_STATUSES}
        budgetDefaults={budgetDefaults}
        modelCatalog={modelCatalog}
        isModelCatalogLoading={modelCatalogQuery.isLoading}
        isSubmitting={editSaving}
        editModelsLoading={editModelsLoading}
        editBudgetLoading={editBudgetLoading}
        onToggleModel={toggleEditModel}
        onSelectAllModels={handleSelectAllEditModels}
        onClearModels={handleClearEditModels}
        onSubmit={handleSaveTenantDetails}
      />
    </div>
  );
}

function parseListInput(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((entry) => entry.trim())
    .filter(Boolean);
}
