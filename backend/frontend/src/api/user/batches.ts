import { userApi } from "../userClient";

export interface UserBatchCounts {
  total: number;
  completed: number;
  failed: number;
  cancelled: number;
}

export interface UserBatchRecord {
  id: string;
  tenant_id: string;
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
  finalizing_at?: string | null;
  failed_at?: string | null;
  expires_at?: string | null;
  counts: UserBatchCounts;
}

export interface UserListBatchesResponse {
  object: string;
  data: UserBatchRecord[];
  tenant?: string;
}

export interface UserListBatchesParams {
  limit?: number;
  offset?: number;
}

export async function listUserTenantBatches(
  tenantId: string,
  params?: UserListBatchesParams,
) {
  const { data } = await userApi.get<UserListBatchesResponse>(
    `/tenants/${tenantId}/batches`,
    { params },
  );
  return data;
}

export async function getUserTenantBatch(
  tenantId: string,
  batchId: string,
) {
  const { data } = await userApi.get<UserBatchRecord>(
    `/tenants/${tenantId}/batches/${batchId}`,
  );
  return data;
}

export async function cancelUserTenantBatch(
  tenantId: string,
  batchId: string,
) {
  const { data } = await userApi.post<UserBatchRecord>(
    `/tenants/${tenantId}/batches/${batchId}/cancel`,
    {},
  );
  return data;
}
