import AnthropicIconLight from "@/assets/providers/anthropic_light.svg";
import AnthropicIconDark from "@/assets/providers/anthropic_dark.svg";
import BedrockIcon from "@/assets/providers/bedrock.svg";
import VertexIcon from "@/assets/providers/vertexai.svg";
import AzureIcon from "@/assets/providers/azure.svg";
import OpenAIIconLight from "@/assets/providers/openai_light.svg";
import OpenAIIconDark from "@/assets/providers/openai_dark.svg";
import OpenAICompatIcon from "@/assets/providers/openai_compatable.svg";

const PROVIDER_ICON_SETS: Record<string, { light: string; dark: string }> = {
  anthropic: {
    light: AnthropicIconLight,
    dark: AnthropicIconDark,
  },
  bedrock: {
    light: BedrockIcon,
    dark: BedrockIcon,
  },
  vertex: {
    light: VertexIcon,
    dark: VertexIcon,
  },
  azure: {
    light: AzureIcon,
    dark: AzureIcon,
  },
  openai: {
    light: OpenAIIconLight,
    dark: OpenAIIconDark,
  },
  "openai-compatible": {
    light: OpenAICompatIcon,
    dark: OpenAICompatIcon,
  },
};

function normalizeProviderKey(provider: string) {
  const trimmed = provider?.toLowerCase() ?? "";
  if (trimmed === "openai_compatible") {
    return "openai-compatible";
  }
  return trimmed;
}

export function getProviderIcon(
  provider: string,
  theme: "light" | "dark",
): string | null {
  const key = normalizeProviderKey(provider);
  const entry = PROVIDER_ICON_SETS[key];
  if (!entry) {
    return null;
  }
  return entry[theme] ?? entry.light ?? null;
}
