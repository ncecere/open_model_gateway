import { userApi } from "../userClient";

export type UserTenant = {
  tenant_id: string;
  name: string;
  status: string;
  role: string;
  joined_at: string;
  created_at: string;
  is_personal: boolean;
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
  created_at: string;
  self?: boolean;
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
  password?: string;
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
