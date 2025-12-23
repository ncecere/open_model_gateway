import { api } from "./client";

export type AdminKeyScope = "admin" | "system";

export interface AdminKeyRecord {
  id: string;
  name: string;
  scope: AdminKeyScope;
  prefix: string;
  owner_user_id?: string | null;
  owner_name?: string;
  owner_email?: string;
  created_by_user_id: string;
  creator_name?: string;
  creator_email?: string;
  expires_at?: string | null;
  revoked_at?: string | null;
  last_used_at?: string | null;
  created_at: string;
}

export interface CreateAdminKeyRequest {
  name: string;
  scope: AdminKeyScope;
  expires_in_seconds: number;
}

export interface CreateAdminKeyResponse {
  admin_key: AdminKeyRecord;
  token: string;
}

export async function listAdminKeys() {
  const { data } = await api.get<{ admin_keys: AdminKeyRecord[] }>(
    "/admin-keys",
  );
  return data.admin_keys;
}

export async function createAdminKey(payload: CreateAdminKeyRequest) {
  const { data } = await api.post<CreateAdminKeyResponse>(
    "/admin-keys",
    payload,
  );
  return data;
}

export async function revokeAdminKey(id: string) {
  const { data } = await api.delete<{ revoked: boolean }>(`/admin-keys/${id}`);
  return data;
}
