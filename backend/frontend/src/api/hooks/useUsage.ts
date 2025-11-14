import { useQuery } from "@tanstack/react-query";
import type { UseQueryOptions } from "@tanstack/react-query";

import {
  getModelDailyUsage,
  getTenantDailyUsage,
  getUsageBreakdown,
  getUsageComparison,
  getUsageOverview,
  getUserDailyUsage,
} from "../usage";
import type {
  ModelDailyUsageParams,
  ModelDailyUsageResponse,
  TenantDailyUsageParams,
  TenantDailyUsageResponse,
  UserDailyUsageParams,
  UserDailyUsageResponse,
  UsageBreakdownParams,
  UsageBreakdownResponse,
  UsageComparisonParams,
  UsageComparisonResponse,
  UsageOverview,
  UsageOverviewParams,
} from "../usage";

export function useUsageOverview(
  params: UsageOverviewParams = {},
  options?: Omit<UseQueryOptions<UsageOverview>, "queryKey" | "queryFn">,
) {
  const period = params.period ?? "7d";
  const tenantKey = params.tenantId ?? "all";
  const startKey = params.start ?? "start";
  const endKey = params.end ?? "end";
  return useQuery({
    queryKey: ["usage-overview", period, tenantKey, startKey, endKey],
    queryFn: () =>
      getUsageOverview({
        period,
        tenantId: params.tenantId,
        start: params.start,
        end: params.end,
      }),
    staleTime: 60_000,
    ...options,
  });
}

export function useUsageBreakdown(
  params: UsageBreakdownParams,
  options?: Omit<UseQueryOptions<UsageBreakdownResponse>, "queryKey" | "queryFn">,
) {
  const period = params.period ?? "30d";
  const entityId = params.entityId ?? "default";
  const limit = params.limit ?? 5;
  const rangeKey =
    params.start && params.end ? `${params.start}-${params.end}` : period;
  return useQuery({
    queryKey: ["usage-breakdown", params.group, rangeKey, entityId, limit],
    queryFn: () =>
      getUsageBreakdown({
        ...params,
        period,
        limit,
      }),
    ...options,
  });
}
export function useTenantUsageOverview(
  tenantId?: string,
  period: UsageOverviewParams["period"] = "30d",
  options?: UseQueryOptions<UsageOverview | null>,
) {
  return useQuery({
    queryKey: ["usage-overview", "tenant", tenantId ?? "none", period],
    queryFn: () =>
      tenantId
        ? getUsageOverview({ period, tenantId })
        : Promise.resolve(null),
    enabled: Boolean(tenantId),
    staleTime: 60_000,
    ...options,
  });
}

export function useUsageComparison(
  params: UsageComparisonParams,
  options?: Omit<UseQueryOptions<UsageComparisonResponse>, "queryKey" | "queryFn">,
) {
  const period = params.period ?? "30d";
  const tenantKey = (params.tenantIds ?? []).slice().sort().join(",");
  const modelKey = (params.modelAliases ?? []).slice().sort().join(",");
  const userKey = (params.userIds ?? []).slice().sort().join(",");
  const rangeKey = `${params.start ?? ""}-${params.end ?? ""}`;
  const enabled =
    (params.tenantIds?.length ?? 0) +
      (params.modelAliases?.length ?? 0) +
      (params.userIds?.length ?? 0) >
    0;
  return useQuery({
	queryKey: ["usage-comparison", period, tenantKey, modelKey, userKey, rangeKey],
	queryFn: () => getUsageComparison({ ...params, period }),
	enabled,
	...options,
  });
}

export function useTenantDailyUsage(
  params?: TenantDailyUsageParams,
  options?: UseQueryOptions<TenantDailyUsageResponse>,
) {
  const enabled = Boolean(params?.tenantId && params?.start && params?.end);
  const tenantId = params?.tenantId ?? "none";
  const start = params?.start ?? "start";
  const end = params?.end ?? "end";
  return useQuery({
    queryKey: ["tenant-daily-usage", tenantId, start, end],
    queryFn: () => {
      if (!params) {
        throw new Error("tenant daily usage params required");
      }
      return getTenantDailyUsage(params);
    },
    enabled,
    ...options,
  });
}

export function useUserDailyUsage(
  params?: UserDailyUsageParams,
  options?: UseQueryOptions<UserDailyUsageResponse>,
) {
  const enabled = Boolean(params?.userId && params?.start && params?.end);
  const userId = params?.userId ?? "none";
  const start = params?.start ?? "start";
  const end = params?.end ?? "end";
  return useQuery({
    queryKey: ["user-daily-usage", userId, start, end],
    queryFn: () => {
      if (!params) {
        throw new Error("user daily usage params required");
      }
      return getUserDailyUsage(params);
    },
    enabled,
    ...options,
  });
}

export function useModelDailyUsage(
  params?: ModelDailyUsageParams,
  options?: UseQueryOptions<ModelDailyUsageResponse>,
) {
  const enabled = Boolean(params?.modelAlias && params?.start && params?.end);
  const modelAlias = params?.modelAlias ?? "none";
  const start = params?.start ?? "start";
  const end = params?.end ?? "end";
  return useQuery({
    queryKey: ["model-daily-usage", modelAlias, start, end],
    queryFn: () => {
      if (!params) {
        throw new Error("model daily usage params required");
      }
      return getModelDailyUsage(params);
    },
    enabled,
    ...options,
  });
}
