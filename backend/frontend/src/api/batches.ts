/**
 * Admin Batches API
 *
 * This module provides admin-level access to batch job management.
 * Uses shared types from api/shared/types.ts.
 */

import { api } from "./client";
import type {
  AdminBatchRecord,
  BatchCounts,
  BatchErrorEntry,
  BatchErrorList,
  CursorPaginationParams,
  CursorPaginatedResponse,
} from "./shared/types";

// ============================================================================
// Types
// ============================================================================

/** Batch record type (alias for AdminBatchRecord) */
export type BatchRecord = AdminBatchRecord;

/** Response for listing tenant batches */
export interface ListTenantBatchesResponse extends CursorPaginatedResponse<BatchRecord> {
  tenant?: string;
}

/** Parameters for listing batches */
export interface ListBatchesParams extends CursorPaginationParams {}

/** Response for listing admin batches */
export interface ListAdminBatchesResponse extends CursorPaginatedResponse<BatchRecord> {}

/** Parameters for listing admin batches */
export interface ListAdminBatchesParams extends CursorPaginationParams {
  tenant_id?: string;
  status?: string;
  q?: string;
}

// Re-export shared types for convenience
export type { BatchCounts, BatchErrorEntry, BatchErrorList };

// ============================================================================
// Tenant Batch Functions
// ============================================================================

/** List batches for a specific tenant */
export async function listTenantBatches(
  tenantId: string,
  params?: ListBatchesParams,
): Promise<ListTenantBatchesResponse> {
  const { data } = await api.get<ListTenantBatchesResponse>(
    `/tenants/${tenantId}/batches`,
    { params },
  );
  return data;
}

/** Get a specific batch for a tenant */
export async function getTenantBatch(
  tenantId: string,
  batchId: string,
): Promise<BatchRecord> {
  const { data } = await api.get<BatchRecord>(`/tenants/${tenantId}/batches/${batchId}`);
  return data;
}

/** Cancel a batch for a tenant */
export async function cancelTenantBatch(
  tenantId: string,
  batchId: string,
): Promise<BatchRecord> {
  const { data } = await api.post<BatchRecord>(
    `/tenants/${tenantId}/batches/${batchId}/cancel`,
    {},
  );
  return data;
}

// ============================================================================
// Admin Batch Functions
// ============================================================================

/** List all batches (admin view) */
export async function listBatches(
  params?: ListAdminBatchesParams,
): Promise<ListAdminBatchesResponse> {
  const { data } = await api.get<ListAdminBatchesResponse>("/batches", { params });
  return data;
}

/** Cancel a batch by ID (admin) */
export async function cancelBatch(batchId: string): Promise<BatchRecord> {
  const { data } = await api.post<BatchRecord>(`/batches/${batchId}/cancel`, {});
  return data;
}
