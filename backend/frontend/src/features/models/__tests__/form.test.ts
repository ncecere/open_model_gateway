import { describe, expect, it } from "vitest";

import type { ModelCatalogEntry } from "@/api/model-catalog";
import { createEmptyModelForm, mapEntryToForm } from "../form";

describe("model form mapping", () => {
  it("defaults tenant_assignable to false", () => {
    const form = createEmptyModelForm();
    expect(form.tenant_assignable).toBe(false);
  });

  it("maps tenant_assignable from catalog entry", () => {
    const entry: ModelCatalogEntry = {
      alias: "gpt-test",
      provider: "openai",
      provider_model: "gpt-test",
      model_type: "llm",
      context_window: 4096,
      max_output_tokens: 2048,
      modalities: ["text"],
      supports_tools: false,
      price_input: 0.01,
      price_output: 0.02,
      currency: "USD",
      enabled: true,
      tenant_assignable: true,
      updated_at: new Date().toISOString(),
      deployment: "gpt-test",
      endpoint: "",
      apiKey: "",
      api_version: "",
      region: "",
      metadata: {},
      weight: 100,
      provider_overrides: {},
      pricing_tiers: {},
    };

    const form = mapEntryToForm(entry);
    expect(form.tenant_assignable).toBe(true);
  });
});
