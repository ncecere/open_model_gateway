import { api } from "./client";

export interface AuditLogEntry {
  id: string;
  user_id?: string;
  action: string;
  resource: string;
  resource_id: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface ListAuditLogsParams {
  limit?: number;
  offset?: number;
  user_id?: string;
  action?: string;
  resource?: string;
}

export interface ListAuditLogsResponse {
  logs: AuditLogEntry[];
  limit: number;
  offset: number;
}

export async function listAuditLogs(params?: ListAuditLogsParams): Promise<ListAuditLogsResponse> {
  const { data } = await api.get<ListAuditLogsResponse>("/audit/logs", { params });
  return data;
}

// Get count of audit logs (for stats)
export async function getAuditLogStats(): Promise<{
  total: number;
  auth_events: number;
  failed_events: number;
  unique_actors: number;
}> {
  // Get a sample of logs to calculate stats
  // In a real implementation, this would be a dedicated endpoint
  const { data } = await api.get<ListAuditLogsResponse>("/audit/logs", {
    params: { limit: 200 },
  });

  const logs = data.logs ?? [];
  const authEvents = logs.filter((l) =>
    l.action.toLowerCase().includes("login") ||
    l.action.toLowerCase().includes("auth")
  ).length;

  const failedEvents = logs.filter((l) =>
    l.action.toLowerCase().includes("failed") ||
    l.action.toLowerCase().includes("error")
  ).length;

  const uniqueActors = new Set(logs.map((l) => l.user_id).filter(Boolean)).size;

  return {
    total: logs.length,
    auth_events: authEvents,
    failed_events: failedEvents,
    unique_actors: uniqueActors,
  };
}
