import { userApi } from "../userClient";

export type UserTenant = {
  tenant_id: string;
  name: string;
  status: string;
  role: string;
  joined_at: string;
  created_at: string;
  is_personal: boolean;
  budget_used_usd?: number;
  budget_limit_usd?: number;
  warning_threshold?: number;
};

export type ListUserTenantsResponse = {
  tenants: UserTenant[];
};

export async function listUserTenants() {
  const { data } = await userApi.get<ListUserTenantsResponse>("/tenants");
  return data.tenants;
}

export type TenantBudgetSummary = {
  limit_usd: number;
  used_usd: number;
  remaining_usd: number;
  warning_threshold: number;
  refresh_schedule: string;
};

export type MembershipRole = "owner" | "admin" | "viewer" | "user";

export type TenantMembership = {
  tenant_id: string;
  user_id: string;
  email: string;
  role: MembershipRole;
  budget?: MemberBudget;
  created_at: string;
  self?: boolean;
};

export type MemberBudget = {
  budget_usd?: number;
  warning_threshold?: number;
  token_cap?: number;
};

export type TenantSummary = {
  id: string;
  name: string;
  status: string;
  role: string;
  created_at: string;
  budget: TenantBudgetSummary;
};

export async function getTenantSummary(tenantId: string) {
  const { data } = await userApi.get<TenantSummary>(
    `/tenants/${tenantId}/summary`,
  );
  return data;
}

export async function listTenantMemberships(tenantId: string) {
  const { data } = await userApi.get<{ memberships: TenantMembership[] }>(
    `/tenants/${tenantId}/memberships`,
  );
  return data.memberships;
}

export interface InviteTenantMemberPayload {
  email: string;
  role: MembershipRole;
}

export async function inviteTenantMember(
  tenantId: string,
  payload: InviteTenantMemberPayload,
) {
  const { data } = await userApi.post<TenantMembership>(
    `/tenants/${tenantId}/memberships`,
    payload,
  );
  return data;
}

export async function removeTenantMember(tenantId: string, userId: string) {
  await userApi.delete(`/tenants/${tenantId}/memberships/${userId}`);
}

export type TenantLimit = {
  requests_per_minute: number;
  tokens_per_minute: number;
  parallel_requests: number;
};

export type TenantLimitPayload = {
  effective: TenantLimit;
  ceiling: TenantLimit;
  overridden: boolean;
};

export type TenantLimitUpdate = Partial<TenantLimit>;

export type MemberBudgetUpdate = Partial<MemberBudget>;

export async function updateTenantMemberBudget(
  tenantId: string,
  userId: string,
  payload: MemberBudgetUpdate,
) {
  const { data } = await userApi.put(
    `/tenants/${tenantId}/memberships/${userId}/budget`,
    payload,
  );
  return data;
}

export async function getTenantLimits(tenantId: string) {
  const { data } = await userApi.get<TenantLimitPayload>(
    `/tenants/${tenantId}/limits`,
  );
  return data;
}

export async function updateTenantLimits(
  tenantId: string,
  payload: TenantLimitUpdate,
) {
  const { data } = await userApi.put<TenantLimitPayload>(
    `/tenants/${tenantId}/limits`,
    payload,
  );
  return data;
}

export async function deleteTenantLimits(tenantId: string) {
  await userApi.delete(`/tenants/${tenantId}/limits`);
}

export type TenantModelEntry = {
  alias: string;
  provider: string;
  model_type: string;
  enabled: boolean;
  tenant_assignable: boolean;
  attached: boolean;
};

export async function listTenantModels(tenantId: string) {
  const { data } = await userApi.get<{ models: TenantModelEntry[] }>(
    `/tenants/${tenantId}/models`,
  );
  return data.models;
}

export async function attachTenantModel(tenantId: string, alias: string) {
  const { data } = await userApi.post<{ attached: boolean }>(
    `/tenants/${tenantId}/models/${alias}`,
  );
  return data;
}

export async function detachTenantModel(tenantId: string, alias: string) {
  await userApi.delete(`/tenants/${tenantId}/models/${alias}`);
}
