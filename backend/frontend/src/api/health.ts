import axios from "axios";

export interface HealthCheck {
  status: string;
  latency_ms?: number;
  error?: string;
}

export interface HealthResponse {
  status: string;
  checks?: {
    postgres?: HealthCheck;
    redis?: HealthCheck;
  };
}

export async function fetchHealth() {
  const { data } = await axios.get<HealthResponse>("/healthz");
  return data;
}
