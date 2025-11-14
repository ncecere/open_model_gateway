import { api } from "./client";

export type TenantStatus = "active" | "suspended" | "pending";

export interface TenantRecord {
  id: string;
  name: string;
  status: TenantStatus;
  created_at: string;
  budget_limit_usd?: number | null;
  budget_used_usd?: number | null;
  warning_threshold?: number | null;
}

export interface ListTenantsResponse {
  tenants: TenantRecord[];
  limit: number;
  offset: number;
}

export interface ListTenantsParams {
  limit?: number;
  offset?: number;
}

export async function listTenants(params?: ListTenantsParams) {
  const { data } = await api.get<ListTenantsResponse>("/tenants", { params });
  return data;
}

export interface PersonalTenantRecord {
  tenant_id: string;
  user_id: string;
  user_email: string;
  user_name: string;
  status: TenantStatus;
  created_at: string;
  budget_limit_usd?: number | null;
  budget_used_usd?: number | null;
  warning_threshold?: number | null;
  membership_count?: number;
}

export interface ListPersonalTenantsResponse {
  personal_tenants: PersonalTenantRecord[];
  limit: number;
  offset: number;
}

export async function listPersonalTenants(params?: ListTenantsParams) {
  const { data } = await api.get<ListPersonalTenantsResponse>(
    "/tenants/personal",
    { params },
  );
  return data;
}

export interface CreateTenantRequest {
  name: string;
  status?: TenantStatus;
}

export async function createTenant(payload: CreateTenantRequest) {
  const { data } = await api.post<TenantRecord>("/tenants", payload);
  return data;
}

export interface UpdateTenantRequest {
  name: string;
}

export async function updateTenant(tenantId: string, payload: UpdateTenantRequest) {
  const { data } = await api.patch<TenantRecord>(`/tenants/${tenantId}`, payload);
  return data;
}

export interface UpdateTenantStatusRequest {
  tenantId: string;
  status: TenantStatus;
}

export async function updateTenantStatus({
  tenantId,
  status,
}: UpdateTenantStatusRequest) {
  const { data } = await api.patch<TenantRecord>(
    `/tenants/${tenantId}/status`,
    { status },
  );
  return data;
}

export interface QuotaPayload {
  budget_usd?: number;
  warning_threshold?: number;
}

export interface RateLimitInfo {
  requests_per_minute?: number;
  tokens_per_minute?: number;
  parallel_requests?: number;
}

export interface ApiKeyRateLimits {
  key: RateLimitInfo;
  tenant: RateLimitInfo;
}

export interface ApiKeyIssuer {
  type: string;
  label: string;
}

export interface ApiKeyRecord {
  id: string;
  tenant_id: string;
  tenant_name?: string;
  prefix: string;
  name: string;
  scopes: string[];
  issuer?: ApiKeyIssuer;
  quota?: QuotaPayload | null;
  budget_refresh_schedule?: string;
  rate_limits?: ApiKeyRateLimits;
  created_at: string;
  revoked_at?: string | null;
  last_used_at?: string | null;
  revoked: boolean;
}

export interface ListTenantApiKeysResponse {
  api_keys: ApiKeyRecord[];
}

export async function listTenantApiKeys(tenantId: string) {
  const { data } = await api.get<ListTenantApiKeysResponse>(
    `/tenants/${tenantId}/api-keys`,
  );
  return data;
}

export interface CreateApiKeyRequest {
  name: string;
  scopes?: string[];
  quota?: QuotaPayload;
}

export interface CreateApiKeyResponse {
  api_key: ApiKeyRecord;
  secret: string;
  token: string;
}

export async function createTenantApiKey(
  tenantId: string,
  payload: CreateApiKeyRequest,
) {
  const { data } = await api.post<CreateApiKeyResponse>(
    `/tenants/${tenantId}/api-keys`,
    payload,
  );
  return data;
}

export async function revokeTenantApiKey(tenantId: string, apiKeyId: string) {
  const { data } = await api.delete<ApiKeyRecord>(
    `/tenants/${tenantId}/api-keys/${apiKeyId}`,
  );
  return data;
}

export async function listAdminApiKeys() {
  const { data } = await api.get<ListTenantApiKeysResponse>("/api-keys");
  return data;
}

export async function listTenantModels(tenantId: string) {
  const { data } = await api.get<{ models: string[] }>(
    `/tenants/${tenantId}/models`,
  );
  return data.models;
}

export async function upsertTenantModels(tenantId: string, models: string[]) {
  const { data } = await api.put<{ models: string[] }>(
    `/tenants/${tenantId}/models`,
    { models },
  );
  return data.models;
}

export async function clearTenantModels(tenantId: string) {
  await api.delete(`/tenants/${tenantId}/models`);
}

export type MembershipRole = "owner" | "admin" | "viewer" | "user";

export interface MembershipRecord {
  tenant_id: string;
  user_id: string;
  email: string;
  role: MembershipRole;
  created_at: string;
}

export interface ListMembershipsResponse {
  memberships: MembershipRecord[];
}

export async function listTenantMemberships(tenantId: string) {
  const { data } = await api.get<ListMembershipsResponse>(
    `/tenants/${tenantId}/memberships`,
  );
  return data;
}

export interface UpsertMembershipRequest {
  email: string;
  role: MembershipRole;
  password?: string;
}

export async function upsertTenantMembership(
  tenantId: string,
  payload: UpsertMembershipRequest,
) {
  const { data } = await api.post<MembershipRecord>(
    `/tenants/${tenantId}/memberships`,
    payload,
  );
  return data;
}

export async function removeTenantMembership(tenantId: string, userId: string) {
  await api.delete(`/tenants/${tenantId}/memberships/${userId}`);
}
