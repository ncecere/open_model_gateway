import {
  deleteTenantRateLimits,
  updateTenant,
  upsertTenantModels,
  upsertTenantRateLimits,
} from "@/api/tenants";
import type { TenantStatus } from "@/api/tenants";
import {
  deleteBudgetOverride,
  upsertBudgetOverride,
} from "@/api/budgets";
import type { UpsertBudgetOverrideRequest, BudgetDefaults } from "@/api/budgets";
import { useToast } from "@/hooks/use-toast";
import type { TenantCreateDialogState, TenantEditDialogState } from "./useTenantDialogs";
import type { UseAdminTenantMutationsReturn } from "./useAdminTenantMutations";
import {
  validateTenantForm,
  parseTenantFormValues,
  parseListInput,
  aliasSelectionsEqual,
} from "../validation";
import { INHERIT_SCHEDULE } from "./useTenantDialogs";

export type UseTenantHandlersOptions = {
  createDialog: TenantCreateDialogState;
  editDialog: TenantEditDialogState;
  mutations: UseAdminTenantMutationsReturn;
  budgetDefaults: BudgetDefaults | undefined;
  modelCatalog: { alias: string }[];
  modelCatalogLoading: boolean;
  editBudgetHadOverride: boolean;
  editRateHadOverride: boolean;
  setEditRateHadOverride: (value: boolean) => void;
  setEditSaving: (value: boolean) => void;
};

