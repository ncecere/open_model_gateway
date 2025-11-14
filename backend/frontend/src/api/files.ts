import { api } from "./client";

export interface AdminFileRecord {
  id: string;
  tenant_id: string;
  tenant_name: string;
  filename: string;
  purpose: string;
  content_type: string;
  bytes: number;
  storage_backend: string;
  encrypted: boolean;
  checksum: string;
  expires_at: string;
  created_at: string;
  deleted_at?: string | null;
}

export interface ListFilesResponse {
  object: string;
  data: AdminFileRecord[];
  limit: number;
  offset: number;
  total: number;
}

export interface ListFilesParams {
  tenant_id?: string;
  purpose?: string;
  state?: string;
  q?: string;
  limit?: number;
  offset?: number;
}

export async function listFiles(params?: ListFilesParams) {
  const { data } = await api.get<ListFilesResponse>("/files", { params });
  return data;
}

export async function getFile(fileId: string) {
  const { data } = await api.get<AdminFileRecord>(`/files/${fileId}`);
  return data;
}

export async function deleteFile(fileId: string) {
  await api.delete(`/files/${fileId}`);
}
