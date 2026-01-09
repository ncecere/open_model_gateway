/**
 * User Usage API
 *
 * This module provides user-level access to usage data and analytics.
 * Uses shared types from api/shared/types.ts.
 */

import { userApi } from "../userClient";
import { getBrowserTimezone } from "@/lib/timezone";
import type {
  UserUsagePoint,
  UserUsageTotals,
  UserTenantUsage,
  UserApiKeyUsage,
  RecentRequest,
  UsageScope,
  UsageScopeDetail,
  UserUsageSummary,
} from "../shared/types";
import type {
  UsageComparisonResponse,
  TenantDailyUsageResponse,
  ModelDailyUsageResponse,
} from "../usage";

// ============================================================================
// Types
// ============================================================================

// Re-export shared types with local aliases for backward compatibility
export type UsagePoint = UserUsagePoint;
export type UsageTotals = UserUsageTotals;
export type {
  UserTenantUsage,
  UserApiKeyUsage as UserAPIKeyUsage,
  RecentRequest,
  UsageScope,
  UsageScopeDetail,
  UserUsageSummary,
};

/** Parameters for getting user usage */
export interface UserUsageParams {
  period?: string;
  scope?: string;
  start?: string;
  end?: string;
}

/** Parameters for getting user tenant daily usage */
export interface UserTenantDailyUsageParams {
  tenantId: string;
  start: string;
  end: string;
}

/** Parameters for getting user model daily usage */
export interface UserModelDailyUsageParams {
  modelAlias: string;
  start: string;
  end: string;
}

// ============================================================================
// Dashboard Functions
// ============================================================================

/** Get user dashboard data */
export async function getUserDashboard(
  period: string = "7d",
  scope?: string,
): Promise<UserUsageSummary> {
  const params: Record<string, string> = {
    period,
    timezone: getBrowserTimezone(),
  };
  if (scope) {
    params.scope = scope;
  }
  const { data } = await userApi.get<UserUsageSummary>("/dashboard", { params });
  return data;
}

// ============================================================================
// Usage Functions
// ============================================================================

/** Get user usage summary */
export async function getUserUsage(params: UserUsageParams = {}): Promise<UserUsageSummary> {
  const { period = "30d", scope, start, end } = params;
  const query: Record<string, string> = { timezone: getBrowserTimezone() };

  if (scope) {
    query.scope = scope;
  }
  if (start && end) {
    query.start = start;
    query.end = end;
  } else {
    query.period = period;
  }

  const { data } = await userApi.get<UserUsageSummary>("/usage", { params: query });
  return data;
}

/** Get user usage comparison */
export async function getUserUsageComparison(
  period: string,
  tenantIds: string[],
  modelAliases: string[] = [],
  options?: { start?: string; end?: string },
): Promise<UsageComparisonResponse> {
  const timezone = getBrowserTimezone();
  const params: Record<string, string> = {
    period,
    timezone,
  };

  if (tenantIds.length) {
    params.tenant_ids = tenantIds.join(",");
  }
  if (modelAliases.length) {
    params.model_aliases = modelAliases.join(",");
  }
  if (options?.start) {
    params.start = options.start;
  }
  if (options?.end) {
    params.end = options.end;
  }

  const { data } = await userApi.get<UsageComparisonResponse>("/usage/compare", { params });
  return data;
}

/** Get user tenant daily usage */
export async function getUserTenantDailyUsage(
  params: UserTenantDailyUsageParams,
): Promise<TenantDailyUsageResponse> {
  const timezone = getBrowserTimezone();
  const { tenantId, start, end } = params;
  const { data } = await userApi.get<TenantDailyUsageResponse>("/usage/tenant/daily", {
    params: {
      tenant_id: tenantId,
      start,
      end,
      timezone,
    },
  });
  return data;
}

/** Get user model daily usage */
export async function getUserModelDailyUsage(
  params: UserModelDailyUsageParams,
): Promise<ModelDailyUsageResponse> {
  const timezone = getBrowserTimezone();
  const { modelAlias, start, end } = params;
  const { data } = await userApi.get<ModelDailyUsageResponse>("/usage/model/daily", {
    params: {
      model_alias: modelAlias,
      start,
      end,
      timezone,
    },
  });
  return data;
}
