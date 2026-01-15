import { type ChangeEvent, useRef } from "react";

import type { VertexProviderConfig } from "@/api/model-catalog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useToast } from "@/hooks/use-toast";

import { defaultVertexOverride, type ModelFormState } from "../../types";
import { normalizeProviderSlug } from "@/api/model-catalog";
import { DEFAULT_PROVIDER_DETAIL, PROVIDER_DETAILS } from "../../providers";

interface ProviderConfigTabProps {
  form: ModelFormState;
  onChange: (form: ModelFormState) => void;
}

export function ProviderConfigTab({ form, onChange }: ProviderConfigTabProps) {
  const providerKey = normalizeProviderSlug(form.provider);
  const providerDetail =
    PROVIDER_DETAILS[providerKey] ?? DEFAULT_PROVIDER_DETAIL;
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

  return (
    <div className="grid gap-4">
      <div className="grid gap-4 sm:grid-cols-2">
        {providerConfig.showDeployment && (
          <div className="space-y-2">
            <Label htmlFor="deployment">Deployment</Label>
            <Input
              id="deployment"
              value={form.deployment}
              onChange={(event) =>
                onChange({ ...form, deployment: event.target.value })
              }
              placeholder="gpt-4o-deployment"
            />
            <p className="text-xs text-muted-foreground">
              Optional; defaults to provider model when left blank.
            </p>
          </div>
        )}

        {providerKeyInline && (
          <div className="space-y-2">
            <Label htmlFor="api_key_inline">Provider key</Label>
            <Input
              id="api_key_inline"
              type="password"
              value={form.api_key}
              onChange={(event) =>
                onChange({ ...form, api_key: event.target.value })
              }
              placeholder="secret"
            />
          </div>
        )}
      </div>

      {(providerConfig.showEndpoint ||
        (providerConfig.showApiKey && !providerKeyInline)) && (
        <div className="grid gap-4 sm:grid-cols-2">
          {providerConfig.showEndpoint && (
            <div className="space-y-2">
              <Label htmlFor="endpoint">Endpoint</Label>
              <Input
                id="endpoint"
                value={form.endpoint}
                onChange={(event) =>
                  onChange({ ...form, endpoint: event.target.value })
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
                  onChange({ ...form, api_key: event.target.value })
                }
                placeholder="secret"
              />
            </div>
          )}
        </div>
      )}

      {providerConfig.showApiVersion && (
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="api_version">API version</Label>
            <Input
              id="api_version"
              value={form.api_version}
              onChange={(event) =>
                onChange({ ...form, api_version: event.target.value })
              }
              placeholder="2024-07-01-preview"
            />
          </div>
        </div>
      )}

      {form.provider === "vertex" && (
        <VertexConfigFields
          value={vertexOverride}
          onChange={setVertexOverride}
        />
      )}

      {!providerConfig.showDeployment &&
        !providerConfig.showEndpoint &&
        !providerConfig.showApiKey &&
        !providerConfig.showApiVersion &&
        form.provider !== "vertex" && (
          <div className="rounded-md border border-dashed p-6 text-center">
            <p className="text-sm text-muted-foreground">
              No additional configuration required for {form.provider || "this provider"}.
            </p>
          </div>
        )}
    </div>
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

      <div className="grid gap-4 sm:grid-cols-2">
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