export function useTenantHandlers({
  createDialog,
  editDialog,
  mutations,
  budgetDefaults,
  modelCatalog,
  modelCatalogLoading,
  editBudgetHadOverride,
  editRateHadOverride,
  setEditRateHadOverride,
  setEditSaving,
}: UseTenantHandlersOptions) {
  const { toast } = useToast();
  const { createTenantMutation, updateStatusMutation, handleStatusChange } = mutations;

  const handleCreateTenant = async () => {
    const error = validateTenantForm(
      {
        name: createDialog.name,
        budgetUsd: createDialog.budgetUsd,
        warningThreshold: createDialog.warningThreshold,
        alertCooldown: createDialog.alertCooldown,
        requestsPerMinute: createDialog.requestsPerMinute,
        tokensPerMinute: createDialog.tokensPerMinute,
        parallelRequests: createDialog.parallelRequests,
        selectedModels: createDialog.selectedModels,
      },
      {
        requireName: true,
        requireModels: true,
        modelCatalogLoading,
        modelCatalogEmpty: modelCatalog.length === 0,
      },
    );

    if (error) {
      toast({ variant: "destructive", title: error.message, description: error.description });
      return;
    }

    const parsed = parseTenantFormValues({
      name: createDialog.name,
      budgetUsd: createDialog.budgetUsd,
      warningThreshold: createDialog.warningThreshold,
      alertCooldown: createDialog.alertCooldown,
      requestsPerMinute: createDialog.requestsPerMinute,
      tokensPerMinute: createDialog.tokensPerMinute,
      parallelRequests: createDialog.parallelRequests,
      selectedModels: createDialog.selectedModels,
    });

    const scheduleSelection =
      createDialog.refreshSchedule === INHERIT_SCHEDULE ? undefined : createDialog.refreshSchedule;

    try {
      const tenant = await createTenantMutation.mutateAsync({
        name: parsed.trimmedName,
        status: createDialog.status,
      });

      if (parsed.trimmedBudget) {
        const emailList = parseListInput(createDialog.alertEmails);
        const webhookList = parseListInput(createDialog.alertWebhooks);
        const payload: UpsertBudgetOverrideRequest = {
          budget_usd: parsed.budgetValue,
          warning_threshold:
            Number.isFinite(parsed.thresholdValue) && parsed.thresholdValue > 0 && parsed.thresholdValue <= 1
              ? parsed.thresholdValue
              : (budgetDefaults?.warning_threshold_perc ?? 0.8),
          refresh_schedule: scheduleSelection || budgetDefaults?.refresh_schedule || "calendar_month",
          alert_emails: emailList.length ? emailList : undefined,
          alert_webhooks: webhookList.length ? webhookList : undefined,
          alert_cooldown_seconds:
            parsed.trimmedCooldown && Number.isFinite(parsed.cooldownValue)
              ? parsed.cooldownValue
              : budgetDefaults?.alert?.cooldown_seconds,
        };

        try {
          await upsertBudgetOverride(tenant.id, payload);
        } catch {
          toast({
            variant: "destructive",
            title: "Tenant created, but budget override failed",
            description: "Update the budget from the Usage tab.",
          });
        }
      }

      try {
        await upsertTenantModels(tenant.id, createDialog.selectedModels);
      } catch {
        toast({
          variant: "destructive",
          title: "Tenant created, but model assignment failed",
          description: "Reopen the tenant dialog to retry.",
        });
      }

      if (parsed.hasRateOverride) {
        try {
          await upsertTenantRateLimits(tenant.id, {
            requests_per_minute: parsed.rpmValue,
            tokens_per_minute: parsed.tpmValue,
            parallel_requests: parsed.parallelValue,
          });
        } catch {
          toast({
            variant: "destructive",
            title: "Tenant created, but rate limits failed to save",
            description: "Reopen the tenant dialog to retry.",
          });
        }
      }

      createDialog.setOpen(false);
    } catch (err) {
      console.error(err);
    }
  };

  const handleSaveTenantDetails = async () => {
    if (!editDialog.tenant) return;

    const error = validateTenantForm(
      {
        name: editDialog.name,
        budgetUsd: editDialog.budgetUsd,
        warningThreshold: editDialog.warningThreshold,
        alertCooldown: editDialog.alertCooldown,
        requestsPerMinute: editDialog.requestsPerMinute,
        tokensPerMinute: editDialog.tokensPerMinute,
        parallelRequests: editDialog.parallelRequests,
        selectedModels: editDialog.selectedModels,
      },
      { requireName: true, requireModels: true },
    );

    if (error) {
      toast({ variant: "destructive", title: error.message, description: error.description });
      return;
    }

    const tenantId = editDialog.tenant.id;
    const parsed = parseTenantFormValues({
      name: editDialog.name,
      budgetUsd: editDialog.budgetUsd,
      warningThreshold: editDialog.warningThreshold,
      alertCooldown: editDialog.alertCooldown,
      requestsPerMinute: editDialog.requestsPerMinute,
      tokensPerMinute: editDialog.tokensPerMinute,
      parallelRequests: editDialog.parallelRequests,
      selectedModels: editDialog.selectedModels,
    });

    const scheduleSelection =
      editDialog.refreshSchedule === INHERIT_SCHEDULE ? undefined : editDialog.refreshSchedule;

    setEditSaving(true);
    try {
      if (parsed.trimmedName !== editDialog.tenant.name) {
        await updateTenant(tenantId, { name: parsed.trimmedName });
      }

      if (editDialog.status !== editDialog.tenant.status) {
        await updateStatusMutation.mutateAsync({ tenantId, status: editDialog.status });
      }

      if (parsed.trimmedBudget) {
        const payload: UpsertBudgetOverrideRequest = {
          budget_usd: parsed.budgetValue,
          warning_threshold:
            Number.isFinite(parsed.thresholdValue) && parsed.thresholdValue > 0 && parsed.thresholdValue <= 1
              ? parsed.thresholdValue
              : (budgetDefaults?.warning_threshold_perc ?? 0.8),
          refresh_schedule: scheduleSelection || budgetDefaults?.refresh_schedule || "calendar_month",
          alert_emails: parseListInput(editDialog.alertEmails),
          alert_webhooks: parseListInput(editDialog.alertWebhooks),
          alert_cooldown_seconds:
            parsed.trimmedCooldown && Number.isFinite(parsed.cooldownValue)
              ? parsed.cooldownValue
              : budgetDefaults?.alert?.cooldown_seconds,
        };
        await upsertBudgetOverride(tenantId, payload);
      } else if (editBudgetHadOverride) {
        await deleteBudgetOverride(tenantId);
      }

      if (!aliasSelectionsEqual(editDialog.selectedModels, editDialog.originalModels)) {
        await upsertTenantModels(tenantId, editDialog.selectedModels);
        editDialog.setOriginalModels(editDialog.selectedModels);
      }

      if (parsed.hasRateOverride) {
        await upsertTenantRateLimits(tenantId, {
          requests_per_minute: parsed.rpmValue,
          tokens_per_minute: parsed.tpmValue,
          parallel_requests: parsed.parallelValue,
        });
        setEditRateHadOverride(true);
      } else if (editRateHadOverride) {
        await deleteTenantRateLimits(tenantId);
        setEditRateHadOverride(false);
      }

      toast({ title: "Tenant updated" });
      editDialog.setOpen(false);
    } catch (err) {
      console.error(err);
      toast({
        variant: "destructive",
        title: "Failed to update tenant",
        description: "Check the form and try again.",
      });
    } finally {
      setEditSaving(false);
    }
  };

  const handleTenantStatusChange = async (tenantId: string, status: TenantStatus, tenants: { id: string; status: string }[]) => {
    const tenant = tenants.find((t) => t.id === tenantId);
    if (!tenant || tenant.status === status) return;
    await handleStatusChange(tenantId, status);
  };

  const handleDeleteTenant = async (tenantId: string, status: string) => {
    if (status === "suspended") {
      toast({ title: "Tenant already suspended" });
      return;
    }
    await handleStatusChange(tenantId, "suspended");
  };

  return {
    handleCreateTenant,
    handleSaveTenantDetails,
    handleTenantStatusChange,
    handleDeleteTenant,
  };
}
