import type { PricingTierMap } from "@/api/model-catalog";
import { userApi } from "../userClient";

export interface UserModel {
  alias: string;
  provider: string;
  model_type: string;
  price_input: number;
  price_output: number;
  currency: string;
  enabled: boolean;
  throughput_tokens_per_second?: number;
  avg_latency_ms?: number;
  status: string;
  pricing_tiers?: PricingTierMap;
}

export async function listUserModels(scope?: string): Promise<UserModel[]> {
  const params = scope ? { scope } : undefined;
  const { data } = await userApi.get<{ models: UserModel[] }>("/models", {
    params,
  });
  return data.models;
}
