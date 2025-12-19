import { api } from "./client";

interface ModelCatalogDTO {
  alias: string;
  provider: string;
  provider_model: string;
  model_type?: string;
  context_window: number;
  max_output_tokens: number;
  modalities_json: string;
  supports_tools: boolean;
  pricing_tiers_json?: string;
  price_input: string;
  price_output: string;
  currency: string;
  enabled: boolean;
  updated_at: string;
  deployment: string;
  endpoint: string;
  api_key: string;
  api_version: string;
  region: string;
  metadata_json: string;
  weight: number;
  provider_config_json?: string;
}

export interface AzureProviderConfig {
  deployment?: string;
  endpoint?: string;
  api_key?: string;
  api_version?: string;
  region?: string;
}

export interface VertexProviderConfig {
  gcp_project_id?: string;
  vertex_location?: string;
  vertex_publisher?: string;
  gcp_credentials_json?: string;
  gcp_credentials_format?: string;
}

export interface BedrockProviderConfig {
  region?: string;
  bedrock_chat_format?: string;
  bedrock_embedding_format?: string;
  bedrock_default_max_tokens?: number;
  bedrock_embed_dims?: number;
  bedrock_embed_normalize?: boolean;
  bedrock_image_task_type?: string;
  anthropic_version?: string;
  aws_access_key_id?: string;
  aws_secret_access_key?: string;
  aws_session_token?: string;
  aws_profile?: string;
}

export interface OpenAIProviderConfig {
  api_key?: string;
  openai_organization?: string;
  base_url?: string;
}

export interface OpenAICompatibleProviderConfig {
  api_key?: string;
  openai_organization?: string;
  base_url?: string;
}

export interface OpenRouterProviderConfig {
  api_key?: string;
  base_url?: string;
  referer?: string;
  app_name?: string;
}

export interface GroqProviderConfig {
  api_key?: string;
  base_url?: string;
  region?: string;
}

export interface ProviderOverrides {
  azure?: AzureProviderConfig;
  vertex?: VertexProviderConfig;
  bedrock?: BedrockProviderConfig;
  openai?: OpenAIProviderConfig;
  openai_compatible?: OpenAICompatibleProviderConfig;
  anthropic?: AnthropicProviderConfig;
  openrouter?: OpenRouterProviderConfig;
  groq?: GroqProviderConfig;
}

export interface AnthropicProviderConfig {
  api_key?: string;
  base_url?: string;
  version?: string;
}

export interface PricingTier {
  unit: string;
  max_units?: number;
  price_per_unit: number;
  metadata?: Record<string, string>;
}

export type PricingTierMap = Record<string, PricingTier[]>;

export interface ModelCatalogEntry {
  alias: string;
  provider: string;
  provider_model: string;
  model_type: string;
  context_window: number;
  max_output_tokens: number;
  modalities: string[];
  supports_tools: boolean;
  price_input: number;
  price_output: number;
  currency: string;
  enabled: boolean;
  updated_at: string;
  deployment: string;
  endpoint: string;
  apiKey: string;
  api_version: string;
  region: string;
  metadata: Record<string, string>;
  weight: number;
  provider_overrides: ProviderOverrides;
  pricing_tiers: PricingTierMap;
}

export interface ModelStatus {
  alias: string;
  status: string;
}

export interface ModelCatalogUpsertRequest {
  alias: string;
  provider: string;
  provider_model: string;
  model_type: string;
  context_window: number;
  max_output_tokens: number;
  modalities: string[];
  supports_tools: boolean;
  price_input: number;
  price_output: number;
  currency: string;
  enabled: boolean;
  deployment: string;
  endpoint: string;
  api_key: string;
  api_version: string;
  region: string;
  metadata: Record<string, string>;
  weight: number;
  provider_overrides?: ProviderOverrides;
  pricing_tiers?: PricingTierMap;
}

export function normalizeProviderSlug(value: string): string {
  const slug = value?.trim().toLowerCase() ?? "";
  if (slug === "openai_compatible") {
    return "openai-compatible";
  }
  if (slug === "open_router" || slug === "open-router") {
    return "openrouter";
  }
  return slug;
}

function decodeBase64(raw: string): string {
  try {
    if (typeof atob === "function") {
      return atob(raw);
    }
  } catch {}
  return raw;
}

function decodeBase64Json<T>(raw: string | null | undefined, fallback: T): T {
  if (!raw) {
    return fallback;
  }
  try {
    const value = JSON.parse(decodeBase64(raw)) as T | null;
    return (value ?? fallback) as T;
  } catch {
    return fallback;
  }
}

function parseDecimal(value: string | number | null | undefined): number {
  if (typeof value === "number") {
    return value;
  }
  if (!value) {
    return 0;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function mapCatalogEntry(entry: ModelCatalogDTO): ModelCatalogEntry {
  return {
    alias: entry.alias,
    provider: normalizeProviderSlug(entry.provider),
    provider_model: entry.provider_model,
    model_type: entry.model_type ?? "llm",
    context_window: entry.context_window,
    max_output_tokens: entry.max_output_tokens,
    modalities: decodeBase64Json<string[]>(entry.modalities_json, []),
    supports_tools: entry.supports_tools,
    price_input: parseDecimal(entry.price_input),
    price_output: parseDecimal(entry.price_output),
    currency: entry.currency,
    enabled: entry.enabled,
    updated_at: entry.updated_at,
    deployment: entry.deployment,
    endpoint: entry.endpoint,
    apiKey: entry.api_key,
    api_version: entry.api_version,
    region: entry.region,
    metadata: decodeBase64Json<Record<string, string>>(entry.metadata_json, {}),
    weight: entry.weight,
    provider_overrides: decodeBase64Json<ProviderOverrides>(
      entry.provider_config_json,
      {},
    ),
    pricing_tiers: decodeBase64Json<PricingTierMap>(
      entry.pricing_tiers_json,
      {},
    ),
  };
}

export async function listModelCatalog() {
  const { data } = await api.get<ModelCatalogDTO[]>("/model-catalog");
  return data.map(mapCatalogEntry);
}

export async function listModelStatuses() {
  const { data } = await api.get<{ statuses: ModelStatus[] }>(
    "/model-catalog/status",
  );
  return data.statuses;
}

export async function upsertModel(payload: ModelCatalogUpsertRequest) {
  const { provider_overrides, pricing_tiers, ...rest } = payload;
  const body: Record<string, unknown> = { ...rest };
  if (provider_overrides?.azure) {
    body.azure = provider_overrides.azure;
  }
  if (provider_overrides?.vertex) {
    body.vertex = provider_overrides.vertex;
  }
  if (provider_overrides?.bedrock) {
    body.bedrock = provider_overrides.bedrock;
  }
  if (provider_overrides?.openai) {
    body.openai = provider_overrides.openai;
  }
  if (provider_overrides?.openai_compatible) {
    body.openai_compatible = provider_overrides.openai_compatible;
  }
  if (provider_overrides?.anthropic) {
    body.anthropic = provider_overrides.anthropic;
  }
  if (provider_overrides?.openrouter) {
    body.openrouter = provider_overrides.openrouter;
  }
  if (provider_overrides?.groq) {
    body.groq = provider_overrides.groq;
  }
  if (pricing_tiers) {
    body.pricing_tiers = pricing_tiers;
  }

  const { data } = await api.post<ModelCatalogDTO>("/model-catalog", body);
  return mapCatalogEntry(data);
}

export async function deleteModel(alias: string) {
  await api.delete(`/model-catalog/${alias}`);
}
