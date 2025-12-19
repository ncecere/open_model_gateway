import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { PricingTiersFormState } from "../types";

const UNIT_OPTIONS = [
  { value: "tokens_per_million", label: "Tokens / 1M" },
  { value: "tokens_per_thousand", label: "Tokens / 1k" },
  { value: "per_image", label: "Per image" },
  { value: "per_megapixel", label: "Per megapixel" },
  { value: "per_minute", label: "Per minute" },
  { value: "per_second", label: "Per second" },
  { value: "per_million_characters", label: "Per 1M characters" },
];

const BUCKET_SUGGESTIONS = [
  "input",
  "cache",
  "output",
  "image",
  "audio",
  "tts",
  "video",
  "embedding",
];

interface PricingTiersEditorProps {
  value: PricingTiersFormState;
  onChange: (value: PricingTiersFormState) => void;
}

export function PricingTiersEditor({ value, onChange }: PricingTiersEditorProps) {
  const buckets = Object.keys(value ?? {});

  const handleAddBucket = () => {
    const suggestion = BUCKET_SUGGESTIONS.find((bucket) => !(bucket in value));
    const initial = suggestion ?? "custom";
    // eslint-disable-next-line no-alert
    const name = window.prompt("Bucket name (e.g., input, output, image)", initial);
    const normalized = name?.trim().toLowerCase();
    if (!normalized) {
      return;
    }
    if (value[normalized]) {
      return;
    }
    onChange({
      ...value,
      [normalized]: [],
    });
  };

  const handleRemoveBucket = (bucket: string) => {
    const next = { ...value };
    delete next[bucket];
    onChange(next);
  };

  const handleAddTier = (bucket: string) => {
    const tier = {
      id: createTierId(),
      unit: defaultUnitForBucket(bucket),
      price_per_unit: "",
      max_units: "",
      metadata: "",
    };
    onChange({
      ...value,
      [bucket]: [...(value[bucket] ?? []), tier],
    });
  };

  const handleRemoveTier = (bucket: string, tierId: string) => {
    const tiers = value[bucket]?.filter((tier) => tier.id !== tierId) ?? [];
    onChange({
      ...value,
      [bucket]: tiers,
    });
  };

  const handleTierFieldChange = (
    bucket: string,
    tierId: string,
    field: "unit" | "price_per_unit" | "max_units" | "metadata",
    nextValue: string,
  ) => {
    const tiers = value[bucket] ?? [];
    onChange({
      ...value,
      [bucket]: tiers.map((tier) =>
        tier.id === tierId
          ? {
              ...tier,
              [field]: nextValue,
            }
          : tier,
      ),
    });
  };

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <Label>Pricing tiers</Label>
        <p className="text-sm text-muted-foreground">
          Define modality-specific buckets (e.g., input, output, image) and stack
          per-bucket tiers top-to-bottom. Add metadata like quality or
          operation=stt to match runtime overrides.
        </p>
      </div>

      {buckets.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No buckets defined yet. Add a bucket to start modeling pricing.
        </p>
      ) : null}

      {buckets.map((bucket) => (
        <div key={bucket} className="space-y-4 rounded-md border bg-card p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="font-medium capitalize">{bucket}</p>
              <p className="text-xs text-muted-foreground">
                Add one or more tiers for {bucket} usage. Max units are optional.
              </p>
            </div>
            <Button
              variant="ghost"
              size="sm"
              className="text-destructive hover:text-destructive"
              onClick={() => handleRemoveBucket(bucket)}
            >
              Remove bucket
            </Button>
          </div>

          {(value[bucket] ?? []).length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No tiers yet. Add a tier to set the first price bracket.
            </p>
          ) : null}

          {(value[bucket] ?? []).map((tier, index) => (
            <div
              key={tier.id}
              className="space-y-4 rounded-md border bg-background p-4"
            >
              <div className="flex items-center justify-between text-sm font-medium">
                <span>Tier {index + 1}</span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="text-destructive hover:text-destructive"
                  onClick={() => handleRemoveTier(bucket, tier.id)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>

              <div className="space-y-2">
                <Label>Unit</Label>
                <Select
                  value={tier.unit}
                  onValueChange={(next) =>
                    handleTierFieldChange(bucket, tier.id, "unit", next)
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="tokens_per_million" />
                  </SelectTrigger>
                  <SelectContent>
                    {UNIT_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label>Price per unit (USD)</Label>
                  <Input
                    value={tier.price_per_unit}
                    onChange={(event) =>
                      handleTierFieldChange(
                        bucket,
                        tier.id,
                        "price_per_unit",
                        event.target.value,
                      )
                    }
                    inputMode="decimal"
                    placeholder="0.05"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Max units (optional)</Label>
                  <Input
                    value={tier.max_units}
                    onChange={(event) =>
                      handleTierFieldChange(
                        bucket,
                        tier.id,
                        "max_units",
                        event.target.value,
                      )
                    }
                    inputMode="decimal"
                    placeholder="e.g. 200000"
                  />
                  <p className="text-xs text-muted-foreground">
                    Leave blank for unlimited.
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <Label>Metadata (key=value per line)</Label>
                <Textarea
                  value={tier.metadata}
                  onChange={(event) =>
                    handleTierFieldChange(
                      bucket,
                      tier.id,
                      "metadata",
                      event.target.value,
                    )
                  }
                  rows={4}
                  placeholder={"quality=hd\noperation=stt"}
                />
                <p className="text-xs text-muted-foreground">
                  Examples: quality=hd, operation=stt, channel=audio_output.
                </p>
              </div>
            </div>
          ))}

          <Button variant="outline" size="sm" onClick={() => handleAddTier(bucket)}>
            Add tier
          </Button>
        </div>
      ))}

      <div>
        <Button variant="secondary" onClick={handleAddBucket}>
          Add bucket
        </Button>
      </div>
    </div>
  );
}

function defaultUnitForBucket(bucket: string) {
  switch (bucket) {
    case "input":
    case "output":
    case "cache":
    case "embedding":
      return "tokens_per_million";
    case "image":
      return "per_image";
    case "audio":
      return "per_minute";
    case "tts":
      return "per_million_characters";
    case "video":
      return "per_second";
    default:
      return "tokens_per_million";
  }
}

function createTierId() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2);
}
