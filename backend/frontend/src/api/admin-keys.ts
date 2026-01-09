/**
 * Admin Keys API
 *
 * This module provides access to admin key management.
 * Admin keys are used for system-level authentication.
 * Uses shared types from api/shared/types.ts.
 */

import { api } from "./client";
import { createResource } from "./shared/factory";
import type {
  AdminKeyRecord,
  AdminKeyScope,
  CreateAdminKeyPayload,
  CreateAdminKeyResult,
} from "./shared/types";

// ============================================================================
// Types
// ============================================================================

/** Request payload for creating an admin key */
export interface CreateAdminKeyRequest extends CreateAdminKeyPayload {}

/** Response when creating an admin key */
export interface CreateAdminKeyResponse extends CreateAdminKeyResult {}

// Re-export shared types for convenience
export type { AdminKeyRecord, AdminKeyScope };

// ============================================================================
// Factory-based Resources
// ============================================================================

/** Admin keys resource using factory pattern */
const adminKeysResource = createResource<AdminKeyRecord, CreateAdminKeyRequest>({
  basePath: "/admin-keys",
  client: api,
  transformListResponse: (data) => {
    const response = data as { admin_keys: AdminKeyRecord[] };
    return response.admin_keys;
  },
});

// ============================================================================
// Admin Key Functions
// ============================================================================

/** List all admin keys */
export async function listAdminKeys(): Promise<AdminKeyRecord[]> {
  return adminKeysResource.list();
}

/** Create a new admin key */
export async function createAdminKey(
  payload: CreateAdminKeyRequest,
): Promise<CreateAdminKeyResponse> {
  const { data } = await api.post<CreateAdminKeyResponse>("/admin-keys", payload);
  return data;
}

/** Revoke an admin key */
export async function revokeAdminKey(id: string): Promise<{ revoked: boolean }> {
  const { data } = await api.delete<{ revoked: boolean }>(`/admin-keys/${id}`);
  return data;
}

// ============================================================================
// Exported Resources (for advanced usage)
// ============================================================================

export const adminKeysResourceApi = adminKeysResource;
