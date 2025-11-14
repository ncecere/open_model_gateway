import { api } from "./client";
import { isAxiosError } from "axios";

export interface BudgetAlertDefaults {
  enabled: boolean;
  emails: string[];
  webhooks: string[];
  cooldown_seconds: number;
}

export interface BudgetDefaultsUserRef {
  id: string;
  name: string;
  email: string;
}

export interface BudgetDefaultsMetadata {
  created_at?: string;
  updated_at?: string;
  created_by?: BudgetDefaultsUserRef;
  updated_by?: BudgetDefaultsUserRef;
}

export interface BudgetDefaults {
  default_usd: number;
  warning_threshold_perc: number;
  refresh_schedule: string;
  alert: BudgetAlertDefaults;
  metadata?: BudgetDefaultsMetadata;
}

export async function getBudgetDefaults() {
  const { data } = await api.get<BudgetDefaults>("/budgets/default");
  return data;
}

export interface UpdateBudgetDefaultsRequest {
  default_usd: number;
  warning_threshold: number;
  refresh_schedule: string;
  alert_emails?: string[];
  alert_webhooks?: string[];
  alert_cooldown_seconds?: number;
}

export async function updateBudgetDefaults(payload: UpdateBudgetDefaultsRequest) {
  const { data } = await api.put<BudgetDefaults>("/budgets/default", payload);
  return data;
}

export interface BudgetOverride {
  tenant_id: string;
  budget_usd: number;
  warning_threshold: number;
  refresh_schedule: string;
  alert_emails: string[];
  alert_webhooks: string[];
  alert_cooldown_seconds: number;
  last_alert_at: string | null;
  last_alert_level: string | null;
  created_at: string;
  updated_at: string;
}

export interface ListBudgetOverridesResponse {
  overrides: BudgetOverride[];
}

export async function listBudgetOverrides(tenantId?: string) {
  const { data } = await api.get<ListBudgetOverridesResponse>(
    "/budgets/overrides",
    {
      params: tenantId ? { tenant_id: tenantId } : undefined,
    },
  );
  return data.overrides;
}

export interface UpsertBudgetOverrideRequest {
  budget_usd: number;
  warning_threshold: number;
  refresh_schedule?: string;
  alert_emails?: string[];
  alert_webhooks?: string[];
  alert_cooldown_seconds?: number;
}

export async function upsertBudgetOverride(
  tenantId: string,
  payload: UpsertBudgetOverrideRequest,
) {
  const { data } = await api.put<BudgetOverride>(
    `/budgets/overrides/${tenantId}`,
    payload,
  );
  return data;
}

export async function deleteBudgetOverride(tenantId: string) {
  await api.delete(`/budgets/overrides/${tenantId}`);
}

export async function getTenantBudget(tenantId: string) {
  try {
    const { data } = await api.get<BudgetOverride>(
      `/tenants/${tenantId}/budget`,
    );
    return data;
  } catch (error) {
    if (isAxiosError(error) && error.response?.status === 404) {
      return null;
    }
    throw error;
  }
}
