/**
 * User API Keys API
 *
 * This module provides user-level access to API key management.
 * Uses the factory pattern from api/shared/factory.ts.
 */

import { userApi } from "../userClient";
import { createResource, createNestedResource } from "../shared/factory";
import { getBrowserTimezone } from "@/lib/timezone";
import type {
  BaseApiKeyRecord,
  RateLimitInfo,
  ApiKeyRateLimits,
  QuotaPayload,
  ApiKeyUsageSummary,
  UserUsagePoint,
  UserUsageTotals,
} from "../shared/types";

// ============================================================================
// Types
// ============================================================================

/** User API key record */
export interface UserAPIKey extends BaseApiKeyRecord {}

/** Response for listing user API keys */
export type ListUserAPIKeysResponse = {
  api_keys: UserAPIKey[];
};

/** Payload for creating a user API key */
export type CreateUserAPIKeyRequest = {
  name: string;
  scopes?: string[];
  quota?: QuotaPayload;
  rate_limits?: RateLimitInfo;
};

/** Response when creating a user API key */
export type CreateUserAPIKeyResponse = {
  api_key: UserAPIKey;
  secret: string;
  token: string;
};

/** Response when listing tenant API keys */
export type TenantAPIKeyListResponse = {
  role: string;
  api_keys: UserAPIKey[];
};

// Re-export shared types for convenience
export type { RateLimitInfo, ApiKeyRateLimits, ApiKeyUsageSummary };

// Re-export usage types from shared
export type UsagePoint = UserUsagePoint;
export type UsageTotals = UserUsageTotals;

// ============================================================================
// Factory-based Resources
// ============================================================================

/** User API keys resource using factory pattern */
const apiKeysResource = createResource<UserAPIKey, CreateUserAPIKeyRequest>({
  basePath: "/api-keys",
  client: userApi,
  transformListResponse: (data) => {
    const response = data as { api_keys: UserAPIKey[] };
    return response.api_keys;
  },
});

/** User tenant API keys nested resource */
const tenantApiKeysResource = createNestedResource<UserAPIKey, CreateUserAPIKeyRequest>({
  parentPath: "/tenants",
  childPath: "api-keys",
  client: userApi,
  transformListResponse: (data) => {
    const response = data as { api_keys: UserAPIKey[] };
    return response.api_keys;
  },
});

// ============================================================================
// Personal API Key Functions
// ============================================================================

/** List all API keys for the current user */
export async function listUserAPIKeys(): Promise<UserAPIKey[]> {
  return apiKeysResource.list();
}

/** Create a personal API key */
export async function createUserAPIKey(
  payload: CreateUserAPIKeyRequest,
): Promise<CreateUserAPIKeyResponse> {
  const { data } = await userApi.post<CreateUserAPIKeyResponse>("/api-keys", payload);
  return data;
}

/** Revoke a personal API key */
export async function revokeUserAPIKey(apiKeyId: string): Promise<UserAPIKey> {
  const { data } = await userApi.post<UserAPIKey>(`/api-keys/${apiKeyId}/revoke`);
  return data;
}

/** Rotate a personal API key */
export async function rotateUserAPIKey(apiKeyId: string): Promise<CreateUserAPIKeyResponse> {
  const { data } = await userApi.post<CreateUserAPIKeyResponse>(`/api-keys/${apiKeyId}/rotate`);
  return data;
}

/** Get usage for a personal API key */
export async function getUserAPIKeyUsage(
  apiKeyId: string,
  period = "30d",
): Promise<ApiKeyUsageSummary> {
  const timezone = getBrowserTimezone();
  const { data } = await userApi.get<ApiKeyUsageSummary>(`/api-keys/${apiKeyId}/usage`, {
    params: { period, timezone },
  });
  return data;
}

// ============================================================================
// Tenant API Key Functions
// ============================================================================

/** List API keys for a tenant */
export async function listTenantAPIKeys(tenantId: string): Promise<TenantAPIKeyListResponse> {
  const { data } = await userApi.get<TenantAPIKeyListResponse>(
    `/tenants/${tenantId}/api-keys`,
  );
  return data;
}

/** Create an API key for a tenant */
export async function createTenantAPIKey(
  tenantId: string,
  payload: CreateUserAPIKeyRequest,
): Promise<CreateUserAPIKeyResponse> {
  const { data } = await userApi.post<CreateUserAPIKeyResponse>(
    `/tenants/${tenantId}/api-keys`,
    payload,
  );
  return data;
}

/** Revoke a tenant API key */
export async function revokeTenantAPIKey(
  tenantId: string,
  apiKeyId: string,
): Promise<UserAPIKey> {
  const { data } = await userApi.post<UserAPIKey>(
    `/tenants/${tenantId}/api-keys/${apiKeyId}/revoke`,
  );
  return data;
}

/** Rotate a tenant API key */
export async function rotateTenantAPIKey(
  tenantId: string,
  apiKeyId: string,
): Promise<CreateUserAPIKeyResponse> {
  const { data } = await userApi.post<CreateUserAPIKeyResponse>(
    `/tenants/${tenantId}/api-keys/${apiKeyId}/rotate`,
  );
  return data;
}

/** Get usage for a tenant API key */
export async function getTenantAPIKeyUsage(
  tenantId: string,
  apiKeyId: string,
  period = "30d",
): Promise<ApiKeyUsageSummary> {
  const timezone = getBrowserTimezone();
  const { data } = await userApi.get<ApiKeyUsageSummary>(
    `/tenants/${tenantId}/api-keys/${apiKeyId}/usage`,
    { params: { period, timezone } },
  );
  return data;
}

// ============================================================================
// Exported Resources (for advanced usage)
// ============================================================================

export const userApiKeysResource = apiKeysResource;
export const userTenantApiKeysResource = tenantApiKeysResource;
