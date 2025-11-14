import { userApi } from "../userClient";
import { getBrowserTimezone } from "@/lib/timezone";
import type {
  ModelDailyUsageResponse,
  TenantDailyUsageResponse,
  UsageComparisonResponse,
} from "../usage";

export type UsagePoint = {
  date: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
};

export type UsageTotals = {
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
};

export type UserTenantUsage = {
  tenant_id: string;
  name: string;
  role: string;
  status: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
  is_personal: boolean;
};

export type UserAPIKeyUsage = {
  api_key_id: string;
  name: string;
  prefix: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
  last_used_at?: string | null;
};

export type RecentRequest = {
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
};

export type UsageScope = {
  id: string;
  kind: "personal" | "tenant";
  tenant_id?: string | null;
  name: string;
  role?: string;
  status?: string;
  totals: UsageTotals;
};

export type UsageScopeDetail = {
  scope: UsageScope;
  series: UsagePoint[];
  api_keys: UserAPIKeyUsage[];
  recent_requests: RecentRequest[];
};

export type UserUsageSummary = {
  period: string;
  start: string;
  end: string;
  timezone: string;
  totals: UsageTotals;
  personal?: UserTenantUsage;
  personal_series?: UsagePoint[];
  memberships: UserTenantUsage[];
  personal_api_keys?: UserAPIKeyUsage[];
  recent_requests?: RecentRequest[];
  scopes?: UsageScope[];
  selected_scope?: UsageScopeDetail;
};

export async function getUserDashboard(period: string = "7d", scope?: string) {
  const params: Record<string, string> = { period, timezone: getBrowserTimezone() };
  if (scope) {
    params.scope = scope;
  }
  const { data } = await userApi.get<UserUsageSummary>("/dashboard", {
    params,
  });
  return data;
}

export interface UserUsageParams {
  period?: string;
  scope?: string;
  start?: string;
  end?: string;
}

export async function getUserUsage(params: UserUsageParams = {}) {
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
  const { data } = await userApi.get<UserUsageSummary>("/usage", {
    params: query,
  });
  return data;
}

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
	const { data } = await userApi.get<UsageComparisonResponse>("/usage/compare", {
		params,
	});
	return data;
}

export interface UserTenantDailyUsageParams {
	tenantId: string;
	start: string;
	end: string;
}

export async function getUserTenantDailyUsage(params: UserTenantDailyUsageParams) {
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

export interface UserModelDailyUsageParams {
	modelAlias: string;
	start: string;
	end: string;
}

export async function getUserModelDailyUsage(params: UserModelDailyUsageParams) {
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
