import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Copy, Pencil, Trash2, MoreHorizontal } from "lucide-react";
import type { ModelCatalogEntry, PricingTier } from "@/api/model-catalog";
import { formatModelTypeLabel } from "../types";
import { getProviderIcon } from "@/features/models/provider-icons";
import { statusToneClass, toneFromStatus } from "@/ui/kit/status";

const currency = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 4,
});

type ModelTableProps = {
  models: ModelCatalogEntry[];
  isLoading: boolean;
  hasAnyModels: boolean;
  onEdit: (model: ModelCatalogEntry) => void;
  onClone: (model: ModelCatalogEntry) => void;
  onDelete: (model: ModelCatalogEntry) => void;
  statuses?: Record<string, string>;
  theme: "light" | "dark";
  selectedAliases?: Set<string>;
  onSelectionChange?: (aliases: Set<string>) => void;
};

export function ModelTable({
  models,
  isLoading,
  hasAnyModels,
  onEdit,
  onClone,
  onDelete,
  statuses,
  theme,
  selectedAliases = new Set(),
  onSelectionChange,
}: ModelTableProps) {
  const allSelected = models.length > 0 && models.every((m) => selectedAliases.has(m.alias));
  const someSelected = models.some((m) => selectedAliases.has(m.alias));

  const toggleAll = () => {
    if (!onSelectionChange) return;
    if (allSelected) {
      // Deselect all visible
      const next = new Set(selectedAliases);
      models.forEach((m) => next.delete(m.alias));
      onSelectionChange(next);
    } else {
      // Select all visible
      const next = new Set(selectedAliases);
      models.forEach((m) => next.add(m.alias));
      onSelectionChange(next);
    }
  };

  const toggleOne = (alias: string) => {
    if (!onSelectionChange) return;
    const next = new Set(selectedAliases);
    if (next.has(alias)) {
      next.delete(alias);
    } else {
      next.add(alias);
    }
    onSelectionChange(next);
  };
  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
    );
  }

  if (!hasAnyModels) {
    return (
      <p className="text-sm text-muted-foreground">
        No models configured yet. Add a catalog entry to expose a provider alias
        to clients.
      </p>
    );
  }

  if (models.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No models match the current search or filters.
      </p>
    );
  }

  const showSelection = Boolean(onSelectionChange);

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {showSelection && (
            <TableHead className="w-12">
              <Checkbox
                checked={allSelected}
                onCheckedChange={toggleAll}
                aria-label="Select all"
                className={someSelected && !allSelected ? "opacity-50" : ""}
              />
            </TableHead>
          )}
          <TableHead>Alias</TableHead>
          <TableHead>Provider model</TableHead>
          <TableHead>Deployment</TableHead>
          <TableHead>Pricing</TableHead>
          <TableHead>Model type</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="w-12 text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {models.map((model) => {
          const status = statuses?.[model.alias]
            ?? (model.enabled ? "unknown" : "disabled");
          const icon = getProviderIcon(model.provider, theme);
          const tierBuckets = summarizePricingBuckets(model.pricing_tiers);
          const tiersToShow = tierBuckets.slice(0, 3);
          const isSelected = selectedAliases.has(model.alias);
          return (
            <TableRow key={model.alias} className={isSelected ? "bg-muted/50" : undefined}>
              {showSelection && (
                <TableCell>
                  <Checkbox
                    checked={isSelected}
                    onCheckedChange={() => toggleOne(model.alias)}
                    aria-label={`Select ${model.alias}`}
                  />
                </TableCell>
              )}
              <TableCell className="font-medium">
                <div className="flex items-center gap-3">
                  {icon ? (
                    <img
                      src={icon}
                      alt=""
                      className="h-8 w-8 rounded bg-muted/40 p-1"
                    />
                  ) : null}
                  <div className="flex flex-col">
                    <span>{model.alias}</span>
                    <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                      <span>{model.provider}</span>
                      {model.tenant_assignable ? (
                        <Badge variant="secondary">Tenant</Badge>
                      ) : null}
                      {model.managed_by === "config" ? (
                        <Badge variant="outline" className="text-[10px] border-amber-500/50 text-amber-600 dark:text-amber-400">
                          Config
                        </Badge>
                      ) : null}
                    </div>
                  </div>
                </div>
              </TableCell>
            <TableCell>
              <div className="flex flex-col">
                <span>{model.provider}</span>
                <span className="text-xs text-muted-foreground">
                  {model.provider_model}
                </span>
              </div>
            </TableCell>
            <TableCell>
              <div className="flex flex-col">
                <span>{model.deployment}</span>
                <span className="text-xs text-muted-foreground">
                  {model.region || "n/a"}
                </span>
              </div>
            </TableCell>
            <TableCell className="text-sm">
              {tiersToShow.length > 0 ? (
                <div className="flex flex-col gap-1">
                  {tiersToShow.map(({ bucket, tiers }) => (
                    <span key={`${model.alias}-${bucket}`}>
                      {formatBucketSummary(bucket, tiers)}
                    </span>
                  ))}
                  {tierBuckets.length > tiersToShow.length ? (
                    <span className="text-xs text-muted-foreground">
                      +{tierBuckets.length - tiersToShow.length} more bucket
                      {tierBuckets.length - tiersToShow.length === 1 ? "" : "s"}
                    </span>
                  ) : null}
                </div>
              ) : (
                <div className="text-sm text-muted-foreground">
                  No pricing tiers configured
                </div>
              )}
            </TableCell>
            <TableCell className="text-sm">
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant="secondary">
                  {formatModelTypeLabel(model.model_type)}
                </Badge>
                {audioChipForType(model.model_type) && (
                  <Badge variant="outline">
                    {audioChipForType(model.model_type)}
                  </Badge>
                )}
              </div>
            </TableCell>
              <TableCell>
                <Badge className={statusToneClass(toneFromStatus(status))}>
                  {formatStatusLabel(status)}
                </Badge>
              </TableCell>
            <TableCell className="text-right">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon">
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel>Actions</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onSelect={() => onEdit(model)}>
                    <Pencil className="mr-2 h-4 w-4" /> Edit
                  </DropdownMenuItem>
                  <DropdownMenuItem onSelect={() => onClone(model)}>
                    <Copy className="mr-2 h-4 w-4" /> Clone
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    className="text-destructive focus:text-destructive"
                    onSelect={() => onDelete(model)}
                  >
                    <Trash2 className="mr-2 h-4 w-4" /> Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
        );
        })}
      </TableBody>
    </Table>
  );
}

