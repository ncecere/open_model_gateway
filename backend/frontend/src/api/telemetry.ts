import { api } from "./client";

export interface ProviderSLIThresholds {
  latency_p95_ms: number;
  error_rate_threshold: number;
  timeout_rate_threshold: number;
  min_samples: number;
}

export interface ProviderSLI {
  provider: string;
  alias: string;
  route?: string;
  window_seconds: number;
  window_start: string;
  window_end: string;
  request_count: number;
  error_count: number;
  timeout_count: number;
  latency_p95_ms: number;
  error_rate: number;
  timeout_rate: number;
  thresholds: ProviderSLIThresholds;
}

export interface ListSLIParams {
  provider?: string;
  alias?: string;
  seed?: boolean;
}

export async function listProviderSLIs(params?: ListSLIParams) {
  const { data } = await api.get<{
    provider?: string;
    alias?: string;
    slis: ProviderSLI[];
    count: number;
  }>("/providers/slis", { params });
  return data.slis;
}

export interface ProviderIncident {
  id: string;
  provider: string;
  alias: string;
  type: string;
  status: string;
  window: number;
  opened_at: string;
  resolved_at?: string | null;
  window_start: string;
  window_end: string;
  request_count: number;
  error_count: number;
  timeout_count: number;
  latency_p95: number;
  error_rate: number;
  timeout_rate: number;
  thresholds: ProviderSLIThresholds;
  metadata?: Record<string, unknown>;
  sample_error?: string;
}

export interface ListIncidentsParams {
  provider?: string;
  alias?: string;
  status?: string;
  limit?: number;
  offset?: number;
  seed?: boolean;
}

export async function listProviderIncidents(params?: ListIncidentsParams) {
  const { data } = await api.get<{
    incidents: ProviderIncident[];
    total: number;
    total_pages: number;
  }>("/providers/incidents", {
    params,
  });
  return data.incidents;
}

export interface ListAlertsParams {
  provider?: string;
  alias?: string;
  limit?: number;
  offset?: number;
}

export async function listProviderAlerts(params?: ListAlertsParams) {
  const { data } = await api.get<{
    alerts: ProviderIncident[];
    count: number;
  }>("/providers/alerts", {
    params,
  });
  return data.alerts;
}

export async function clearTelemetrySeed() {
  await api.post("/providers/seed/clear");
}
