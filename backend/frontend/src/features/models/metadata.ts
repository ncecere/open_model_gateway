import type { CustomMetadataEntry } from "./types";

export type MetadataField = {
  key: string;
  label: string;
  description?: string;
  placeholder?: string;
  input?: "text" | "number" | "textarea" | "select" | "boolean";
  options?: { label: string; value: string }[];
};

export type MetadataSection = {
  id: string;
  title: string;
  description?: string;
  fields: MetadataField[];
};

const BASE_METADATA_SECTIONS: MetadataSection[] = [
  {
    id: "routing",
    title: "Routing & rate limits",
    description:
      "Optional per-model caps that override the tenant defaults (RPM, TPM, and concurrent requests).",
    fields: [
      {
        key: "requests_per_minute",
        label: "Requests per minute",
        placeholder: "60",
        input: "number",
      },
      {
        key: "tokens_per_minute",
        label: "Tokens per minute",
        placeholder: "10_000",
        input: "number",
      },
      {
        key: "parallel_requests",
        label: "Parallel requests",
        placeholder: "4",
        input: "number",
      },
    ],
  },
  {
    id: "image-pricing",
    title: "Image pricing overrides",
    description:
      "Used for spend reporting when this model performs image generation/edits/variations.",
    fields: [
      {
        key: "price_image_cents",
        label: "Price per image (cents)",
        placeholder: "2.5",
        input: "number",
      },
    ],
  },
  {
    id: "audio-defaults",
    title: "Audio defaults",
    description:
      "Configure voice + format hints for `/v1/audio/speech` calls routed through this model.",
    fields: [
      {
        key: "audio_voice",
        label: "Preferred voice",
        placeholder: "alloy",
      },
      {
        key: "audio_default_voice",
        label: "Fallback voice",
        placeholder: "verse",
      },
      {
        key: "audio_format",
        label: "Audio format",
        placeholder: "mp3",
      },
    ],
  },
];

const PROVIDER_METADATA_SECTIONS: Record<string, MetadataSection[]> = {
  openai: [
    {
      id: "openai-headers",
      title: "OpenAI headers",
      description:
        "Configure optional organization routing or override the base URL for dedicated endpoints.",
      fields: [
        {
          key: "openai_organization",
          label: "Organization ID",
          placeholder: "org_abc123",
        },
        {
          key: "base_url",
          label: "Base URL override",
          placeholder: "https://api.openai.com/v1",
        },
      ],
    },
  ],
  openai_compatible: [
    {
      id: "openai-compatible-endpoint",
      title: "Compatible endpoint settings",
      description:
        "Required base URL + optional API key/org overrides for Flux, custom gateways, etc.",
      fields: [
        {
          key: "base_url",
          label: "Base URL",
          placeholder: "https://my-gateway.example.com/v1",
        },
        {
          key: "api_key",
          label: "Provider API key override",
          description:
            "Populate if this model should use a different key than the catalog entry.",
        },
        {
          key: "openai_organization",
          label: "Organization header",
          placeholder: "org_abc123",
        },
      ],
    },
  ],
  bedrock: [
    {
      id: "bedrock-behavior",
      title: "Bedrock behavior",
      description:
        "Tune request payloads for Titan/Claude adapters (chat format, embeddings, and image workflows).",
      fields: [
        {
          key: "bedrock_chat_format",
          label: "Chat format",
          placeholder: "anthropic_messages",
        },
        {
          key: "bedrock_embedding_format",
          label: "Embedding format",
          placeholder: "float",
        },
        {
          key: "bedrock_default_max_tokens",
          label: "Default max tokens",
          placeholder: "1024",
          input: "number",
        },
        {
          key: "bedrock_embed_dims",
          label: "Embedding dimensions",
          placeholder: "1536",
          input: "number",
        },
        {
          key: "bedrock_embed_normalize",
          label: "Normalize embeddings",
          description: "Set true to unit-normalize embeddings returned by Titan.",
          input: "boolean",
        },
        {
          key: "bedrock_image_task_type",
          label: "Image task type",
          placeholder: "TEXT_IMAGE",
          description: "TEXT_IMAGE for generation, IMAGE_IMAGE for variations.",
        },
        {
          key: "anthropic_version",
          label: "Anthropic API version",
          placeholder: "bedrock-2023-05-31",
        },
      ],
    },
    {
      id: "bedrock-aws",
      title: "AWS credentials overrides",
      description:
        "Fallback to these values if the global AWS provider config is empty.",
      fields: [
        { key: "aws_access_key_id", label: "Access key ID" },
        { key: "aws_secret_access_key", label: "Secret access key" },
        { key: "aws_session_token", label: "Session token" },
        { key: "aws_profile", label: "Shared credentials profile" },
      ],
    },
  ],
  vertex: [
    {
      id: "vertex-image",
      title: "Vertex image tuning",
      description:
        "Optional Imagen parameters for edits/variations (mask controls, guidance, etc.).",
      fields: [
        {
          key: "vertex_edit_mode",
          label: "Edit mode",
          placeholder: "EDIT_MODE_INPAINT_INSERTION",
        },
        {
          key: "vertex_guidance_scale",
          label: "Guidance scale",
          placeholder: "1.2",
          input: "number",
        },
        {
          key: "vertex_person_generation",
          label: "Person generation policy",
          placeholder: "PERSON_GENERATION_MODE_ALLOWED",
        },
        {
          key: "vertex_mask_mode",
          label: "Mask mode",
          placeholder: "MASK_MODE_USER_PROVIDED",
        },
        {
          key: "vertex_mask_dilation",
          label: "Mask dilation",
          placeholder: "0.01",
          input: "number",
        },
        {
          key: "vertex_variation_prompt",
          label: "Variation prompt seed",
          placeholder: "Add impressionist lighting",
        },
        {
          key: "vertex_base_steps",
          label: "Edit base steps",
          placeholder: "25",
          input: "number",
        },
      ],
    },
  ],
};

