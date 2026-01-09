/**
 * Shared types used across admin and user API layers.
 * These types represent common data structures returned by the backend.
 */

// ============================================================================
// Pagination
// ============================================================================

export interface PaginationParams {
  limit?: number;
  offset?: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  limit: number;
  offset: number;
  total?: number;
}

// ============================================================================
// Rate Limits
// ============================================================================

export interface RateLimitInfo {
  requests_per_minute?: number;
  tokens_per_minute?: number;
  parallel_requests?: number;
}

export interface RateLimitOverride {
  requests_per_minute: number;
  tokens_per_minute: number;
  parallel_requests: number;
}

export interface ApiKeyRateLimits {
  key: RateLimitInfo;
  tenant: RateLimitInfo;
}

// ============================================================================
// Budgets & Quotas
// ============================================================================

export interface QuotaPayload {
  budget_usd?: number;
  budget_cents?: number;
  warning_threshold?: number;
}

export interface BudgetSummary {
  limit_usd: number;
  used_usd: number;
  remaining_usd: number;
  warning_threshold: number;
  refresh_schedule: string;
}

// ============================================================================
// Tenants
// ============================================================================

export type TenantStatus = "active" | "suspended" | "pending";

export interface BaseTenantRecord {
  id: string;
  name: string;
  status: TenantStatus;
  created_at: string;
}

export interface TenantBudgetInfo {
  budget_limit_usd?: number | null;
  budget_used_usd?: number | null;
  warning_threshold?: number | null;
}

// ============================================================================
// Memberships
// ============================================================================

export type MembershipRole = "owner" | "admin" | "viewer" | "user";

export interface BaseMembershipRecord {
  tenant_id: string;
  user_id: string;
  email: string;
  role: MembershipRole;
  created_at: string;
}

export interface MemberBudget {
  budget_usd?: number;
  warning_threshold?: number;
  token_cap?: number;
}

// ============================================================================
// API Keys
// ============================================================================

export interface ApiKeyIssuer {
  type: string;
  label: string;
}

export interface BaseApiKeyRecord {
  id: string;
  tenant_id: string;
  prefix: string;
  name: string;
  scopes: string[];
  quota?: QuotaPayload | null;
  budget_refresh_schedule?: string;
  rate_limits?: ApiKeyRateLimits;
  created_at: string;
  revoked_at?: string | null;
  last_used_at?: string | null;
  revoked: boolean;
}

export interface CreateApiKeyPayload {
  name: string;
  scopes?: string[];
  quota?: QuotaPayload;
  rate_limits?: RateLimitInfo;
}

export interface CreateApiKeyResult<T extends BaseApiKeyRecord> {
  api_key: T;
  secret: string;
  token: string;
}

// ============================================================================
// Usage
// ============================================================================

export interface UsageTotals {
  requests: number;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cost_usd: number;
}

export interface UsagePoint {
  timestamp: string;
  requests: number;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cost_usd: number;
}

export interface UsageSummary {
  period: string;
  start: string;
  end: string;
  timezone: string;
  totals: UsageTotals;
  series: UsagePoint[];
}

// ============================================================================
// Models
// ============================================================================

export interface ModelEntry {
  alias: string;
  provider: string;
  model_type: string;
  enabled: boolean;
}

export interface TenantModelEntry extends ModelEntry {
  tenant_assignable: boolean;
  attached: boolean;
}

// ============================================================================
// Personal Tenants (Admin)
// ============================================================================

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

// ============================================================================
// Batches
// ============================================================================

export interface BatchCounts {
  total: number;
  completed: number;
  failed: number;
  cancelled: number;
}

export interface BatchErrorEntry {
  code: string;
  message: string;
  param?: string;
  line?: number | null;
}

export interface BatchErrorList {
  object: string;
  data: BatchErrorEntry[];
}

