import type {
  ModelCatalogEntry,
  ProviderOverrides,
  PricingTier,
  PricingTierMap,
  VertexProviderConfig,
} from "@/api/model-catalog";

import {
  partitionMetadataForProvider,
  sanitizeMetadataPayload,
} from "./metadata";
import {
  defaultVertexOverride,
  type ModelFormState,
  type PricingTierForm,
  type PricingTiersFormState,
} from "./types";

const DEFAULT_PRICING_BUCKETS = ["input", "output"] as const;

function createEmptyPricingTiers(): PricingTiersFormState {
  const buckets: PricingTiersFormState = {};
  for (const bucket of DEFAULT_PRICING_BUCKETS) {
    buckets[bucket] = [];
  }
  return buckets;
}

function generateTierId() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2);
}

function formatMetadataLines(metadata?: Record<string, string>) {
  if (!metadata || Object.keys(metadata).length === 0) {
    return "";
  }
  return Object.entries(metadata)
    .map(([key, value]) => `${key}=${value ?? ""}`)
    .join("\n");
}

function parseMetadataLines(input: string) {
  const trimmed = input.trim();
  if (!trimmed) {
    return undefined;
  }
  const map: Record<string, string> = {};
  trimmed.split(/\n+/).forEach((line) => {
    const raw = line.trim();
    if (!raw) {
      return;
    }
    const [key, ...rest] = raw.split("=");
    const normalizedKey = key.trim();
    if (!normalizedKey) {
      return;
    }
    map[normalizedKey] = rest.join("=").trim();
  });
  return Object.keys(map).length > 0 ? map : undefined;
}

function mapPricingTiersToForm(
  pricing?: PricingTierMap,
): PricingTiersFormState {
  const state = createEmptyPricingTiers();
  if (!pricing) {
    return state;
  }
  for (const [bucket, tiers] of Object.entries(pricing)) {
    state[bucket] = tiers.map((tier) => ({
      id: generateTierId(),
      unit: tier.unit || "tokens_per_million",
      price_per_unit:
        tier.price_per_unit !== undefined
          ? tier.price_per_unit.toString()
          : "",
      max_units:
        tier.max_units !== undefined ? tier.max_units.toString() : "",
      metadata: formatMetadataLines(tier.metadata),
    } satisfies PricingTierForm));
  }
  return state;
}

export function createEmptyModelForm(): ModelFormState {
  return {
    alias: "",
    provider: "azure",
    provider_model: "",
    model_type: "llm",
    context_window: "",
    max_output_tokens: "",
    modalities: [],
    supports_tools: false,
    price_input: "",
    price_output: "",
    currency: "USD",
    deployment: "",
    endpoint: "",
    api_key: "",
    api_version: "",
    region: "",
    metadata: {},
    customMetadata: [],
    weight: "",
    enabled: true,
    tenant_assignable: false,
    provider_overrides: {},
    pricing_tiers: createEmptyPricingTiers(),
  };
}

export function cloneProviderOverrides(
  overrides?: ProviderOverrides,
): ProviderOverrides {
  if (!overrides) {
    return {};
  }
  try {
    return JSON.parse(JSON.stringify(overrides));
  } catch {
    return { ...overrides };
  }
}

export function mapEntryToForm(entry: ModelCatalogEntry): ModelFormState {
  const { metadata, customMetadata } = partitionMetadataForProvider(
    entry.metadata,
    entry.provider,
  );
  return {
    alias: entry.alias,
    provider: entry.provider,
    provider_model: entry.provider_model,
    model_type: entry.model_type || "llm",
    context_window: entry.context_window,
    max_output_tokens: entry.max_output_tokens,
    modalities: [...entry.modalities],
    supports_tools: entry.supports_tools,
    price_input:
      entry.price_input !== undefined ? entry.price_input.toString() : "",
    price_output:
      entry.price_output !== undefined ? entry.price_output.toString() : "",
    currency: entry.currency || "USD",
    deployment: entry.deployment,
    endpoint: entry.endpoint,
    api_key: entry.apiKey || "",
    api_version: entry.api_version,
    region: entry.region,
    metadata,
    customMetadata,
    weight: entry.weight,
    enabled: entry.enabled,
    tenant_assignable: entry.tenant_assignable,
    provider_overrides: {
      ...entry.provider_overrides,
      vertex: entry.provider_overrides?.vertex
        ? ({
            ...defaultVertexOverride(),
            ...entry.provider_overrides.vertex,
          } satisfies VertexProviderConfig)
        : undefined,
    },
    pricing_tiers: mapPricingTiersToForm(entry.pricing_tiers),
  };
}

export function buildMetadataPayload(form: ModelFormState) {
  return sanitizeMetadataPayload(form.metadata, form.customMetadata);
}

export function buildPricingPayload(
  form: ModelFormState,
): PricingTierMap | undefined {
  const entries = Object.entries(form.pricing_tiers ?? {});
  if (entries.length === 0) {
    return undefined;
  }

  const output: PricingTierMap = {};
  let hasAny = false;

  for (const [bucket, tiers] of entries) {
    if (tiers.length === 0) {
      continue;
    }
    const normalized = tiers
      .map((tier) => {
        const unit = tier.unit?.trim() || "tokens_per_million";
        const price = Number(tier.price_per_unit);
        if (!Number.isFinite(price) || price <= 0) {
          return undefined;
        }
        const normalizedTier: PricingTier = {
          unit,
          price_per_unit: price,
        };
        const maxUnits = Number(tier.max_units);
        if (
          tier.max_units.trim() !== "" &&
          Number.isFinite(maxUnits) &&
          maxUnits > 0
        ) {
          normalizedTier.max_units = maxUnits;
        }
        const metadata = parseMetadataLines(tier.metadata);
        if (metadata) {
          normalizedTier.metadata = metadata;
        }
        return normalizedTier;
      })
      .filter((tier): tier is PricingTier => Boolean(tier));

    if (normalized.length > 0) {
      output[bucket] = normalized;
      hasAny = true;
    }
  }

  return hasAny ? output : undefined;
}
