import { useState } from "react";

import {
  type ModelCatalogUpsertRequest,
  type ProviderOverrides,
  type VertexProviderConfig,
  normalizeProviderSlug,
} from "@/api/model-catalog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { PROVIDER_DETAILS, DEFAULT_PROVIDER_DETAIL } from "../providers";
import { buildMetadataPayload, buildPricingPayload } from "../form";
import { normalizeMetadataForProvider } from "../metadata";
import { defaultVertexOverride, type ModelFormState } from "../types";
import { BasicInfoTab } from "./tabs/BasicInfoTab";
import { ProviderConfigTab } from "./tabs/ProviderConfigTab";
import { PricingTab } from "./tabs/PricingTab";
import { AdvancedTab } from "./tabs/AdvancedTab";

type TabValue = "basic" | "provider" | "pricing" | "advanced";

interface ValidationErrors {
  basic: string[];
  provider: string[];
  pricing: string[];
  advanced: string[];
}

function validateForm(form: ModelFormState): ValidationErrors {
  const errors: ValidationErrors = {
    basic: [],
    provider: [],
    pricing: [],
    advanced: [],
  };

  // Basic tab validation
  if (!form.alias.trim()) {
    errors.basic.push("Alias is required");
  }
  if (!form.provider.trim()) {
    errors.basic.push("Provider is required");
  }
  if (!form.provider_model.trim()) {
    errors.basic.push("Provider model is required");
  }

  return errors;
}

function getErrorCount(errors: ValidationErrors, tab: TabValue): number {
  return errors[tab].length;
}

function hasAnyErrors(errors: ValidationErrors): boolean {
  return Object.values(errors).some((arr) => arr.length > 0);
}