export interface BaseBatchRecord {
  id: string;
  tenant_id: string;
  endpoint: string;
  status: string;
  completion_window: string;
  max_concurrency: number;
  metadata?: Record<string, string>;
  input_file_id: string;
  output_file_id?: string | null;
  error_file_id?: string | null;
  created_at: string;
  updated_at: string;
  in_progress_at?: string | null;
  completed_at?: string | null;
  cancelled_at?: string | null;
  cancelling_at?: string | null;
  finalizing_at?: string | null;
  failed_at?: string | null;
  expires_at?: string | null;
  expired_at?: string | null;
  counts: BatchCounts;
  errors?: BatchErrorList;
}

export interface AdminBatchRecord extends BaseBatchRecord {
  api_key_id: string;
  tenant_name?: string;
}

export interface CursorPaginationParams {
  limit?: number;
  after?: string;
}

export interface CursorPaginatedResponse<T> {
  object: string;
  data: T[];
  has_more: boolean;
  first_id?: string;
  last_id?: string;
}

// ============================================================================
// Files
// ============================================================================

export interface BaseFileRecord {
  id: string;
  tenant_id: string;
  filename: string;
  purpose: string;
  content_type: string;
  bytes: number;
  created_at: string;
  expires_at: string;
  deleted_at?: string | null;
  status: string;
  status_details?: string | null;
}

export interface AdminFileRecord extends BaseFileRecord {
  tenant_name: string;
  storage_backend: string;
  encrypted: boolean;
  checksum: string;
}

// ============================================================================
// Admin Keys
// ============================================================================

export type AdminKeyScope = "admin" | "system";

export interface AdminKeyRecord {
  id: string;
  name: string;
  scope: AdminKeyScope;
  prefix: string;
  owner_user_id?: string | null;
  owner_name?: string;
  owner_email?: string;
  created_by_user_id: string;
  creator_name?: string;
  creator_email?: string;
  expires_at?: string | null;
  revoked_at?: string | null;
  last_used_at?: string | null;
  created_at: string;
}

export interface CreateAdminKeyPayload {
  name: string;
  scope: AdminKeyScope;
  expires_in_seconds: number;
}

export interface CreateAdminKeyResult {
  admin_key: AdminKeyRecord;
  token: string;
}

// ============================================================================
// User-specific Types
// ============================================================================

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

export interface UserMembershipRecord extends BaseMembershipRecord {
  budget?: MemberBudget;
  self?: boolean;
}

export interface TenantLimits {
  requests_per_minute: number;
  tokens_per_minute: number;
  parallel_requests: number;
}

export interface TenantLimitsPayload {
  effective: TenantLimits;
  ceiling: TenantLimits;
  overridden: boolean;
}

// ============================================================================
// User Usage Types
// ============================================================================

export interface UserUsagePoint {
  date: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
}

export interface UserUsageTotals {
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
}

export interface UserTenantUsage {
  tenant_id: string;
  name: string;
  role: string;
  status: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
  is_personal: boolean;
}

export interface UserApiKeyUsage {
  api_key_id: string;
  name: string;
  prefix: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
  last_used_at?: string | null;
}

export interface RecentRequest {
  id: string;
  api_key_id: string;
  api_key_name: string;
  model_alias: string;
  provider: string;
  status: number;
  latency_ms: number;
  cost_cents: number;
  cost_usd?: number;
  timestamp: string;
  error_code?: string | null;
}

export interface UsageScope {
  id: string;
  kind: "personal" | "tenant";
  tenant_id?: string | null;
  name: string;
  role?: string;
  status?: string;
  totals: UserUsageTotals;
}

export interface UsageScopeDetail {
  scope: UsageScope;
  series: UserUsagePoint[];
  api_keys: UserApiKeyUsage[];
  recent_requests: RecentRequest[];
}

export interface UserUsageSummary {
  period: string;
  start: string;
  end: string;
  timezone: string;
  totals: UserUsageTotals;
  personal?: UserTenantUsage;
  personal_series?: UserUsagePoint[];
  memberships: UserTenantUsage[];
  personal_api_keys?: UserApiKeyUsage[];
  recent_requests?: RecentRequest[];
  scopes?: UsageScope[];
  selected_scope?: UsageScopeDetail;
}

export interface ApiKeyUsageSummary {
  api_key_id: string;
  period: string;
  start: string;
  end: string;
  timezone: string;
  totals: UserUsageTotals;
  series: UserUsagePoint[];
}
