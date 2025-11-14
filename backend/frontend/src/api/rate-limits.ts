import { api } from "./client";

export type RateLimitDefaults = {
  requests_per_minute: number;
  tokens_per_minute: number;
  parallel_requests_key: number;
  parallel_requests_tenant: number;
};

export async function getRateLimitDefaults() {
  const { data } = await api.get<RateLimitDefaults>("/settings/rate-limits");
  return data;
}

export async function updateRateLimitDefaults(payload: RateLimitDefaults) {
  const { data } = await api.put<RateLimitDefaults>(
    "/settings/rate-limits",
    payload,
  );
  return data;
}
