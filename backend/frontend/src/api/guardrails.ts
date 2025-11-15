import { api, type RequestConfig } from "./client";

export type GuardrailKeywordConfig = {
  blocked_keywords?: string[];
};

export type GuardrailModerationConfig = {
  enabled?: boolean;
  provider?: string;
  action?: string;
  webhook_url?: string;
  webhook_auth_header?: string;
  webhook_auth_value?: string;
  timeout_seconds?: number;
};

export type GuardrailConfig = {
  enabled?: boolean;
  prompt?: GuardrailKeywordConfig;
  response?: GuardrailKeywordConfig;
  moderation?: GuardrailModerationConfig;
};

export type GuardrailResponse = {
  config: GuardrailConfig;
};

export async function getTenantGuardrails(
  tenantId: string,
  config?: RequestConfig,
): Promise<GuardrailResponse> {
  const { data } = await api.get<GuardrailResponse>(
    `/tenants/${tenantId}/guardrails`,
    config,
  );
  return data;
}

export async function upsertTenantGuardrails(
  tenantId: string,
  config: GuardrailConfig,
): Promise<GuardrailResponse> {
  const { data } = await api.put<GuardrailResponse>(
    `/tenants/${tenantId}/guardrails`,
    { config },
  );
  return data;
}

export async function deleteTenantGuardrails(tenantId: string): Promise<void> {
  await api.delete(`/tenants/${tenantId}/guardrails`);
}

export async function getApiKeyGuardrails(
  tenantId: string,
  apiKeyId: string,
  config?: RequestConfig,
): Promise<GuardrailResponse> {
  const { data } = await api.get<GuardrailResponse>(
    `/tenants/${tenantId}/api-keys/${apiKeyId}/guardrails`,
    config,
  );
  return data;
}

export async function upsertApiKeyGuardrails(
  tenantId: string,
  apiKeyId: string,
  config: GuardrailConfig,
): Promise<GuardrailResponse> {
  const { data } = await api.put<GuardrailResponse>(
    `/tenants/${tenantId}/api-keys/${apiKeyId}/guardrails`,
    { config },
  );
  return data;
}

export async function deleteApiKeyGuardrails(
  tenantId: string,
  apiKeyId: string,
): Promise<void> {
  await api.delete(`/tenants/${tenantId}/api-keys/${apiKeyId}/guardrails`);
}

export type GuardrailEventRecord = {
  id: string;
  tenant_id?: string;
  tenant_name?: string;
  api_key_id?: string;
  api_key_name?: string;
  model_alias?: string;
  action: string;
  category?: string;
  stage?: string;
  violations?: string[];
  details: Record<string, unknown>;
  created_at: string;
};

export type ListGuardrailEventsParams = {
  tenantId?: string;
  apiKeyId?: string;
  action?: string;
  stage?: string;
  category?: string;
  start?: string;
  end?: string;
  limit?: number;
  offset?: number;
};

export type GuardrailEventsResponse = {
  events: GuardrailEventRecord[];
  total: number;
  next_offset: number;
};

export async function listGuardrailEvents(
  params: ListGuardrailEventsParams = {},
): Promise<GuardrailEventsResponse> {
  const {
    tenantId,
    apiKeyId,
    action,
    stage,
    category,
    start,
    end,
    limit = 50,
    offset = 0,
  } = params;
  const query: Record<string, string> = {
    limit: limit.toString(),
    offset: offset.toString(),
  };
  if (tenantId) {
    query.tenant_id = tenantId;
  }
  if (apiKeyId) {
    query.api_key_id = apiKeyId;
  }
  if (action) {
    query.action = action;
  }
  if (stage) {
    query.stage = stage;
  }
  if (category) {
    query.category = category;
  }
  if (start) {
    query.start = start;
  }
  if (end) {
    query.end = end;
  }
  const { data } = await api.get<GuardrailEventsResponse>(
    `/guardrails/events`,
    { params: query },
  );
  return data;
}
