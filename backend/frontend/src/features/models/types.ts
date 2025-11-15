import type {
  ProviderOverrides,
  VertexProviderConfig,
} from "@/api/model-catalog";

export type CustomMetadataEntry = {
  id: string;
  key: string;
  value: string;
};

export type ModelFormState = {
  alias: string;
  provider: string;
  provider_model: string;
  model_type: string;
  context_window: number | "";
  max_output_tokens: number | "";
  modalities: string[];
  supports_tools: boolean;
  price_input: string;
  price_output: string;
  currency: string;
  deployment: string;
  endpoint: string;
  api_key: string;
  api_version: string;
  region: string;
  metadata: Record<string, string>;
  customMetadata: CustomMetadataEntry[];
  weight: number | "";
  enabled: boolean;
  provider_overrides: ProviderOverrides;
};

export function defaultVertexOverride(): VertexProviderConfig {
  return {
    gcp_project_id: "",
    vertex_location: "",
    vertex_publisher: "",
    gcp_credentials_json: "",
    gcp_credentials_format: "json",
  };
}

export const MODEL_TYPE_OPTIONS = [
  { value: "llm", label: "LLM" },
  { value: "embedding", label: "Embedding" },
  { value: "image", label: "Image" },
  { value: "audio", label: "Audio" },
  { value: "video", label: "Video" },
  { value: "moderation", label: "Moderation" },
];

export const MODEL_TYPE_LABELS: Record<string, string> = MODEL_TYPE_OPTIONS.reduce(
  (acc, option) => {
    acc[option.value] = option.label;
    return acc;
  },
  {} as Record<string, string>,
);

export function formatModelTypeLabel(value: string | undefined) {
  if (!value) {
    return "LLM";
  }
  return MODEL_TYPE_LABELS[value] ?? value.toUpperCase();
}
