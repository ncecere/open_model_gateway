import { useState } from "react";
import { ChevronDown, ChevronUp, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
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

import type { ProviderDetail } from "../providers";
import {
  createCustomMetadataEntry,
  metadataHasValues,
  metadataSectionsForProvider,
  type MetadataField,
} from "../metadata";
import type { CustomMetadataEntry } from "../types";

type ModelMetadataEditorProps = {
  providerDetail: ProviderDetail;
  metadata: Record<string, string>;
  customMetadata: CustomMetadataEntry[];
  onMetadataChange: (key: string, value: string) => void;
  onCustomMetadataChange: (next: CustomMetadataEntry[]) => void;
};

export function ModelMetadataEditor({
  providerDetail,
  metadata,
  customMetadata,
  onMetadataChange,
  onCustomMetadataChange,
}: ModelMetadataEditorProps) {
  const sections = metadataSectionsForProvider(providerDetail.value);
  const hasValues = metadataHasValues(metadata, customMetadata);
  const [expanded, setExpanded] = useState(hasValues);

  const handleAddCustomRow = () => {
    onCustomMetadataChange([...customMetadata, createCustomMetadataEntry()]);
  };

  const handleCustomKeyChange = (id: string, value: string) => {
    onCustomMetadataChange(
      customMetadata.map((entry) =>
        entry.id === id ? { ...entry, key: value } : entry,
      ),
    );
  };

  const handleCustomValueChange = (id: string, value: string) => {
    onCustomMetadataChange(
      customMetadata.map((entry) =>
        entry.id === id ? { ...entry, value } : entry,
      ),
    );
  };

  const handleRemoveCustomRow = (id: string) => {
    onCustomMetadataChange(customMetadata.filter((entry) => entry.id !== id));
  };

  return (
    <div className="rounded-md border p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium">Metadata overrides</p>
          <div className="text-xs text-muted-foreground">
            <p>Provider-specific hints plus routing/audio/image pricing knobs.</p>
          </div>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => setExpanded((prev) => !prev)}
        >
          {expanded ? (
            <>
              <ChevronUp className="mr-2 h-4 w-4" />
              Hide metadata
            </>
          ) : (
            <>
              <ChevronDown className="mr-2 h-4 w-4" />
              Show metadata
            </>
          )}
        </Button>
      </div>

      {expanded ? (
        <div className="space-y-4">
          {sections.map((section) => (
            <div key={section.id} className="space-y-3 rounded-md border p-4">
              <div>
                <p className="text-sm font-medium">{section.title}</p>
                {section.description ? (
                  <p className="text-xs text-muted-foreground">
                    {section.description}
                  </p>
                ) : null}
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                {section.fields.map((field) => (
                  <MetadataFieldControl
                    key={field.key}
                    field={field}
                    value={metadata[field.key] ?? ""}
                    onChange={(value) => onMetadataChange(field.key, value)}
                  />
                ))}
              </div>
            </div>
          ))}

          <div className="space-y-3 rounded-md border border-dashed p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p className="text-sm font-medium">Advanced metadata</p>
                <p className="text-xs text-muted-foreground">
                  Key/value pairs not covered above.
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleAddCustomRow}
              >
                <Plus className="mr-2 h-4 w-4" />
                Add row
              </Button>
            </div>

            {customMetadata.length === 0 ? (
              <p className="text-xs text-muted-foreground">
                No custom metadata configured.
              </p>
            ) : (
              <div className="space-y-3">
                {customMetadata.map((entry) => (
                  <div
                    key={entry.id}
                    className="grid gap-2 sm:grid-cols-[1fr_1fr_auto]"
                  >
                    <Input
                      value={entry.key}
                      onChange={(event) =>
                        handleCustomKeyChange(entry.id, event.target.value)
                      }
                      placeholder="key"
                    />
                    <Input
                      value={entry.value}
                      onChange={(event) =>
                        handleCustomValueChange(entry.id, event.target.value)
                      }
                      placeholder="value"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => handleRemoveCustomRow(entry.id)}
                    >
                      <Trash2 className="h-4 w-4" />
                      <span className="sr-only">Remove row</span>
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      ) : (
        <p className="text-xs text-muted-foreground">
          {hasValues
            ? "Metadata overrides configured. Expand to review or edit them."
            : "Metadata overrides are hidden. Expand to configure routing, pricing, and provider-specific hints."}
        </p>
      )}
    </div>
  );
}

function MetadataFieldControl({
  field,
  value,
  onChange,
}: {
  field: MetadataField;
  value: string;
  onChange: (value: string) => void;
}) {
  const description = field.description ? (
    <p className="text-xs text-muted-foreground">{field.description}</p>
  ) : null;

  if (field.input === "select" && field.options) {
    return (
      <div className="space-y-2">
        <Label>{field.label}</Label>
        <Select value={value} onValueChange={onChange}>
          <SelectTrigger>
            <SelectValue placeholder={field.placeholder ?? "Select value"} />
          </SelectTrigger>
          <SelectContent>
            {field.options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {description}
      </div>
    );
  }

  if (field.input === "textarea") {
    return (
      <div className="space-y-2 sm:col-span-2">
        <Label>{field.label}</Label>
        <Textarea
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={field.placeholder}
          rows={3}
        />
        {description}
      </div>
    );
  }

  if (field.input === "boolean") {
    return (
      <div className="space-y-2">
        <Label>{field.label}</Label>
        <div className="flex items-center gap-3 rounded-md border px-3 py-2">
          <Switch
            checked={value === "true"}
            onCheckedChange={(checked) => onChange(checked ? "true" : "")}
          />
          {description}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <Label>{field.label}</Label>
      <Input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={field.placeholder}
        inputMode={field.input === "number" ? "decimal" : undefined}
      />
      {description}
    </div>
  );
}
