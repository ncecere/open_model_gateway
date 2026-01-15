import { useQuery } from "@tanstack/react-query";
import type { UseQueryOptions } from "@tanstack/react-query";

import {
  fetchSystemStatus,
  fetchHealthStatus,
  type SystemStatusResponse,
  type HealthResponse,
} from "../system-status";

export function useSystemStatus(
  options?: Omit<UseQueryOptions<SystemStatusResponse>, "queryKey" | "queryFn">
) {
  return useQuery({
    queryKey: ["system-status"],
    queryFn: fetchSystemStatus,
    staleTime: 15_000,
    refetchInterval: 30_000, // Auto-refresh every 30 seconds
    ...options,
  });
}

export function useHealthStatus(
  options?: Omit<UseQueryOptions<HealthResponse>, "queryKey" | "queryFn">
) {
  return useQuery({
    queryKey: ["health-status"],
    queryFn: fetchHealthStatus,
    staleTime: 10_000,
    ...options,
  });
}
