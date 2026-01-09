/**
 * Admin Files API
 *
 * This module provides admin-level access to file management.
 * Uses shared types from api/shared/types.ts.
 */

import { api } from "./client";
import { createResource } from "./shared/factory";
import type {
  AdminFileRecord,
  PaginationParams,
  PaginatedResponse,
} from "./shared/types";

// ============================================================================
// Types
// ============================================================================

/** Response for listing files */
export interface ListFilesResponse extends PaginatedResponse<AdminFileRecord> {
  object: string;
}

/** Parameters for listing files */
export interface ListFilesParams extends PaginationParams {
  tenant_id?: string;
  purpose?: string;
  state?: string;
  q?: string;
}

// Re-export shared types for convenience
export type { AdminFileRecord };

// ============================================================================
// Factory-based Resources
// ============================================================================

/** Admin files resource using factory pattern */
const filesResource = createResource<AdminFileRecord>({
  basePath: "/files",
  client: api,
  transformListResponse: (data) => {
    const response = data as { data: AdminFileRecord[] };
    return response.data;
  },
});

// ============================================================================
// File Functions
// ============================================================================

/** List all files with optional filters */
export async function listFiles(params?: ListFilesParams): Promise<ListFilesResponse> {
  const { data } = await api.get<ListFilesResponse>("/files", { params });
  return data;
}

/** Get a specific file by ID */
export async function getFile(fileId: string): Promise<AdminFileRecord> {
  const file = await filesResource.get(fileId);
  if (!file) {
    throw new Error(`File not found: ${fileId}`);
  }
  return file;
}

/** Delete a file by ID */
export async function deleteFile(fileId: string): Promise<void> {
  await filesResource.remove(fileId);
}

/** Open file content in a new browser tab (admin endpoint) */
export function downloadAdminFileContent(fileId: string): void {
  const url = `/admin/files/${fileId}/content`;
  window.open(url, "_blank", "noopener");
}

// ============================================================================
// Exported Resources (for advanced usage)
// ============================================================================

export const adminFilesResource = filesResource;
