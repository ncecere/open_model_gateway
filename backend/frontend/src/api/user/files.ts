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
  status: string;
  status_details?: string | null;
}

export interface UserFilesListResponse {
  files: UserFileRecord[];
  has_more: boolean;
  next_cursor?: string;
}

export async function listUserFiles(
  tenantId: string,
  params?: { limit?: number; after?: string; purpose?: string },
) {
  const { data } = await userApi.get<UserFilesListResponse>(`/tenants/${tenantId}/files`, {
    params,
  });
  return data;
}

export async function downloadUserFile(tenantId: string, fileId: string) {
  return userApi.get<Blob>(`/tenants/${tenantId}/files/${fileId}/content`, {
    responseType: "blob",
  });
}
