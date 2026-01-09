/**
 * Admin Tenants API
 *
 * This module provides admin-level access to tenant management.
 * Uses the factory pattern from api/shared/factory.ts.
 */

import { api } from "./client";
import { createResource, createNestedResource } from "./shared/factory";
import type {
  BaseTenantRecord,
  TenantBudgetInfo,
  TenantStatus,
  BaseMembershipRecord,
  MembershipRole,
  BaseApiKeyRecord,
  ApiKeyIssuer,
  QuotaPayload,
  RateLimitInfo,
  ApiKeyRateLimits,
  RateLimitOverride,
  PersonalTenantRecord as SharedPersonalTenantRecord,
  PaginationParams,
} from "./shared/types";

// Re-export PersonalTenantRecord for backward compatibility
export type PersonalTenantRecord = SharedPersonalTenantRecord;

// ============================================================================
// Types
// ============================================================================

/** Admin tenant record with budget information */
export interface TenantRecord extends BaseTenantRecord, TenantBudgetInfo {}

/** Response for listing tenants */
export interface ListTenantsResponse {
  tenants: TenantRecord[];
  limit: number;
  offset: number;
}

/** Parameters for listing tenants */
export interface ListTenantsParams extends PaginationParams {}

/** Payload for creating a tenant */
export interface CreateTenantRequest {
  name: string;
  status?: TenantStatus;
}

/** Payload for updating a tenant */
export interface UpdateTenantRequest {
  name: string;
}

/** Payload for updating tenant status */
export interface UpdateTenantStatusRequest {
  tenantId: string;
  status: TenantStatus;
}

/** Response for listing personal tenants */
export interface ListPersonalTenantsResponse {
  personal_tenants: PersonalTenantRecord[];
  limit: number;
  offset: number;
}

/** Admin membership record */
export interface MembershipRecord extends BaseMembershipRecord {}

/** Response for listing memberships */
export interface ListMembershipsResponse {
  memberships: MembershipRecord[];
}

/** Payload for upserting a membership */
export interface UpsertMembershipRequest {
  email: string;
  role: MembershipRole;
  password?: string;
}

/** Admin API key record with optional tenant name */
export interface ApiKeyRecord extends BaseApiKeyRecord {
  tenant_name?: string;
  issuer?: ApiKeyIssuer;
}

/** Response for listing API keys */
export interface ListTenantApiKeysResponse {
  api_keys: ApiKeyRecord[];
}

/** Payload for creating an API key */
export interface CreateApiKeyRequest {
  name: string;
  scopes?: string[];
  quota?: QuotaPayload;
  rate_limits?: ApiKeyRateLimitOverride;
}

/** API key rate limit override */
export interface ApiKeyRateLimitOverride extends RateLimitInfo {}

/** Response when creating an API key */
export interface CreateApiKeyResponse {
  api_key: ApiKeyRecord;
  secret: string;
  token: string;
}

/** Tenant rate limit override */
export interface TenantRateLimitOverride extends RateLimitOverride {}

// Re-export shared types for convenience
export type { TenantStatus, MembershipRole, QuotaPayload, RateLimitInfo, ApiKeyRateLimits };

// ============================================================================
// Factory-based Resources
// ============================================================================

/** Admin tenants resource using factory pattern */
const tenantsResource = createResource<TenantRecord, CreateTenantRequest, UpdateTenantRequest>({
  basePath: "/tenants",
  client: api,
  transformListResponse: (data) => {
    const response = data as { tenants: TenantRecord[] };
    return response.tenants;
  },
});

/** Admin tenant memberships nested resource */
const membershipsResource = createNestedResource<MembershipRecord, UpsertMembershipRequest>({
  parentPath: "/tenants",
  childPath: "memberships",
  client: api,
  transformListResponse: (data) => {
    const response = data as { memberships: MembershipRecord[] };
    return response.memberships;
  },
});

/** Admin tenant API keys nested resource */
const apiKeysResource = createNestedResource<ApiKeyRecord, CreateApiKeyRequest>({
  parentPath: "/tenants",
  childPath: "api-keys",
  client: api,
  transformListResponse: (data) => {
    const response = data as { api_keys: ApiKeyRecord[] };
    return response.api_keys;
  },
});

// ============================================================================
// Tenant Functions
// ============================================================================

/** List all tenants with optional pagination */
export async function listTenants(params?: ListTenantsParams): Promise<ListTenantsResponse> {
  const { data } = await api.get<ListTenantsResponse>("/tenants", { params });
  return data;
}

