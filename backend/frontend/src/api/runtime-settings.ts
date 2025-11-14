import { api } from "./client";

export interface FileSettingsPayload {
  max_size_mb: number;
  default_ttl_seconds: number;
  max_ttl_seconds: number;
}

export async function getFileSettings() {
  const { data } = await api.get<FileSettingsPayload>("/settings/files");
  return data;
}

export async function updateFileSettings(payload: FileSettingsPayload) {
  const { data } = await api.put<FileSettingsPayload>("/settings/files", payload);
  return data;
}

export interface BatchSettingsPayload {
  max_requests: number;
  max_concurrency: number;
  default_ttl_seconds: number;
  max_ttl_seconds: number;
}

export async function getBatchSettings() {
  const { data } = await api.get<BatchSettingsPayload>("/settings/batches");
  return data;
}

export async function updateBatchSettings(payload: BatchSettingsPayload) {
  const { data } = await api.put<BatchSettingsPayload>("/settings/batches", payload);
  return data;
}

export interface SMTPSettingsPayload {
  host: string;
  port: number;
  username: string;
  password: string;
  from: string;
  use_tls: boolean;
  skip_tls_verify: boolean;
  connect_timeout_seconds: number;
}

export interface WebhookSettingsPayload {
  timeout_seconds: number;
  max_retries: number;
}

export interface AlertSettingsPayload {
  smtp: SMTPSettingsPayload;
  webhook: WebhookSettingsPayload;
}

export async function getAlertSettings() {
  const { data } = await api.get<AlertSettingsPayload>("/settings/alerts");
  return data;
}

export async function updateAlertSettings(payload: AlertSettingsPayload) {
  const { data } = await api.put<AlertSettingsPayload>("/settings/alerts", payload);
  return data;
}

export async function sendTestAlertEmail(email: string) {
  await api.post("/settings/alerts/test-email", { email });
}
