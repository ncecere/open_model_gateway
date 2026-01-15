import axios from "axios";
import { api } from "./client";
import { listProviderSLIs, type ProviderSLI } from "./telemetry";

// Health check response from /healthz
export interface HealthCheck {
  status: string;
  latency_ms?: number;
  error?: string;
}

export interface HealthResponse {
  status: "ok" | "degraded" | "error";
  checks?: Record<string, HealthCheck>;
}

// Backend system metrics response
export interface BackendSystemMetrics {
  infrastructure: {
    database: {
      status: string;
      total_connections: number;
      idle_connections: number;
      acquired_connections: number;
      max_connections: number;
      latency_ms: number;
      connection_usage_percent: number;
    };
    redis: {
      status: string;
      latency_ms: number;
    };
  };
  requests: {
    total_24h: number;
    total_1h: number;
    error_count_24h: number;
    error_rate_24h: number;
    avg_latency_ms: number;
    p95_latency_ms: number;
    requests_per_minute: number;
  };
  tokens: {
    total_24h: number;
    input_24h: number;
    output_24h: number;
    tokens_per_minute: number;
    avg_tokens_per_request: number;
  };
  providers: {
    total_providers: number;
    healthy_providers: number;
    degraded_routes: number;
    active_incidents: number;
  };
  runtime: {
    goroutines: number;
    heap_alloc_mb: number;
    heap_sys_mb: number;
    num_gc: number;
    uptime_hours: number;
  };
  collected_at: string;
}

// Service status derived from health checks and provider SLIs
export interface ServiceStatus {
  name: string;
  status: "operational" | "degraded" | "down" | "maintenance";
  latency?: number;
  uptime?: number;
  lastChecked: Date;
  details?: string;
}

// System metrics for display
export interface SystemMetric {
  label: string;
  value: number;
  max: number;
  unit: string;
  status: "good" | "warning" | "critical";
}

// Full system status response
export interface SystemStatusResponse {
  overall: "operational" | "degraded" | "down";
  services: ServiceStatus[];
  metrics: SystemMetric[];
  lastUpdated: Date;
}

// Fetch core health status
export async function fetchHealthStatus(): Promise<HealthResponse> {
  try {
    const { data } = await axios.get<HealthResponse>("/healthz");
    return data;
  } catch {
    return { status: "error", checks: {} };
  }
}

// Fetch backend system metrics
async function fetchBackendMetrics(): Promise<BackendSystemMetrics | null> {
  try {
    const { data } = await api.get<BackendSystemMetrics>("/system/metrics");
    return data;
  } catch {
    return null;
  }
}

// Fetch provider SLIs for provider health status
async function fetchProviderSLIs(): Promise<ProviderSLI[]> {
  try {
    return await listProviderSLIs();
  } catch {
    return [];
  }
}

// Fetch combined system status
export async function fetchSystemStatus(): Promise<SystemStatusResponse> {
  const now = new Date();

  let healthResponse: HealthResponse = { status: "ok", checks: {} };
  let providerSLIs: ProviderSLI[] = [];
  let backendMetrics: BackendSystemMetrics | null = null;

  try {
    [healthResponse, providerSLIs, backendMetrics] = await Promise.all([
      fetchHealthStatus(),
      fetchProviderSLIs(),
      fetchBackendMetrics(),
    ]);
  } catch {
    // Continue with defaults if fetching fails
  }

  const services: ServiceStatus[] = [];

  // Add core infrastructure services from health checks
  if (healthResponse.checks) {
    for (const [name, check] of Object.entries(healthResponse.checks)) {
      const displayName = formatServiceName(name);
      services.push({
        name: displayName,
        status: check.status === "ok" ? "operational" : "degraded",
        latency: check.latency_ms,
        lastChecked: now,
        details: check.error,
      });
    }
  }

  // Add API Gateway as always operational if we got a response
  services.unshift({
    name: "API Gateway",
    status: "operational",
    latency: 0,
    lastChecked: now,
  });

  // Add provider services from SLIs
  const providerMap = new Map<string, ServiceStatus>();
  for (const sli of providerSLIs) {
    const existing = providerMap.get(sli.provider);
    const isHealthy = isProviderHealthy(sli);

    if (!existing) {
      providerMap.set(sli.provider, {
        name: formatProviderName(sli.provider),
        status: isHealthy ? "operational" : "degraded",
        latency: sli.latency_p95_ms,
        uptime: calculateUptime(sli),
        lastChecked: now,
        details: isHealthy ? undefined : getProviderIssue(sli),
      });
    } else if (!isHealthy && existing.status === "operational") {
      existing.status = "degraded";
      existing.details = getProviderIssue(sli);
    }
  }

  // Add provider services
  for (const service of providerMap.values()) {
    services.push(service);
  }

  // Determine overall status
  const hasDown = services.some((s) => s.status === "down");
  const hasDegraded = services.some((s) => s.status === "degraded");
  const overall = hasDown ? "down" : hasDegraded ? "degraded" : "operational";

  // Build metrics from real backend data
  const metrics = buildMetricsFromBackend(backendMetrics);

  return {
    overall,
    services,
    metrics,
    lastUpdated: now,
  };
}

