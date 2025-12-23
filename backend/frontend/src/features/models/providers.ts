import AnthropicLogo from "@/assets/providers/anthropic_light.svg";
import AzureLogo from "@/assets/providers/azure.svg";
import BedrockLogo from "@/assets/providers/bedrock.svg";
import OpenAILogo from "@/assets/providers/openai_light.svg";
import OpenAICompatibleLogo from "@/assets/providers/openai_compatable.svg";
import VLLMLogo from "@/assets/providers/vllm.svg";
import VertexLogo from "@/assets/providers/vertexai.svg";
import OpenRouterLogo from "@/assets/providers/openrouter_light.svg";
import GroqLogo from "@/assets/providers/groq.svg";

export type ProviderConfig = {
  showEndpoint: boolean;
  showApiKey: boolean;
  showApiVersion: boolean;
  showDeployment: boolean;
  showRegion: boolean;
};

export type ProviderDetail = {
  value: string;
  label: string;
  logo: string | null;
  config: ProviderConfig;
};

export const DEFAULT_PROVIDER_DETAIL: ProviderDetail = {
  value: "custom",
  label: "Custom provider",
  logo: null,
  config: {
    showEndpoint: false,
    showApiKey: false,
    showApiVersion: false,
    showDeployment: true,
    showRegion: true,
  },
};

export const PROVIDER_DETAILS: Record<string, ProviderDetail> = {
  azure: {
    value: "azure",
    label: "Azure OpenAI",
    logo: AzureLogo,
    config: {
      showEndpoint: true,
      showApiKey: true,
      showApiVersion: true,
      showDeployment: true,
      showRegion: true,
    },
  },
  bedrock: {
    value: "bedrock",
    label: "Amazon Bedrock",
    logo: BedrockLogo,
    config: {
      showEndpoint: false,
      showApiKey: false,
      showApiVersion: false,
      showDeployment: true,
      showRegion: true,
    },
  },
  openai: {
    value: "openai",
    label: "OpenAI",
    logo: OpenAILogo,
    config: {
      showEndpoint: false,
      showApiKey: true,
      showApiVersion: false,
      showDeployment: false,
      showRegion: false,
    },
  },
  "openai-compatible": {
    value: "openai-compatible",
    label: "OpenAI-compatible",
    logo: OpenAICompatibleLogo,
    config: {
      showEndpoint: true,
      showApiKey: true,
      showApiVersion: false,
      showDeployment: false,
      showRegion: false,
    },
  },
  openrouter: {
    value: "openrouter",
    label: "OpenRouter",
    logo: OpenRouterLogo,
    config: {
      showEndpoint: false,
      showApiKey: true,
      showApiVersion: false,
      showDeployment: false,
      showRegion: false,
    },
  },
  groq: {
    value: "groq",
    label: "Groq",
    logo: GroqLogo,
    config: {
      showEndpoint: false,
      showApiKey: true,
      showApiVersion: false,
      showDeployment: false,
      showRegion: true,
    },
  },
  vllm: {
    value: "vllm",
    label: "vLLM / TGI",
    logo: VLLMLogo,
    config: {
      showEndpoint: true,
      showApiKey: true,
      showApiVersion: false,
      showDeployment: false,
      showRegion: false,
    },
  },
  anthropic: {
    value: "anthropic",
    label: "Anthropic",
    logo: AnthropicLogo,
    config: {
      showEndpoint: false,
      showApiKey: true,
      showApiVersion: false,
      showDeployment: false,
      showRegion: false,
    },
  },
  vertex: {
    value: "vertex",
    label: "Google Vertex",
    logo: VertexLogo,
    config: {
      showEndpoint: false,
      showApiKey: false,
      showApiVersion: false,
      showDeployment: true,
      showRegion: false,
    },
  },
};

export const SUPPORTED_PROVIDERS = Object.values(PROVIDER_DETAILS).map(
  ({ value, label }) => ({ value, label }),
);
