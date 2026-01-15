import { Checkbox } from "@/components/ui/checkbox";
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

import { MODEL_TYPE_OPTIONS, type ModelFormState } from "../../types";
import { SUPPORTED_PROVIDERS } from "../../providers";

const ALL_MODALITIES = ["text", "image", "audio", "video"] as const;

interface BasicInfoTabProps {
  form: ModelFormState;
  onChange: (form: ModelFormState) => void;
  mode: "create" | "edit";
  onProviderChange: (provider: string) => void;
  onModelTypeChange: (modelType: string) => void;
}

export function BasicInfoTab({
  form,
  onChange,
  mode,
  onProviderChange,
  onModelTypeChange,
}: BasicInfoTabProps) {
  const handleModalitiesChange = (modality: string, checked: boolean) => {
    const next = new Set(form.modalities);
    if (checked) {
      next.add(modality);
    } else {
      next.delete(modality);
    }
    onChange({ ...form, modalities: Array.from(next) });
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="alias">
            Alias <span className="text-destructive">*</span>
          </Label>
          <Input
            id="alias"
            value={form.alias}
            onChange={(event) =>
              onChange({ ...form, alias: event.target.value })
            }
            placeholder="gpt-4o"
            disabled={mode === "edit"}
            required
          />
          <p className="text-xs text-muted-foreground">
            Unique identifier for this model configuration.
          </p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="provider">
            Provider <span className="text-destructive">*</span>
          </Label>
          <Select value={form.provider} onValueChange={onProviderChange}>
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

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="provider_model">
            Provider model <span className="text-destructive">*</span>
          </Label>
          <Input
            id="provider_model"
            value={form.provider_model}
            onChange={(event) =>
              onChange({ ...form, provider_model: event.target.value })
            }
            placeholder="gpt-4o"
            required
          />
          <p className="text-xs text-muted-foreground">
            The model ID as recognized by the provider.
          </p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="model_type">Model type</Label>
          <Select value={form.model_type} onValueChange={onModelTypeChange}>
            <SelectTrigger id="model_type">
              <SelectValue placeholder="Select type" />
            </SelectTrigger>
            <SelectContent>
              {MODEL_TYPE_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="space-y-2">
        <Label>Modalities</Label>
        <div className="flex flex-wrap gap-4">
          {ALL_MODALITIES.map((modality) => (
            <label key={modality} className="flex items-center gap-2 text-sm">
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
        <p className="text-xs text-muted-foreground">
          Input/output modalities this model supports.
        </p>
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

      <div className="grid gap-4 sm:grid-cols-2">
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
        <div className="flex items-center justify-between rounded-md border p-4">
          <div>
            <Label htmlFor="tenant-assignable" className="mb-1 block">
              Tenant-assignable
            </Label>
            <p className="text-xs text-muted-foreground">
              Allow tenant admins to attach this alias.
            </p>
          </div>
          <Switch
            id="tenant-assignable"
            checked={form.tenant_assignable}
            onCheckedChange={(checked) =>
              onChange({
                ...form,
                tenant_assignable: Boolean(checked),
              })
            }
          />
        </div>
      </div>
    </div>
  );
}
