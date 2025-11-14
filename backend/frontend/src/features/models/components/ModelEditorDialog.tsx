import {
  type ChangeEvent,
  useRef,
} from "react";

import {
  type ModelCatalogUpsertRequest,
  type ProviderOverrides,
  type VertexProviderConfig,
} from "@/api/model-catalog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useToast } from "@/hooks/use-toast";

import {
  DEFAULT_PROVIDER_DETAIL,
  PROVIDER_DETAILS,
  SUPPORTED_PROVIDERS,
} from "../providers";
import { buildMetadataPayload } from "../form";
import { normalizeMetadataForProvider } from "../metadata";
import {
  defaultVertexOverride,
  type CustomMetadataEntry,
  type ModelFormState,
} from "../types";
import { ModelMetadataEditor } from "./ModelMetadataEditor";

const ALL_MODALITIES = ["text", "image", "audio", "video"] as const;

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
  const providerDetail =
    PROVIDER_DETAILS[form.provider] ?? DEFAULT_PROVIDER_DETAIL;
  const providerConfig = providerDetail.config;
  const providerKeyInline =
    providerConfig.showApiKey && !providerConfig.showDeployment;
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

  const setVertexOverride = (next: VertexProviderConfig) => {
    onChange({
      ...form,
      provider_overrides: {
        ...form.provider_overrides,
        vertex: next,
      },
    });
  };

  const handleStringChange = (key: keyof ModelFormState, value: string) => {
    if (key === "provider") {
      const { metadata, customMetadata } = normalizeMetadataForProvider(
        form.metadata,
        form.customMetadata,
        value,
      );
      onChange({
        ...form,
        provider: value,
        metadata,
        customMetadata,
      });
      return;
    }
    onChange({ ...form, [key]: value });
  };

  const handleNumericChange = (key: keyof ModelFormState, value: string) => {
    if (value === "") {
      onChange({ ...form, [key]: "" });
      return;
    }
    const parsed = Number(value);
    if (!Number.isNaN(parsed)) {
      onChange({ ...form, [key]: parsed });
    }
  };

  const handleModalitiesChange = (modality: string, checked: boolean) => {
    const next = new Set(form.modalities);
    if (checked) {
      next.add(modality);
    } else {
      next.delete(modality);
    }
    onChange({ ...form, modalities: Array.from(next) });
  };

  const requiredMissing =
    !form.alias.trim() ||
    !form.provider.trim() ||
    !form.provider_model.trim() ||
    (providerConfig.showDeployment && !form.deployment.trim());

  const handleMetadataValueChange = (key: string, value: string) => {
    const next = { ...form.metadata };
    if (value === "") {
      delete next[key];
    } else {
      next[key] = value;
    }
    onChange({ ...form, metadata: next });
  };

  const handleCustomMetadataChange = (next: CustomMetadataEntry[]) => {
    onChange({ ...form, customMetadata: next });
  };

  const handleSubmit = () => {
    if (requiredMissing) {
      return;
    }

    const resolvedDeployment = providerConfig.showDeployment
      ? form.deployment.trim()
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
      provider_overrides:
        Object.keys(provider_overrides).length > 0
          ? provider_overrides
          : undefined,
    };

    onSubmit(payload);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {mode === "create" ? "Add model" : `Edit ${form.alias}`}
          </DialogTitle>
          <DialogDescription>
            Provide deployment metadata, pricing, and routing configuration.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid gap-2 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="alias">Alias</Label>
              <Input
                id="alias"
                value={form.alias}
                onChange={(event) =>
                  handleStringChange("alias", event.target.value)
                }
                placeholder="gpt-4o"
                disabled={mode === "edit"}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="provider">Provider</Label>
              <Select
                value={form.provider}
                onValueChange={(value) => handleStringChange("provider", value)}
              >
                <SelectTrigger id="provider">
                  <SelectValue placeholder="Select provider" />
                </SelectTrigger>
                <SelectContent>
                  {SUPPORTED_PROVIDERS.map((provider) => (
                    <SelectItem key={provider.value} value={provider.value}>
                      {provider.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid gap-2 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="provider_model">Provider model</Label>
              <Input
                id="provider_model"
                value={form.provider_model}
                onChange={(event) =>
                  handleStringChange("provider_model", event.target.value)
                }
                placeholder="gpt-4o"
                required
              />
            </div>
            {providerConfig.showDeployment ? (
              <div className="space-y-2">
                <Label htmlFor="deployment">Deployment</Label>
                <Input
                  id="deployment"
                  value={form.deployment}
                  onChange={(event) =>
                    handleStringChange("deployment", event.target.value)
                  }
                  placeholder="gpt-4o-deployment"
                  required={providerConfig.showDeployment}
                />
              </div>
            ) : providerKeyInline ? (
              <div className="space-y-2">
                <Label htmlFor="api_key_inline">Provider key</Label>
                <Input
                  id="api_key_inline"
                  type="password"
                  value={form.api_key}
                  onChange={(event) =>
                    handleStringChange("api_key", event.target.value)
                  }
                  placeholder="secret"
                />
              </div>
            ) : (
              <div className="space-y-2 sm:col-span-1" />
            )}
          </div>

          {providerConfig.showEndpoint ||
          (providerConfig.showApiKey && !providerKeyInline) ? (
            <div className="grid gap-2 sm:grid-cols-2">
              {providerConfig.showEndpoint && (
                <div className="space-y-2">
                  <Label htmlFor="endpoint">Endpoint</Label>
                  <Input
                    id="endpoint"
                    value={form.endpoint}
                    onChange={(event) =>
                      handleStringChange("endpoint", event.target.value)
                    }
                    placeholder="https://your-resource.openai.azure.com"
                  />
                </div>
              )}
              {providerConfig.showApiKey && !providerKeyInline && (
                <div className="space-y-2">
                  <Label htmlFor="api_key">Provider key</Label>
                  <Input
                    id="api_key"
                    type="password"
                    value={form.api_key}
                    onChange={(event) =>
                      handleStringChange("api_key", event.target.value)
                    }
                    placeholder="secret"
                  />
                </div>
              )}
            </div>
          ) : null}

          {providerConfig.showApiVersion && (
            <div className="grid gap-2 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="api_version">API version</Label>
                <Input
                  id="api_version"
                  value={form.api_version}
                  onChange={(event) =>
                    handleStringChange("api_version", event.target.value)
                  }
                  placeholder="2024-07-01-preview"
                />
              </div>
            </div>
          )}

          {form.provider === "vertex" ? (
            <VertexConfigFields
              value={vertexOverride}
              onChange={setVertexOverride}
            />
          ) : null}

          <div className="grid gap-2 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="price_input">Price input ($)</Label>
              <Input
                id="price_input"
                value={form.price_input}
                onChange={(event) =>
                  handleStringChange("price_input", event.target.value)
                }
                inputMode="decimal"
                placeholder="0.005"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="price_output">Price output ($)</Label>
              <Input
                id="price_output"
                value={form.price_output}
                onChange={(event) =>
                  handleStringChange("price_output", event.target.value)
                }
                inputMode="decimal"
                placeholder="0.015"
              />
            </div>
          </div>

          <div className="grid gap-2 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="context_window">Context window</Label>
              <Input
                id="context_window"
                value={form.context_window}
                onChange={(event) =>
                  handleNumericChange("context_window", event.target.value)
                }
                placeholder="128000"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="max_output_tokens">Max output tokens</Label>
              <Input
                id="max_output_tokens"
                value={form.max_output_tokens}
                onChange={(event) =>
                  handleNumericChange("max_output_tokens", event.target.value)
                }
                placeholder="4096"
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label>Modalities</Label>
            <div className="flex flex-wrap gap-3">
              {ALL_MODALITIES.map((modality) => (
                <label
                  key={modality}
                  className="flex items-center gap-2 text-sm"
                >
                  <Checkbox
                    checked={form.modalities.includes(modality)}
                    onCheckedChange={(checked) =>
                      handleModalitiesChange(modality, Boolean(checked))
                    }
                  />
                  <span className="capitalize">{modality}</span>
                </label>
              ))}
            </div>
          </div>

          <ModelMetadataEditor
            providerDetail={providerDetail}
            metadata={form.metadata}
            customMetadata={form.customMetadata}
            onMetadataChange={handleMetadataValueChange}
            onCustomMetadataChange={handleCustomMetadataChange}
          />

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="weight">Routing weight</Label>
              <Input
                id="weight"
                value={form.weight}
                onChange={(event) =>
                  handleNumericChange("weight", event.target.value)
                }
                placeholder="100"
              />
            </div>
            <div className="flex items-center justify-between rounded-md border p-4">
              <div>
                <Label htmlFor="enabled" className="mb-1 block">
                  Enabled
                </Label>
                <p className="text-xs text-muted-foreground">
                  Toggle availability for this alias.
                </p>
              </div>
              <Switch
                id="enabled"
                checked={form.enabled}
                onCheckedChange={(checked) =>
                  onChange({ ...form, enabled: Boolean(checked) })
                }
              />
            </div>
          </div>

          <div className="flex items-center justify-between rounded-md border p-4">
            <div>
              <Label htmlFor="supports_tools" className="mb-1 block">
                Tool calling support
              </Label>
              <p className="text-xs text-muted-foreground">
                Enable if this model supports tool/function calls.
              </p>
            </div>
            <Switch
              id="supports_tools"
              checked={form.supports_tools}
              onCheckedChange={(checked) =>
                onChange({ ...form, supports_tools: Boolean(checked) })
              }
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={loading || requiredMissing}>
            {loading ? "Saving..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function VertexConfigFields({
  value,
  onChange,
}: {
  value: VertexProviderConfig;
  onChange: (next: VertexProviderConfig) => void;
}) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { toast } = useToast();
  const mergedValue = {
    ...defaultVertexOverride(),
    ...value,
  };

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const text = String(reader.result ?? "");
        const pretty = JSON.stringify(JSON.parse(text), null, 2);
        onChange({
          ...mergedValue,
          gcp_credentials_json: pretty,
          gcp_credentials_format: "json",
        });
        toast({
          title: "Credentials loaded",
          description: file.name,
        });
      } catch {
        toast({
          variant: "destructive",
          title: "Invalid JSON file",
          description: "Upload a valid Google service account credential.",
        });
      }
    };
    reader.readAsText(file);
    event.target.value = "";
  };

  const handleVertexInput = (
    key: keyof VertexProviderConfig,
    event: ChangeEvent<HTMLInputElement>,
  ) => {
    onChange({
      ...mergedValue,
      [key]: event.target.value,
    });
  };

  return (
    <div className="space-y-4 rounded-md border p-4">
      <div className="space-y-1">
        <p className="text-sm font-medium">Vertex provider settings</p>
        <p className="text-xs text-muted-foreground">
          Store the GCP project, location, and service-account credentials
          needed for Gemini routing. Credentials are saved encrypted on the
          backend.
        </p>
      </div>

      <div className="grid gap-2 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="vertex_project">GCP project ID</Label>
          <Input
            id="vertex_project"
            value={mergedValue.gcp_project_id ?? ""}
            onChange={(event) => handleVertexInput("gcp_project_id", event)}
            placeholder="my-project"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="vertex_location">Vertex location</Label>
          <Input
            id="vertex_location"
            value={mergedValue.vertex_location ?? ""}
            onChange={(event) => handleVertexInput("vertex_location", event)}
            placeholder="us-east1"
          />
          <p className="text-xs text-muted-foreground">
            This value also becomes the region for the model entry.
          </p>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="vertex_publisher">Publisher (optional)</Label>
        <Input
          id="vertex_publisher"
          value={mergedValue.vertex_publisher ?? ""}
          onChange={(event) => handleVertexInput("vertex_publisher", event)}
          placeholder="google"
        />
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between gap-2">
          <Label htmlFor="vertex_credentials" className="mb-0">
            Service-account JSON
          </Label>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => fileInputRef.current?.click()}
            >
              Upload JSON
            </Button>
            {mergedValue.gcp_credentials_json ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() =>
                  onChange({
                    ...mergedValue,
                    gcp_credentials_json: "",
                  })
                }
              >
                Clear
              </Button>
            ) : null}
          </div>
        </div>
        <Textarea
          id="vertex_credentials"
          value={mergedValue.gcp_credentials_json ?? ""}
          onChange={(event) =>
            onChange({
              ...mergedValue,
              gcp_credentials_json: event.target.value,
              gcp_credentials_format: "json",
            })
          }
          rows={6}
          placeholder="Paste your Google service account JSON"
        />
        <p className="text-xs text-muted-foreground">
          Upload or paste the JSON from `vertex.json`. We only support JSON
          format (base64 is set automatically when needed).
        </p>
        <input
          ref={fileInputRef}
          type="file"
          accept="application/json,.json"
          className="hidden"
          onChange={handleFileChange}
        />
      </div>
    </div>
  );
}
