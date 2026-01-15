import { useQuery } from "@tanstack/react-query";
import type { UseQueryOptions } from "@tanstack/react-query";

import {
  listAuditLogs,
  getAuditLogStats,
  type ListAuditLogsParams,
  type ListAuditLogsResponse,
} from "../audit";

export function useAuditLogs(
  params?: ListAuditLogsParams,
  options?: Omit<UseQueryOptions<ListAuditLogsResponse>, "queryKey" | "queryFn">
) {
  return useQuery({
    queryKey: [
      "audit-logs",
      params?.limit ?? 50,
      params?.offset ?? 0,
      params?.user_id ?? null,
      params?.action ?? null,
      params?.resource ?? null,
    ],
    queryFn: () => listAuditLogs(params),
    staleTime: 30_000,
    ...options,
  });
}

export function useAuditLogStats(
  options?: Omit<
    UseQueryOptions<{
      total: number;
      auth_events: number;
      failed_events: number;
      unique_actors: number;
    }>,
    "queryKey" | "queryFn"
  >
) {
  return useQuery({
    queryKey: ["audit-log-stats"],
    queryFn: getAuditLogStats,
    staleTime: 60_000,
    ...options,
  });
}
