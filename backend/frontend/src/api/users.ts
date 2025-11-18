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

export interface CreateUserRequest {
  email: string;
  name: string;
  password?: string;
  send_invite?: boolean;
}

export interface CreateUserResponse {
  user: AdminUser;
  invite_sent?: boolean;
}

export async function createUser(payload: CreateUserRequest) {
  const { data } = await api.post<CreateUserResponse>("/users", payload);
  return data;
}

export async function sendUserInvite(userId: string) {
  await api.post(`/users/${userId}/invite`);
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
