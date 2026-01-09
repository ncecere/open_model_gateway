/**
 * Example usage of the API factory pattern.
 *
 * This file demonstrates how to use the factory functions
 * to create typed API endpoints with minimal boilerplate.
 *
 * NOTE: These are examples showing the pattern. The actual API implementations
 * in api/tenants.ts etc. can be migrated to use these patterns incrementally.
 */

import { api } from "../client";
import { userApi } from "../userClient";
import {
  createResource,
  createNestedResource,
  createEndpoint,
} from "./factory";
import { queryKeys } from "./queryKeys";
import type {
  BaseTenantRecord,
  TenantStatus,
  TenantBudgetInfo,
  BaseMembershipRecord,
  MembershipRole,
  BaseApiKeyRecord,
  CreateApiKeyPayload,
  CreateApiKeyResult,
  RateLimitOverride,
} from "./types";

// ============================================================================
// Admin API Example: Tenants
// ============================================================================

/**
 * Extended tenant record for admin API.
 */
export interface AdminTenantRecord extends BaseTenantRecord, TenantBudgetInfo {}

/**
 * Payload for creating a tenant.
 */
export interface CreateTenantPayload {
  name: string;
  status?: TenantStatus;
}

/**
 * Payload for updating a tenant.
 */
export interface UpdateTenantPayload {
  name?: string;
}

/**
 * Admin Tenants API using the factory pattern.
 *
 * @example
 * ```ts
 * // List all tenants
 * const tenants = await adminTenantsApi.list({ limit: 50 });
 *
 * // Get a single tenant
 * const tenant = await adminTenantsApi.get("tenant-123");
 *
 * // Create a new tenant
 * const newTenant = await adminTenantsApi.create({ name: "Acme Corp" });
 *
 * // Update a tenant
 * const updated = await adminTenantsApi.patch("tenant-123", { name: "New Name" });
 *
 * // Delete a tenant
 * await adminTenantsApi.remove("tenant-123");
 * ```
 */
export const adminTenantsApi = createResource<
  AdminTenantRecord,
  CreateTenantPayload,
  UpdateTenantPayload
>({
  basePath: "/tenants",
  client: api,
  transformListResponse: (data) => {
    const response = data as { tenants: AdminTenantRecord[] };
    return response.tenants;
  },
});

/**
 * Custom endpoint for updating tenant status.
 */
export const updateTenantStatus = createEndpoint({ client: api }).patch<
  AdminTenantRecord,
  { status: TenantStatus },
  { tenantId: string }
>("/tenants/:tenantId/status");

// ============================================================================
// Admin API Example: Tenant Memberships
// ============================================================================

/**
 * Admin membership record.
 */
export interface AdminMembershipRecord extends BaseMembershipRecord {}

/**
 * Payload for upserting a membership.
 */
export interface UpsertMembershipPayload {
  email: string;
  role: MembershipRole;
  password?: string;
}

/**
 * Admin Tenant Memberships API using nested resource factory.
 *
 * @example
 * ```ts
 * // List memberships for a tenant
 * const members = await adminMembershipsApi.list("tenant-123");
 *
 * // Add a member to a tenant
 * const member = await adminMembershipsApi.create("tenant-123", {
 *   email: "user@example.com",
 *   role: "admin",
 * });
 *
 * // Remove a member
 * await adminMembershipsApi.remove("tenant-123", "user-456");
 * ```
 */
export const adminMembershipsApi = createNestedResource<
  AdminMembershipRecord,
  UpsertMembershipPayload
>({
  parentPath: "/tenants",
  childPath: "memberships",
  client: api,
  transformListResponse: (data) => {
    const response = data as { memberships: AdminMembershipRecord[] };
    return response.memberships;
  },
});

// ============================================================================
// Admin API Example: Tenant API Keys
// ============================================================================

/**
 * Admin API key record with tenant name.
 */
export interface AdminApiKeyRecord extends BaseApiKeyRecord {
  tenant_name?: string;
}

/**
 * Admin Tenant API Keys using nested resource factory.
 *
 * @example
 * ```ts
 * // List API keys for a tenant
 * const keys = await adminApiKeysApi.list("tenant-123");
 *
 * // Create an API key (returns secret)
 * const result = await createTenantApiKey("tenant-123", { name: "Production" });
 * console.log("Save this secret:", result.secret);
 *
 * // Revoke an API key
 * await adminApiKeysApi.remove("tenant-123", "key-456");
 * ```
 */
