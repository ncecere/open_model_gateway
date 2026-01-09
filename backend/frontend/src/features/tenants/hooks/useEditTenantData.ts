import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  getTenantRateLimits,
  listTenantMemberships,
  listTenantModels,
} from "@/api/tenants";
import { getTenantBudget } from "@/api/budgets";
import { useToast } from "@/hooks/use-toast";
import type { TenantEditDialogState } from "./useTenantDialogs";
import { INHERIT_SCHEDULE } from "./useTenantDialogs";
import type { TenantStatus } from "@/api/tenants";

export type BudgetDefaults = {
  warning_threshold_perc?: number;
  refresh_schedule?: string;
  alert?: {
    emails?: string[];
    webhooks?: string[];
    cooldown_seconds?: number;
  };
};

export type UseEditTenantDataOptions = {
  editDialog: TenantEditDialogState;
  budgetDefaults?: BudgetDefaults;
};

export function useEditTenantData({ editDialog, budgetDefaults }: UseEditTenantDataOptions) {
  const { toast } = useToast();

  const [editModelsLoading, setEditModelsLoading] = useState(false);
  const [editBudgetLoading, setEditBudgetLoading] = useState(false);
  const [editBudgetHadOverride, setEditBudgetHadOverride] = useState(false);
  const [editRateLoading, setEditRateLoading] = useState(false);
  const [editRateHadOverride, setEditRateHadOverride] = useState(false);

  const membershipsQuery = useQuery({
    queryKey: ["tenant-memberships", editDialog.tenant?.id],
    queryFn: () => listTenantMemberships(editDialog.tenant?.id as string),
    enabled: Boolean(editDialog.open && editDialog.tenant?.id),
  });

  useEffect(() => {
    if (!editDialog.open || !editDialog.tenant) {
      if (!editDialog.open) {
        setEditBudgetLoading(false);
        setEditModelsLoading(false);
        setEditRateLoading(false);
        setEditRateHadOverride(false);
      }
      return;
    }

    editDialog.setName(editDialog.tenant.name);
    editDialog.setStatus(editDialog.tenant.status as TenantStatus);

    // Set defaults for budget
    editDialog.setBudgetUsd("");
    editDialog.setWarningThreshold(
      budgetDefaults?.warning_threshold_perc != null
        ? budgetDefaults.warning_threshold_perc.toString()
        : "",
    );
    editDialog.setRefreshSchedule(INHERIT_SCHEDULE);
    editDialog.setAlertEmails((budgetDefaults?.alert?.emails ?? []).join(", "));
    editDialog.setAlertWebhooks((budgetDefaults?.alert?.webhooks ?? []).join(", "));
    const cooldownSeconds = budgetDefaults?.alert?.cooldown_seconds;
    editDialog.setAlertCooldown(
      cooldownSeconds != null ? cooldownSeconds.toString() : "",
    );
    editDialog.setRequestsPerMinute("");
    editDialog.setTokensPerMinute("");
    editDialog.setParallelRequests("");
    setEditBudgetHadOverride(false);
    setEditRateHadOverride(false);

    // Load budget override
    setEditBudgetLoading(true);
    getTenantBudget(editDialog.tenant.id)
      .then((override) => {
        if (override) {
          setEditBudgetHadOverride(true);
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

    // Load model access
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

    // Load rate limits
    setEditRateLoading(true);
    getTenantRateLimits(editDialog.tenant.id)
      .then((limits) => {
        if (limits) {
          setEditRateHadOverride(true);
          editDialog.setRequestsPerMinute(limits.requests_per_minute.toString());
          editDialog.setTokensPerMinute(limits.tokens_per_minute.toString());
          editDialog.setParallelRequests(limits.parallel_requests.toString());
        }
      })
      .catch(() => {
        toast({
          variant: "destructive",
          title: "Failed to load rate limits",
          description: "Try reopening the dialog.",
        });
      })
      .finally(() => setEditRateLoading(false));
  }, [editDialog.open, editDialog.tenant, budgetDefaults, toast]);

  return {
    membershipsQuery,
    editModelsLoading,
    editBudgetLoading,
    editBudgetHadOverride,
    setEditBudgetHadOverride,
    editRateLoading,
    editRateHadOverride,
    setEditRateHadOverride,
  };
}