/** List all personal tenants */
export async function listPersonalTenants(
  params?: ListTenantsParams,
): Promise<ListPersonalTenantsResponse> {
  const { data } = await api.get<ListPersonalTenantsResponse>("/tenants/personal", { params });
  return data;
}

/** Create a new tenant */
export async function createTenant(payload: CreateTenantRequest): Promise<TenantRecord> {
  return tenantsResource.create(payload);
}

/** Update a tenant */
export async function updateTenant(
  tenantId: string,
  payload: UpdateTenantRequest,
): Promise<TenantRecord> {
  return tenantsResource.patch(tenantId, payload);
}

/** Update a tenant's status */
export async function updateTenantStatus({
  tenantId,
  status,
}: UpdateTenantStatusRequest): Promise<TenantRecord> {
  const { data } = await api.patch<TenantRecord>(`/tenants/${tenantId}/status`, { status });
  return data;
}

// ============================================================================
// API Key Functions
// ============================================================================

/** List API keys for a tenant */
export async function listTenantApiKeys(tenantId: string): Promise<ListTenantApiKeysResponse> {
  const keys = await apiKeysResource.list(tenantId);
  return { api_keys: keys };
}

/** Create an API key for a tenant */
export async function createTenantApiKey(
  tenantId: string,
  payload: CreateApiKeyRequest,
): Promise<CreateApiKeyResponse> {
  const { data } = await api.post<CreateApiKeyResponse>(
    `/tenants/${tenantId}/api-keys`,
    payload,
  );
  return data;
}

/** Revoke an API key */
export async function revokeTenantApiKey(
  tenantId: string,
  apiKeyId: string,
): Promise<ApiKeyRecord> {
  const { data } = await api.delete<ApiKeyRecord>(`/tenants/${tenantId}/api-keys/${apiKeyId}`);
  return data;
}

/** List all API keys (admin view) */
export async function listAdminApiKeys(): Promise<ListTenantApiKeysResponse> {
  const { data } = await api.get<ListTenantApiKeysResponse>("/api-keys");
  return data;
}

// ============================================================================
// Model Functions
// ============================================================================

/** List models available to a tenant */
export async function listTenantModels(tenantId: string): Promise<string[]> {
  const { data } = await api.get<{ models: string[] }>(`/tenants/${tenantId}/models`);
  return data.models;
}

/** Update models available to a tenant */
export async function upsertTenantModels(tenantId: string, models: string[]): Promise<string[]> {
  const { data } = await api.put<{ models: string[] }>(`/tenants/${tenantId}/models`, { models });
  return data.models;
}

/** Remove all model restrictions from a tenant */
export async function clearTenantModels(tenantId: string): Promise<void> {
  await api.delete(`/tenants/${tenantId}/models`);
}

// ============================================================================
// Membership Functions
// ============================================================================

/** List memberships for a tenant */
export async function listTenantMemberships(tenantId: string): Promise<ListMembershipsResponse> {
  const memberships = await membershipsResource.list(tenantId);
  return { memberships };
}

/** Add or update a membership */
export async function upsertTenantMembership(
  tenantId: string,
  payload: UpsertMembershipRequest,
): Promise<MembershipRecord> {
  return membershipsResource.create(tenantId, payload);
}

/** Remove a membership */
export async function removeTenantMembership(tenantId: string, userId: string): Promise<void> {
  return membershipsResource.remove(tenantId, userId);
}

// ============================================================================
// Rate Limit Functions
// ============================================================================

/** Get rate limits for a tenant (returns null if not set) */
export async function getTenantRateLimits(
  tenantId: string,
): Promise<TenantRateLimitOverride | null> {
  const response = await api.get<TenantRateLimitOverride>(`/tenants/${tenantId}/rate-limits`, {
    validateStatus: (status) => (status >= 200 && status < 300) || status === 404,
  });
  if (response.status === 404) {
    return null;
  }
  return response.data;
}

/** Set rate limits for a tenant */
export async function upsertTenantRateLimits(
  tenantId: string,
  payload: TenantRateLimitOverride,
): Promise<TenantRateLimitOverride> {
  const { data } = await api.put<TenantRateLimitOverride>(
    `/tenants/${tenantId}/rate-limits`,
    payload,
  );
  return data;
}

/** Remove rate limits from a tenant */
export async function deleteTenantRateLimits(tenantId: string): Promise<void> {
  await api.delete(`/tenants/${tenantId}/rate-limits`);
}

// ============================================================================
// Exported Resources (for advanced usage)
// ============================================================================

export const adminTenantsResource = tenantsResource;
export const adminMembershipsResource = membershipsResource;
export const adminApiKeysResource = apiKeysResource;
