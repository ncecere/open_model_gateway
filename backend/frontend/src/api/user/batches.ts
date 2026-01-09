/**
 * User Batches API
 *
 * This module provides user-level access to batch job management.
 * Uses shared types from api/shared/types.ts.
 */

import { userApi } from "../userClient";
import { createNestedResource } from "../shared/factory";
import type {
  BaseBatchRecord,
  BatchCounts,
  BatchErrorEntry,
  BatchErrorList,
  CursorPaginationParams,
  CursorPaginatedResponse,
} from "../shared/types";

// ============================================================================
// Types
// ============================================================================

/** User batch record */
export interface UserBatchRecord extends BaseBatchRecord {}

/** User batch counts (alias for BatchCounts) */
export type UserBatchCounts = BatchCounts;

/** User batch error entry (alias for BatchErrorEntry) */
export type UserBatchErrorEntry = BatchErrorEntry;

/** User batch error list (alias for BatchErrorList) */
export type UserBatchErrorList = BatchErrorList;

/** Response for listing user batches */
export interface UserListBatchesResponse extends CursorPaginatedResponse<UserBatchRecord> {
  tenant?: string;
}

/** Parameters for listing user batches */
export interface UserListBatchesParams extends CursorPaginationParams {}

// Re-export shared types for convenience
export type { BatchCounts, BatchErrorEntry, BatchErrorList };

// ============================================================================
// Factory-based Resources
// ============================================================================

/** User tenant batches nested resource */
const tenantBatchesResource = createNestedResource<UserBatchRecord>({
  parentPath: "/tenants",
  childPath: "batches",
  client: userApi,
  transformListResponse: (data) => {
    const response = data as { data: UserBatchRecord[] };
    return response.data;
  },
});

// ============================================================================
// Batch Functions
// ============================================================================

/** List batches for a tenant */
export async function listUserTenantBatches(
  tenantId: string,
  params?: UserListBatchesParams,
): Promise<UserListBatchesResponse> {
  const { data } = await userApi.get<UserListBatchesResponse>(
    `/tenants/${tenantId}/batches`,
    { params },
  );
  return data;
}

/** Get a specific batch for a tenant */
export async function getUserTenantBatch(
  tenantId: string,
  batchId: string,
): Promise<UserBatchRecord> {
  const batch = await tenantBatchesResource.get(tenantId, batchId);
  if (!batch) {
    throw new Error(`Batch not found: ${batchId}`);
  }
  return batch;
}

/** Cancel a batch for a tenant */
export async function cancelUserTenantBatch(
  tenantId: string,
  batchId: string,
): Promise<UserBatchRecord> {
  const { data } = await userApi.post<UserBatchRecord>(
    `/tenants/${tenantId}/batches/${batchId}/cancel`,
    {},
  );
  return data;
}

// ============================================================================
// Exported Resources (for advanced usage)
// ============================================================================

export const userTenantBatchesResource = tenantBatchesResource;
