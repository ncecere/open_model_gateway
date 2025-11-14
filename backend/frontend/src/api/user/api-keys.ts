import { userApi } from "../userClient";
import type { UsagePoint, UsageTotals } from "./usage";
import { getBrowserTimezone } from "@/lib/timezone";

export type RateLimitInfo = {
  requests_per_minute?: number;
  tokens_per_minute?: number;
  parallel_requests?: number;
};

export type ApiKeyRateLimits = {
  key: RateLimitInfo;
  tenant: RateLimitInfo;
};

export type UserAPIKey = {
  id: string;
  tenant_id: string;
  prefix: string;
  name: string;
  scopes: string[];
  quota?: {
    budget_usd?: number;
    budget_cents?: number;
    warning_threshold?: number;
  } | null;
  budget_refresh_schedule?: string;
  rate_limits?: ApiKeyRateLimits;
  created_at: string;
  revoked_at?: string | null;
  last_used_at?: string | null;
  revoked: boolean;
};

export type ListUserAPIKeysResponse = {
  api_keys: UserAPIKey[];
};

export async function listUserAPIKeys() {
  const { data } = await userApi.get<ListUserAPIKeysResponse>("/api-keys");
  return data.api_keys;
}

export type CreateUserAPIKeyRequest = {
  name: string;
  scopes?: string[];
  quota?: {
    budget_usd?: number;
    warning_threshold?: number;
  };
};

export type CreateUserAPIKeyResponse = {
  api_key: UserAPIKey;
  secret: string;
  token: string;
};

export async function createUserAPIKey(payload: CreateUserAPIKeyRequest) {
  const { data } = await userApi.post<CreateUserAPIKeyResponse>(
    "/api-keys",
    payload,
  );
  return data;
}

export async function revokeUserAPIKey(apiKeyId: string) {
  const { data } = await userApi.post<UserAPIKey>(
    `/api-keys/${apiKeyId}/revoke`,
  );
  return data;
}

export type APIKeyUsageSummary = {
  api_key_id: string;
  period: string;
  start: string;
  end: string;
  timezone: string;
  totals: UsageTotals;
  series: UsagePoint[];
};

export async function getUserAPIKeyUsage(apiKeyId: string, period = "30d") {
  const timezone = getBrowserTimezone();
  const { data } = await userApi.get<APIKeyUsageSummary>(
    `/api-keys/${apiKeyId}/usage`,
    {
      params: { period, timezone },
    },
  );
  return data;
}

export type TenantAPIKeyListResponse = {
  role: string;
  api_keys: UserAPIKey[];
};

export async function listTenantAPIKeys(tenantId: string) {
  const { data } = await userApi.get<TenantAPIKeyListResponse>(
    `/tenants/${tenantId}/api-keys`,
  );
  return data;
}

export async function createTenantAPIKey(
  tenantId: string,
  payload: CreateUserAPIKeyRequest,
) {
  const { data } = await userApi.post<CreateUserAPIKeyResponse>(
    `/tenants/${tenantId}/api-keys`,
    payload,
  );
  return data;
}

export async function revokeTenantAPIKey(tenantId: string, apiKeyId: string) {
  const { data } = await userApi.post<UserAPIKey>(
    `/tenants/${tenantId}/api-keys/${apiKeyId}/revoke`,
  );
  return data;
}

export async function getTenantAPIKeyUsage(
  tenantId: string,
  apiKeyId: string,
  period = "30d",
) {
  const timezone = getBrowserTimezone();
  const { data } = await userApi.get<APIKeyUsageSummary>(
    `/tenants/${tenantId}/api-keys/${apiKeyId}/usage`,
    { params: { period, timezone } },
  );
  return data;
}
