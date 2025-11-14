import { api } from "./client";
import type { UserProfile } from "./types";

export type AdminUser = UserProfile;

export interface ListUsersResponse {
  users: AdminUser[];
  limit: number;
  offset: number;
}

export interface ListUsersParams {
  limit?: number;
  offset?: number;
}

export async function listUsers(params?: ListUsersParams) {
  const { data } = await api.get<ListUsersResponse>("/users", { params });
  return data;
}

export interface UserTenantMembership {
  tenant_id: string;
  tenant_name: string;
  role: string;
  status: string;
  joined_at: string;
}

export async function getUserTenants(userId: string) {
  const { data } = await api.get<{ tenants: UserTenantMembership[] }>(
    `/users/${userId}/tenants`,
  );
  return data.tenants;
}
