import type {
  ModelCatalogEntry,
  ProviderOverrides,
  VertexProviderConfig,
} from "@/api/model-catalog";

import {
  partitionMetadataForProvider,
  sanitizeMetadataPayload,
} from "./metadata";
import { defaultVertexOverride, type ModelFormState } from "./types";

export function createEmptyModelForm(): ModelFormState {
  return {
    alias: "",
    provider: "azure",
    provider_model: "",
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
    provider_overrides: {},
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
    provider_overrides: {
      ...entry.provider_overrides,
      vertex: entry.provider_overrides?.vertex
        ? ({
            ...defaultVertexOverride(),
            ...entry.provider_overrides.vertex,
          } satisfies VertexProviderConfig)
        : undefined,
    },
  };
}

export function buildMetadataPayload(form: ModelFormState) {
  return sanitizeMetadataPayload(form.metadata, form.customMetadata);
}