function formatStatusLabel(status: string) {
  if (!status) {
    return "Unknown";
  }
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function audioChipForType(modelType?: string | null) {
  if (!modelType) {
    return null;
  }
  switch (modelType.toLowerCase()) {
    case "audio_transcription":
      return "Audio · STT";
    case "audio_speech":
      return "Audio · TTS";
    default:
      return null;
  }
}

function summarizePricingBuckets(
  pricing?: Record<string, PricingTier[]>,
) {
  if (!pricing) {
    return [] as { bucket: string; tiers: PricingTier[] }[];
  }
  return Object.entries(pricing)
    .map(([bucket, tiers]) => ({ bucket, tiers: tiers ?? [] }))
    .filter((entry) => entry.tiers.length > 0);
}

function formatBucketSummary(bucket: string, tiers: PricingTier[]) {
  const [first] = tiers;
  if (!first) {
    return `${bucket}: n/a`;
  }
  const price = currency.format(first.price_per_unit ?? 0);
  const unitLabel = formatUnitLabel(first.unit);
  const scope = first.max_units
    ? `≤${Number(first.max_units).toLocaleString()} `
    : "";
  const metadataLabel =
    first.metadata?.quality ??
    first.metadata?.operation ??
    first.metadata?.channel ??
    first.metadata?.resolution;
  const summary = `${scope}${price} / ${unitLabel}`.trim();
  const annotated = metadataLabel ? `${summary} (${metadataLabel})` : summary;
  const suffix = tiers.length > 1 ? ` (+${tiers.length - 1} tiers)` : "";
  return `${bucket}: ${annotated}${suffix}`;
}

function formatUnitLabel(unit?: string) {
  switch (unit) {
    case "tokens_per_million":
      return "tokens per 1M";
    case "tokens_per_thousand":
      return "tokens per 1k";
    case "per_image":
      return "image";
    case "per_megapixel":
      return "megapixel";
    case "per_minute":
      return "minute";
    case "per_second":
      return "second";
    case "per_million_characters":
      return "characters per 1M";
    default:
      return unit ?? "unit";
  }
}
