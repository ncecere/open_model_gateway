import { api } from "./client";

export interface BatchCounts {
  total: number;
  completed: number;
  failed: number;
  cancelled: number;
}

export interface BatchErrorEntry {
  code: string;
  message: string;
  param?: string;
  line?: number | null;
}

export interface BatchErrorList {
  object: string;
  data: BatchErrorEntry[];
}

export interface BatchRecord {
  id: string;
  tenant_id: string;
  tenant_name?: string;
  api_key_id: string;
  endpoint: string;
  status: string;
  completion_window: string;
  max_concurrency: number;
  metadata?: Record<string, string>;
  input_file_id: string;
  output_file_id?: string | null;
  error_file_id?: string | null;
  created_at: string;
  updated_at: string;
  in_progress_at?: string | null;
  completed_at?: string | null;
  cancelled_at?: string | null;
  cancelling_at?: string | null;
  finalizing_at?: string | null;
  failed_at?: string | null;
  expires_at?: string | null;
  expired_at?: string | null;
  counts: BatchCounts;
  errors?: BatchErrorList;
}

export interface ListTenantBatchesResponse {
  object: string;
  data: BatchRecord[];
  tenant?: string;
  has_more: boolean;
  first_id?: string;
  last_id?: string;
}

export interface ListBatchesParams {
  limit?: number;
  after?: string;
}

export interface ListAdminBatchesResponse {
  object: string;
  data: BatchRecord[];
  has_more: boolean;
  first_id?: string;
  last_id?: string;
}

export interface ListAdminBatchesParams extends ListBatchesParams {
  tenant_id?: string;
  status?: string;
  q?: string;
}

export async function listTenantBatches(
  tenantId: string,
  params?: ListBatchesParams,
) {
  const { data } = await api.get<ListTenantBatchesResponse>(
    `/tenants/${tenantId}/batches`,
    { params },
  );
  return data;
}

export async function listBatches(params?: ListAdminBatchesParams) {
  const { data } = await api.get<ListAdminBatchesResponse>("/batches", {
    params,
  });
  return data;
}

export async function getTenantBatch(tenantId: string, batchId: string) {
  const { data } = await api.get<BatchRecord>(
    `/tenants/${tenantId}/batches/${batchId}`,
  );
  return data;
}

export async function cancelTenantBatch(tenantId: string, batchId: string) {
  const { data } = await api.post<BatchRecord>(
    `/tenants/${tenantId}/batches/${batchId}/cancel`,
    {},
  );
  return data;
}

export async function cancelBatch(batchId: string) {
  const { data } = await api.post<BatchRecord>(`/batches/${batchId}/cancel`, {});
  return data;
}
