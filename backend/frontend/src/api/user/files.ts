/**
 * User Files API
 *
 * This module provides user-level access to file management.
 * Uses shared types from api/shared/types.ts.
 */

import { userApi } from "../userClient";
import { createNestedResource } from "../shared/factory";
import type { BaseFileRecord } from "../shared/types";

// ============================================================================
// Types
// ============================================================================

/** User file record */
export interface UserFileRecord extends BaseFileRecord {}

/** Response for listing user files */
export interface UserFilesListResponse {
  files: UserFileRecord[];
  has_more: boolean;
  next_cursor?: string;
}

/** Parameters for listing user files */
export interface UserFilesListParams {
  limit?: number;
  after?: string;
  purpose?: string;
}

// ============================================================================
// Factory-based Resources
// ============================================================================

/** User tenant files nested resource */
const tenantFilesResource = createNestedResource<UserFileRecord>({
  parentPath: "/tenants",
  childPath: "files",
  client: userApi,
  transformListResponse: (data) => {
    const response = data as { files: UserFileRecord[] };
    return response.files;
  },
});

// ============================================================================
// File Functions
// ============================================================================

/** List files for a tenant */
export async function listUserFiles(
  tenantId: string,
  params?: UserFilesListParams,
): Promise<UserFilesListResponse> {
  const { data } = await userApi.get<UserFilesListResponse>(
    `/tenants/${tenantId}/files`,
    { params },
  );
  return data;
}

/** Download a file's content */
export async function downloadUserFile(
  tenantId: string,
  fileId: string,
): Promise<{ data: Blob }> {
  return userApi.get<Blob>(`/tenants/${tenantId}/files/${fileId}/content`, {
    responseType: "blob",
  });
}

// ============================================================================
// Exported Resources (for advanced usage)
// ============================================================================

export const userTenantFilesResource = tenantFilesResource;
