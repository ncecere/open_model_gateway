import { useState, useEffect } from "react";

import type {
  MembershipRole,
  TenantRecord,
  TenantStatus,
} from "@/api/tenants";
import type { AdminUser } from "@/api/users";

export const INHERIT_SCHEDULE = "__inherit__";

export function useTenantCreateDialog(defaults?: {
  warning_threshold_perc?: number;
  alert?: {
    emails?: string[];
    webhooks?: string[];
    cooldown_seconds?: number;
  };
  default_usd?: number;
}, modelAliases: string[] = []) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [status, setStatus] = useState<TenantStatus>("active");
  const [budgetUsd, setBudgetUsd] = useState("");
  const [warningThreshold, setWarningThreshold] = useState("");
  const [refreshSchedule, setRefreshSchedule] = useState<string>(INHERIT_SCHEDULE);
  const [alertEmails, setAlertEmails] = useState("");
  const [alertWebhooks, setAlertWebhooks] = useState("");
  const [alertCooldown, setAlertCooldown] = useState("");
  const [selectedModels, setSelectedModels] = useState<string[]>(modelAliases);

  useEffect(() => {
    if (open) {
      setName("");
      setStatus("active");
      setBudgetUsd("");
      setWarningThreshold(
        defaults?.warning_threshold_perc != null
          ? defaults.warning_threshold_perc.toString()
          : "",
      );
      setRefreshSchedule(INHERIT_SCHEDULE);
      setAlertEmails((defaults?.alert?.emails ?? []).join(", "));
      setAlertWebhooks((defaults?.alert?.webhooks ?? []).join(", "));
      setAlertCooldown(
        defaults?.alert?.cooldown_seconds != null
          ? defaults.alert.cooldown_seconds.toString()
          : "",
      );
      setSelectedModels(modelAliases);
    }
  }, [open, defaults, modelAliases]);

  return {
    open,
    setOpen,
    name,
    setName,
    status,
    setStatus,
    budgetUsd,
    setBudgetUsd,
    warningThreshold,
    setWarningThreshold,
    refreshSchedule,
    setRefreshSchedule,
    alertEmails,
    setAlertEmails,
    alertWebhooks,
    setAlertWebhooks,
    alertCooldown,
    setAlertCooldown,
    selectedModels,
    setSelectedModels,
  };
}

export type TenantCreateDialogState = ReturnType<typeof useTenantCreateDialog>;

export function useTenantEditDialog() {
  const [open, setOpen] = useState(false);
  const [tenant, setTenant] = useState<TenantRecord | null>(null);

  const [name, setName] = useState("");
  const [status, setStatus] = useState<TenantStatus>("active");
  const [budgetUsd, setBudgetUsd] = useState("");
  const [warningThreshold, setWarningThreshold] = useState("");
  const [refreshSchedule, setRefreshSchedule] = useState<string>(INHERIT_SCHEDULE);
  const [alertEmails, setAlertEmails] = useState("");
  const [alertWebhooks, setAlertWebhooks] = useState("");
  const [alertCooldown, setAlertCooldown] = useState("");
  const [selectedModels, setSelectedModels] = useState<string[]>([]);
  const [originalModels, setOriginalModels] = useState<string[]>([]);

  const reset = () => {
    setTenant(null);
    setName("");
    setStatus("active");
    setBudgetUsd("");
    setWarningThreshold("");
    setRefreshSchedule(INHERIT_SCHEDULE);
    setAlertEmails("");
    setAlertWebhooks("");
    setAlertCooldown("");
    setSelectedModels([]);
    setOriginalModels([]);
  };

  useEffect(() => {
    if (!open) {
      reset();
    }
  }, [open]);

  return {
    open,
    setOpen,
    tenant,
    setTenant,
    name,
    setName,
    status,
    setStatus,
    budgetUsd,
    setBudgetUsd,
    warningThreshold,
    setWarningThreshold,
    refreshSchedule,
    setRefreshSchedule,
    alertEmails,
    setAlertEmails,
    alertWebhooks,
    setAlertWebhooks,
    alertCooldown,
    setAlertCooldown,
    selectedModels,
    setSelectedModels,
    originalModels,
    setOriginalModels,
    reset,
  };
}

export type TenantEditDialogState = ReturnType<typeof useTenantEditDialog>;

export function useMembershipDialog(users: AdminUser[] = []) {
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<MembershipRole>("admin");

  useEffect(() => {
    if (!open) {
      setEmail("");
      setRole("admin");
    }
  }, [open]);

  const normalizedQuery = email.trim().toLowerCase();
  const suggestions = users
    .filter((user) => {
      if (!normalizedQuery) return true;
      const emailMatch = user.email.toLowerCase().includes(normalizedQuery);
      const nameMatch =
        user.name?.toLowerCase().includes(normalizedQuery) ?? false;
      return emailMatch || nameMatch;
    })
    .slice(0, 5);

  return {
    open,
    setOpen,
    email,
    setEmail,
    role,
    setRole,
    suggestions,
  };
}

export type MembershipDialogState = ReturnType<typeof useMembershipDialog>;
