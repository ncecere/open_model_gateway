import { api } from "./client";
import { getBrowserTimezone } from "@/lib/timezone";

export interface UsagePoint {
  date: string;
  requests: number;
  cost_cents: number;
  tokens: number;
  cost_usd?: number;
}

export type UsagePeriod = "7d" | "30d" | "90d";
export type UsageComparisonPeriod = UsagePeriod | "custom";

export interface UsageOverview {
  period: UsagePeriod;
  start: string;
  end: string;
  timezone: string;
  total_requests: number;
  total_cost_cents: number;
  total_cost_usd?: number;
  total_tokens: number;
  points: UsagePoint[];
}

export interface UsageOverviewParams {
  period?: UsagePeriod;
  tenantId?: string;
  start?: string;
  end?: string;
}

export async function getUsageOverview(
  params: UsageOverviewParams = {},
): Promise<UsageOverview> {
  const { period = "7d", tenantId, start, end } = params;
  const timezone = getBrowserTimezone();
  const query: Record<string, string> = {
    timezone,
  };
  if (tenantId) {
    query.tenant_id = tenantId;
  }
  if (start && end) {
    query.start = start;
    query.end = end;
  } else {
    query.period = period;
  }
  const { data } = await api.get<UsageOverview>("/usage/summary", {
    params: query,
  });
  return data;
}

export interface UsageBreakdownItem {
  id: string;
  label: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
}

export interface UsageBreakdownSeriesPoint {
  date: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
}

export interface UsageBreakdownSeries {
  id: string;
  label: string;
  points: UsageBreakdownSeriesPoint[];
  timezone?: string;
}

export interface UsageBreakdownResponse {
  group: "tenant" | "model" | "user";
  period: string;
  start: string;
  end: string;
  timezone: string;
  items: UsageBreakdownItem[];
  series: UsageBreakdownSeries | null;
}

export interface UsageBreakdownParams {
  group: "tenant" | "model" | "user";
  period?: UsagePeriod;
  limit?: number;
  entityId?: string;
  start?: string;
  end?: string;
}

export async function getUsageBreakdown(params: UsageBreakdownParams) {
  const { group, period = "30d", limit = 5, entityId, start, end } = params;
  const timezone = getBrowserTimezone();
  const query: Record<string, string> = {
    group,
    limit: limit.toString(),
    timezone,
  };
  if (entityId) {
    query.entity_id = entityId;
  }
  if (start && end) {
    query.start = start;
    query.end = end;
  } else {
    query.period = period;
  }
  const { data } = await api.get<UsageBreakdownResponse>("/usage/breakdown", {
    params: query,
  });
  return data;
}

export interface UsageComparisonSeries {
	kind: "tenant" | "model" | "user";
	id: string;
	label: string;
	tenant_id?: string | null;
	tenant_status?: string;
	tenant_kind?: string;
	user_id?: string | null;
	user_email?: string | null;
	user_name?: string | null;
	provider?: string;
  totals: {
    requests: number;
    tokens: number;
    cost_cents: number;
    cost_usd?: number;
  };
  points: UsagePoint[];
	active_start?: string | null;
	active_end?: string | null;
}

export interface UsageComparisonResponse {
  period: string;
  start: string;
  end: string;
  timezone: string;
  series: UsageComparisonSeries[];
}

export interface UsageComparisonParams {
	period?: UsageComparisonPeriod;
	tenantIds?: string[];
	modelAliases?: string[];
	userIds?: string[];
	start?: string;
	end?: string;
}

export async function getUsageComparison(
	params: UsageComparisonParams,
): Promise<UsageComparisonResponse> {
	const { period = "30d", tenantIds = [], modelAliases = [], userIds = [], start, end } = params;
	const timezone = getBrowserTimezone();
	const query: Record<string, string> = { period, timezone };
	if (tenantIds.length) {
		query.tenant_ids = tenantIds.join(",");
	}
	if (modelAliases.length) {
		query.model_aliases = modelAliases.join(",");
	}
	if (userIds.length) {
		query.user_ids = userIds.join(",");
	}
	if (start) {
		query.start = start;
	}
	if (end) {
		query.end = end;
	}
	const { data } = await api.get<UsageComparisonResponse>("/usage/compare", {
		params: query,
	});
	return data;
}

export interface TenantDailyUsageKey {
	api_key_id: string;
	api_key_name: string;
	api_key_prefix: string;
	requests: number;
	tokens: number;
	cost_cents: number;
	cost_usd?: number;
}

export interface TenantDailyUsageDay {
	date: string;
	requests: number;
	tokens: number;
	cost_cents: number;
	cost_usd?: number;
	keys: TenantDailyUsageKey[];
}

export interface TenantDailyUsageResponse {
	tenant_id: string;
	start: string;
	end: string;
	timezone: string;
	days: TenantDailyUsageDay[];
}

export interface TenantDailyUsageParams {
	tenantId: string;
	start: string;
	end: string;
}

export async function getTenantDailyUsage(params: TenantDailyUsageParams): Promise<TenantDailyUsageResponse> {
	const timezone = getBrowserTimezone();
	const { tenantId, start, end } = params;
	const { data } = await api.get<TenantDailyUsageResponse>("/usage/tenant/daily", {
		params: {
			tenant_id: tenantId,
			start,
			end,
			timezone,
		},
	});
	return data;
}

export interface UserDailyTenantUsage {
	tenant_id: string;
	tenant_name: string;
	requests: number;
	tokens: number;
	cost_cents: number;
	cost_usd?: number;
}

export interface UserDailyUsageDay {
	date: string;
	requests: number;
	tokens: number;
	cost_cents: number;
	cost_usd?: number;
	tenants: UserDailyTenantUsage[];
}

export interface UserDailyUsageResponse {
	user_id: string;
	start: string;
	end: string;
	timezone: string;
	days: UserDailyUsageDay[];
}

export interface UserDailyUsageParams {
	userId: string;
	start: string;
	end: string;
}

export async function getUserDailyUsage(params: UserDailyUsageParams): Promise<UserDailyUsageResponse> {
	const timezone = getBrowserTimezone();
	const { userId, start, end } = params;
	const { data } = await api.get<UserDailyUsageResponse>("/usage/user/daily", {
		params: {
			user_id: userId,
			start,
			end,
			timezone,
		},
	});
	return data;
}

export interface ModelDailyTenantUsage {
	tenant_id: string;
	tenant_name: string;
	requests: number;
	tokens: number;
	cost_cents: number;
	cost_usd?: number;
}

export interface ModelDailyUsageDay {
	date: string;
	requests: number;
	tokens: number;
	cost_cents: number;
	cost_usd?: number;
	tenants: ModelDailyTenantUsage[];
}

export interface ModelDailyUsageResponse {
	model_alias: string;
	start: string;
	end: string;
	timezone: string;
	days: ModelDailyUsageDay[];
}

export interface ModelDailyUsageParams {
	modelAlias: string;
	start: string;
	end: string;
}

export async function getModelDailyUsage(params: ModelDailyUsageParams): Promise<ModelDailyUsageResponse> {
	const timezone = getBrowserTimezone();
	const { modelAlias, start, end } = params;
	const { data } = await api.get<ModelDailyUsageResponse>("/usage/model/daily", {
		params: {
			model_alias: modelAlias,
			start,
			end,
			timezone,
		},
	});
	return data;
}