export function ModelEditorDialog({
  open,
  onOpenChange,
  form,
  onChange,
  onSubmit,
  loading,
  mode,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  form: ModelFormState;
  onChange: (form: ModelFormState) => void;
  onSubmit: (payload: ModelCatalogUpsertRequest) => void;
  loading: boolean;
  mode: "create" | "edit";
}) {
  const [activeTab, setActiveTab] = useState<TabValue>("basic");
  const errors = validateForm(form);

  const providerKey = normalizeProviderSlug(form.provider);
  const providerDetail =
    PROVIDER_DETAILS[providerKey] ?? DEFAULT_PROVIDER_DETAIL;
  const providerConfig = providerDetail.config;

  const baseVertexOverride = {
    ...defaultVertexOverride(),
    ...(form.provider_overrides.vertex ?? {}),
  };
  const vertexOverride =
    form.provider === "vertex"
      ? {
          ...baseVertexOverride,
          vertex_location:
            baseVertexOverride.vertex_location ||
            form.metadata["vertex_location"] ||
            form.region ||
            "",
        }
      : baseVertexOverride;

  const handleProviderChange = (provider: string) => {
    const { metadata, customMetadata } = normalizeMetadataForProvider(
      form.metadata,
      form.customMetadata,
      provider,
    );
    onChange({
      ...form,
      provider,
      metadata,
      customMetadata,
    });
  };

  const handleModelTypeChange = (modelType: string) => {
    const nextModalities = new Set(form.modalities);
    if (modelType.toLowerCase().startsWith("audio")) {
      nextModalities.add("audio");
    }
    onChange({
      ...form,
      model_type: modelType,
      modalities: Array.from(nextModalities),
    });
  };

  const handleSubmit = () => {
    if (hasAnyErrors(errors)) {
      // Navigate to first tab with errors
      for (const tab of ["basic", "provider", "pricing", "advanced"] as TabValue[]) {
        if (errors[tab].length > 0) {
          setActiveTab(tab);
          return;
        }
      }
      return;
    }

    const resolvedDeployment = providerConfig.showDeployment
      ? form.deployment.trim() ||
        form.provider_model.trim() ||
        form.alias.trim()
      : form.provider_model.trim() || form.alias.trim();

    const provider_overrides: ProviderOverrides = {};
    const vertexConfig = form.provider_overrides.vertex;
    if (vertexConfig) {
      const cleanedVertex: VertexProviderConfig = {};
      if (vertexConfig.gcp_project_id?.trim()) {
        cleanedVertex.gcp_project_id = vertexConfig.gcp_project_id.trim();
      }
      if (vertexConfig.vertex_location?.trim()) {
        cleanedVertex.vertex_location = vertexConfig.vertex_location.trim();
      }
      if (vertexConfig.vertex_publisher?.trim()) {
        cleanedVertex.vertex_publisher = vertexConfig.vertex_publisher.trim();
      }
      if (vertexConfig.gcp_credentials_json?.trim()) {
        cleanedVertex.gcp_credentials_json = vertexConfig.gcp_credentials_json.trim();
        cleanedVertex.gcp_credentials_format =
          vertexConfig.gcp_credentials_format?.trim() || "json";
      }
      if (Object.keys(cleanedVertex).length > 0) {
        provider_overrides.vertex = cleanedVertex;
      }
    }

    const derivedRegion =
      form.provider === "vertex"
        ? (vertexOverride.vertex_location?.trim() ?? "")
        : form.region.trim();

    const payload: ModelCatalogUpsertRequest = {
      alias: form.alias.trim(),
      provider: form.provider.trim(),
      provider_model: form.provider_model.trim(),
      model_type: form.model_type || "llm",
      context_window: Number(form.context_window) || 0,
      max_output_tokens: Number(form.max_output_tokens) || 0,
      modalities: form.modalities,
      supports_tools: form.supports_tools,
      price_input: form.price_input ? Number.parseFloat(form.price_input) : 0,
      price_output: form.price_output
        ? Number.parseFloat(form.price_output)
        : 0,
      currency: form.currency.trim() || "USD",
      deployment: resolvedDeployment,
      endpoint: form.endpoint.trim(),
      api_key: form.api_key.trim(),
      api_version: form.api_version.trim(),
      region: derivedRegion,
      metadata: buildMetadataPayload(form),
      weight: Number(form.weight) || 100,
      enabled: form.enabled,
      tenant_assignable: form.tenant_assignable,
      provider_overrides:
        Object.keys(provider_overrides).length > 0
          ? provider_overrides
          : undefined,
    };

    const pricingPayload = buildPricingPayload(form);
    if (pricingPayload) {
      payload.pricing_tiers = pricingPayload;
    }

    onSubmit(payload);
  };

  const renderTabTrigger = (value: TabValue, label: string) => {
    const errorCount = getErrorCount(errors, value);
    return (
      <TabsTrigger value={value} className="relative">
        {label}
        {errorCount > 0 && (
          <span className="ml-1.5 inline-flex h-4 w-4 items-center justify-center rounded-full bg-destructive text-[10px] font-medium text-destructive-foreground">
            {errorCount}
          </span>
        )}
      </TabsTrigger>
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-hidden sm:max-w-2xl flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {mode === "create" ? "Add model" : `Edit ${form.alias}`}
          </DialogTitle>
          <DialogDescription>
            Configure model deployment, pricing, and routing settings.
          </DialogDescription>
        </DialogHeader>

        <Tabs
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as TabValue)}
          className="flex-1 overflow-hidden flex flex-col"
        >
          <TabsList className="grid w-full grid-cols-4">
            {renderTabTrigger("basic", "Basic")}
            {renderTabTrigger("provider", "Provider")}
            {renderTabTrigger("pricing", "Pricing")}
            {renderTabTrigger("advanced", "Advanced")}
          </TabsList>

          <div className="flex-1 overflow-y-auto py-4">
            <TabsContent value="basic" className="mt-0">
              <BasicInfoTab
                form={form}
                onChange={onChange}
                mode={mode}
                onProviderChange={handleProviderChange}
                onModelTypeChange={handleModelTypeChange}
              />
            </TabsContent>

            <TabsContent value="provider" className="mt-0">
              <ProviderConfigTab form={form} onChange={onChange} />
            </TabsContent>

            <TabsContent value="pricing" className="mt-0">
              <PricingTab form={form} onChange={onChange} />
            </TabsContent>

            <TabsContent value="advanced" className="mt-0">
              <AdvancedTab form={form} onChange={onChange} />
            </TabsContent>
          </div>
        </Tabs>

        <DialogFooter className="border-t pt-4">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={loading || hasAnyErrors(errors)}
          >
            {loading ? "Saving..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
