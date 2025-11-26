import { useQuery } from "@tanstack/react-query";
import type { UseQueryOptions } from "@tanstack/react-query";

import {
  listProviderAlerts,
  listProviderIncidents,
  listProviderSLIs,
  type ListAlertsParams,
  type ListIncidentsParams,
  type ListSLIParams,
  type ProviderIncident,
  type ProviderSLI,
} from "../telemetry";

export function useProviderSLIs(
  params?: ListSLIParams,
  options?: Omit<UseQueryOptions<ProviderSLI[]>, "queryKey" | "queryFn">,
) {
  return useQuery({
    queryKey: [
      "provider-slis",
      params?.provider ?? "all",
      params?.alias ?? "all",
      params?.seed ?? false,
    ],
    queryFn: () => listProviderSLIs(params),
    staleTime: 30_000,
    ...options,
  });
}

export function useProviderIncidents(
  params?: ListIncidentsParams,
  options?: Omit<UseQueryOptions<ProviderIncident[]>, "queryKey" | "queryFn">,
) {
  return useQuery({
    queryKey: [
      "provider-incidents",
      params?.provider ?? "all",
      params?.alias ?? "all",
      params?.status ?? "any",
      params?.seed ?? false,
    ],
    queryFn: () => listProviderIncidents(params),
    staleTime: 30_000,
    ...options,
  });
}

export function useProviderAlerts(
  params?: ListAlertsParams,
  options?: Omit<UseQueryOptions<ProviderIncident[]>, "queryKey" | "queryFn">,
) {
  return useQuery({
    queryKey: [
      "provider-alerts",
      params?.provider ?? "all",
      params?.alias ?? "all",
    ],
    queryFn: () => listProviderAlerts(params),
    staleTime: 30_000,
    ...options,
  });
}