// Build display metrics from backend metrics
function buildMetricsFromBackend(backendMetrics: BackendSystemMetrics | null): SystemMetric[] {
  if (!backendMetrics) {
    // Return empty/zero metrics if no backend data
    return [
      { label: "Requests (24h)", value: 0, max: 10000, unit: "", status: "good" },
      { label: "Requests/min", value: 0, max: 100, unit: "", status: "good" },
      { label: "Error Rate", value: 0, max: 100, unit: "%", status: "good" },
      { label: "Avg Latency", value: 0, max: 5000, unit: "ms", status: "good" },
    ];
  }

  const { requests, tokens, infrastructure, runtime } = backendMetrics;

  // Determine status thresholds
  const getErrorStatus = (rate: number): "good" | "warning" | "critical" => {
    if (rate > 10) return "critical";
    if (rate > 5) return "warning";
    return "good";
  };

  const getLatencyStatus = (ms: number): "good" | "warning" | "critical" => {
    if (ms > 2000) return "critical";
    if (ms > 1000) return "warning";
    return "good";
  };

  const getConnectionStatus = (percent: number): "good" | "warning" | "critical" => {
    if (percent > 90) return "critical";
    if (percent > 75) return "warning";
    return "good";
  };

  return [
    {
      label: "Requests (24h)",
      value: requests.total_24h,
      max: Math.max(requests.total_24h * 1.5, 10000),
      unit: "",
      status: "good",
    },
    {
      label: "Requests/min",
      value: Math.round(requests.requests_per_minute * 10) / 10,
      max: 100,
      unit: "",
      status: "good",
    },
    {
      label: "Error Rate",
      value: Math.round(requests.error_rate_24h * 100) / 100,
      max: 100,
      unit: "%",
      status: getErrorStatus(requests.error_rate_24h),
    },
    {
      label: "Avg Latency",
      value: Math.round(requests.avg_latency_ms),
      max: 5000,
      unit: "ms",
      status: getLatencyStatus(requests.avg_latency_ms),
    },
    {
      label: "P95 Latency",
      value: Math.round(requests.p95_latency_ms),
      max: 5000,
      unit: "ms",
      status: getLatencyStatus(requests.p95_latency_ms),
    },
    {
      label: "Tokens (24h)",
      value: tokens.total_24h,
      max: Math.max(tokens.total_24h * 1.5, 100000),
      unit: "",
      status: "good",
    },
    {
      label: "Tokens/min",
      value: Math.round(tokens.tokens_per_minute),
      max: 10000,
      unit: "",
      status: "good",
    },
    {
      label: "DB Connections",
      value: infrastructure.database.acquired_connections,
      max: infrastructure.database.max_connections || 100,
      unit: "",
      status: getConnectionStatus(infrastructure.database.connection_usage_percent),
    },
    {
      label: "Goroutines",
      value: runtime.goroutines,
      max: 1000,
      unit: "",
      status: runtime.goroutines > 500 ? "warning" : "good",
    },
    {
      label: "Heap Memory",
      value: runtime.heap_alloc_mb,
      max: runtime.heap_sys_mb || 1000,
      unit: "MB",
      status: runtime.heap_alloc_mb > runtime.heap_sys_mb * 0.9 ? "warning" : "good",
    },
  ];
}

// Helper functions
function formatServiceName(name: string): string {
  return name
    .split(/[_-]/)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

function formatProviderName(provider: string): string {
  const names: Record<string, string> = {
    openai: "OpenAI Provider",
    anthropic: "Anthropic Provider",
    azureopenai: "Azure OpenAI",
    bedrock: "AWS Bedrock",
    vertex: "Vertex AI",
    groq: "Groq Provider",
    openrouter: "OpenRouter",
    vllm: "vLLM",
  };
  return names[provider.toLowerCase()] ?? `${formatServiceName(provider)} Provider`;
}

function isProviderHealthy(sli: ProviderSLI): boolean {
  if (sli.request_count < (sli.thresholds?.min_samples ?? 10)) {
    return true; // Not enough data
  }
  return (
    sli.error_rate <= (sli.thresholds?.error_rate_threshold ?? 0.05) &&
    sli.timeout_rate <= (sli.thresholds?.timeout_rate_threshold ?? 0.05) &&
    sli.latency_p95_ms <= (sli.thresholds?.latency_p95_ms ?? 5000)
  );
}

function calculateUptime(sli: ProviderSLI): number {
  // Approximate uptime based on error and timeout rates
  const errorRate = sli.error_rate ?? 0;
  const timeoutRate = sli.timeout_rate ?? 0;
  const successRate = 1 - errorRate - timeoutRate;
  return Math.round(successRate * 10000) / 100; // Round to 2 decimal places
}

function getProviderIssue(sli: ProviderSLI): string {
  const issues: string[] = [];

  if (sli.error_rate > (sli.thresholds?.error_rate_threshold ?? 0.05)) {
    issues.push(`High error rate (${(sli.error_rate * 100).toFixed(1)}%)`);
  }
  if (sli.timeout_rate > (sli.thresholds?.timeout_rate_threshold ?? 0.05)) {
    issues.push(`High timeout rate (${(sli.timeout_rate * 100).toFixed(1)}%)`);
  }
  if (sli.latency_p95_ms > (sli.thresholds?.latency_p95_ms ?? 5000)) {
    issues.push(`Elevated latency (${sli.latency_p95_ms}ms)`);
  }

  return issues.join("; ") || "Performance degradation detected";
}
