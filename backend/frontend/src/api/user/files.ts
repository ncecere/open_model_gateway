import { userApi } from "../userClient";

export interface UserFileRecord {
  id: string;
  tenant_id: string;
  filename: string;
  purpose: string;
  content_type: string;
  bytes: number;
  created_at: string;
  expires_at: string;
  deleted_at?: string | null;
}

export async function listUserFiles(tenantId: string, params?: { limit?: number; offset?: number }) {
  const { data } = await userApi.get<{ files: UserFileRecord[] }>(
    `/tenants/${tenantId}/files`,
    {
      params,
    },
  );
  return data.files;
}
