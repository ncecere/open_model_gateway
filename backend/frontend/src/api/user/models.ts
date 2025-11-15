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
}

export async function listUserModels(): Promise<UserModel[]> {
  const { data } = await userApi.get<{ models: UserModel[] }>("/models");
  return data.models;
}