const BASE_METADATA_KEYS = new Set(
  BASE_METADATA_SECTIONS.flatMap((section) =>
    section.fields.map((field) => field.key),
  ),
);

const PROVIDER_METADATA_KEYS: Record<string, Set<string>> = Object.fromEntries(
  Object.entries(PROVIDER_METADATA_SECTIONS).map(([provider, sections]) => [
    provider,
    new Set(sections.flatMap((section) => section.fields.map((field) => field.key))),
  ]),
);

function getKnownMetadataKeys(provider: string): Set<string> {
  const keys = new Set(BASE_METADATA_KEYS);
  const providerKeys = PROVIDER_METADATA_KEYS[provider];
  if (providerKeys) {
    providerKeys.forEach((key) => keys.add(key));
  }
  return keys;
}

export function metadataSectionsForProvider(provider: string): MetadataSection[] {
  return [
    ...BASE_METADATA_SECTIONS,
    ...(PROVIDER_METADATA_SECTIONS[provider] ?? []),
  ].filter((section) => section.fields.length > 0);
}

export function metadataHasValues(
  metadata: Record<string, string>,
  customMetadata: CustomMetadataEntry[],
): boolean {
  return (
    Object.values(metadata ?? {}).some((value) => value.trim() !== "") ||
    customMetadata.some(
      (entry) => entry.key.trim() !== "" && entry.value.trim() !== "",
    )
  );
}

export function createCustomMetadataEntry(
  key = "",
  value = "",
): CustomMetadataEntry {
  const id =
    typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
      ? crypto.randomUUID()
      : Math.random().toString(36).slice(2);
  return { id, key, value };
}

export function partitionMetadataForProvider(
  source: Record<string, string>,
  provider: string,
): { metadata: Record<string, string>; customMetadata: CustomMetadataEntry[] } {
  const metadata: Record<string, string> = {};
  const customMetadata: CustomMetadataEntry[] = [];
  if (!source) {
    return { metadata, customMetadata };
  }
  const knownKeys = getKnownMetadataKeys(provider);
  Object.entries(source).forEach(([key, value]) => {
    if (knownKeys.has(key)) {
      metadata[key] = value;
    } else {
      customMetadata.push(createCustomMetadataEntry(key, value));
    }
  });
  return { metadata, customMetadata };
}

export function normalizeMetadataForProvider(
  metadata: Record<string, string>,
  customMetadata: CustomMetadataEntry[],
  provider: string,
): { metadata: Record<string, string>; customMetadata: CustomMetadataEntry[] } {
  const allowedKeys = getKnownMetadataKeys(provider);
  const nextMetadata: Record<string, string> = {};
  const preservedCustom: CustomMetadataEntry[] = [];
  const customByKey = new Map<string, CustomMetadataEntry>();

  customMetadata.forEach((entry) => {
    const cloned = { ...entry };
    const trimmedKey = cloned.key.trim();
    if (trimmedKey && allowedKeys.has(trimmedKey)) {
      nextMetadata[trimmedKey] = cloned.value;
      return;
    }
    preservedCustom.push(cloned);
    if (trimmedKey) {
      customByKey.set(trimmedKey, cloned);
    }
  });

  Object.entries(metadata).forEach(([key, value]) => {
    if (allowedKeys.has(key)) {
      nextMetadata[key] = value;
      return;
    }
    const trimmedKey = key.trim();
    if (trimmedKey && customByKey.has(trimmedKey)) {
      customByKey.get(trimmedKey)!.value = value;
      return;
    }
    const extra = createCustomMetadataEntry(key, value);
    preservedCustom.push(extra);
    if (trimmedKey) {
      customByKey.set(trimmedKey, extra);
    }
  });

  return { metadata: nextMetadata, customMetadata: preservedCustom };
}

export function sanitizeMetadataPayload(
  metadata: Record<string, string>,
  customMetadata: CustomMetadataEntry[],
): Record<string, string> {
  const payload: Record<string, string> = {};
  Object.entries(metadata).forEach(([key, value]) => {
    const trimmedKey = key.trim();
    const trimmedValue = value?.trim?.() ?? "";
    if (trimmedKey && trimmedValue) {
      payload[trimmedKey] = trimmedValue;
    }
  });
  customMetadata.forEach((entry) => {
    const trimmedKey = entry.key.trim();
    const trimmedValue = entry.value.trim();
    if (trimmedKey && trimmedValue) {
      payload[trimmedKey] = trimmedValue;
    }
  });
  return payload;
}