export const adminApiKeysApi = createNestedResource<
  AdminApiKeyRecord,
  CreateApiKeyPayload
>({
  parentPath: "/tenants",
  childPath: "api-keys",
  client: api,
  transformListResponse: (data) => {
    const response = data as { api_keys: AdminApiKeyRecord[] };
    return response.api_keys;
  },
});

/**
 * Create tenant API key (special endpoint that returns secret).
 */
export async function createTenantApiKeyExample(
  tenantId: string,
  payload: CreateApiKeyPayload,
): Promise<CreateApiKeyResult<AdminApiKeyRecord>> {
  const { data } = await api.post(`/tenants/${tenantId}/api-keys`, payload);
  return data as CreateApiKeyResult<AdminApiKeyRecord>;
}

// ============================================================================
// Admin API Example: Rate Limits
// ============================================================================

/**
 * Get tenant rate limits with 404 handling.
 *
 * @example
 * ```ts
 * const limits = await getTenantRateLimitsExample("tenant-123");
 * if (limits) {
 *   console.log("RPM:", limits.requests_per_minute);
 * } else {
 *   console.log("No custom rate limits set");
 * }
 * ```
 */
export async function getTenantRateLimitsExample(
  tenantId: string,
): Promise<RateLimitOverride | null> {
  const response = await api.get(`/tenants/${tenantId}/rate-limits`, {
    validateStatus: (status) =>
      (status >= 200 && status < 300) || status === 404,
  });

  if (response.status === 404) {
    return null;
  }

  return response.data as RateLimitOverride;
}

/**
 * Upsert tenant rate limits.
 */
export const upsertTenantRateLimits = createEndpoint({ client: api }).put<
  RateLimitOverride,
  RateLimitOverride,
  { tenantId: string }
>("/tenants/:tenantId/rate-limits");

/**
 * Delete tenant rate limits.
 */
export const deleteTenantRateLimits = createEndpoint({ client: api }).delete<{
  tenantId: string;
}>("/tenants/:tenantId/rate-limits");

// ============================================================================
// Query Keys Usage Example
// ============================================================================

/**
 * Example showing how to use query keys with React Query.
 *
 * @example
 * ```ts
 * // In a component:
 * const tenantsQuery = useQuery({
 *   queryKey: queryKeys.tenants.list(),
 *   queryFn: () => adminTenantsApi.list(),
 * });
 *
 * const tenantQuery = useQuery({
 *   queryKey: queryKeys.tenants.detail("tenant-123").key,
 *   queryFn: () => adminTenantsApi.get("tenant-123"),
 * });
 *
 * const membersQuery = useQuery({
 *   queryKey: queryKeys.tenantMemberships.list("tenant-123"),
 *   queryFn: () => adminMembershipsApi.list("tenant-123"),
 * });
 *
 * // Invalidation:
 * queryClient.invalidateQueries({ queryKey: queryKeys.tenants.all });
 * queryClient.invalidateQueries({ queryKey: queryKeys.tenantMemberships.all("tenant-123") });
 * ```
 */
export function exampleQueryKeysUsage() {
  // These are just for documentation - not meant to be called
  return {
    tenantsList: queryKeys.tenants.list(),
    tenantsListWithParams: queryKeys.tenants.list({ status: "active" }),
    tenantDetail: queryKeys.tenants.detail("tenant-123").key,
    tenantMemberships: queryKeys.tenants.detail("tenant-123").nested("memberships"),
    membershipsList: queryKeys.tenantMemberships.list("tenant-123"),
    membershipDetail: queryKeys.tenantMemberships.detail("tenant-123", "user-456"),
  };
}

// ============================================================================
// User API Example
// ============================================================================

/**
 * User tenant with role information.
 */
export interface UserTenantRecord {
  tenant_id: string;
  name: string;
  status: string;
  role: string;
  joined_at: string;
  created_at: string;
  is_personal: boolean;
  budget_used_usd?: number;
  budget_limit_usd?: number;
  warning_threshold?: number;
}

/**
 * User Tenants API (read-only for regular users).
 *
 * @example
 * ```ts
 * // List tenants user has access to
 * const myTenants = await userTenantsApi.list();
 * ```
 */
export const userTenantsApi = createResource<UserTenantRecord>({
  basePath: "/tenants",
  client: userApi,
  transformListResponse: (data) => {
    const response = data as { tenants: UserTenantRecord[] };
    return response.tenants;
  },
});
