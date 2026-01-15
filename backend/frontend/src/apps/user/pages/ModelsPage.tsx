import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, X } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { formatModelTypeLabel } from "@/features/models/types";
import { listUserModels, type UserModel } from "@/api/user/models";
import { useTheme } from "@/providers/ThemeProvider";
import { getProviderIcon } from "@/features/models/provider-icons";
import { statusToneClass, toneFromStatus } from "@/ui/kit/status";
import { useUserTenantsQuery } from "../hooks/useUserData";
import { PageHeader } from "@/components/layouts";
import type { PricingTier } from "@/api/model-catalog";

const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 4,
});

const throughputFormatter = new Intl.NumberFormat(undefined, {
  maximumFractionDigits: 1,
});

export function UserModelsPage() {
  const tenantsQuery = useUserTenantsQuery();
  const scopeOptions = useMemo(() => {
    const tenants = tenantsQuery.data ?? [];
    const baseOptions = tenants
      .map((tenant) => ({
        value: tenant.is_personal ? "personal" : tenant.tenant_id,
        label: tenant.is_personal
          ? "Personal"
          : tenant.name?.trim().length
            ? tenant.name
            : tenant.tenant_id,
      }))
      .sort((a, b) => {
        if (a.value === "personal") {
          return -1;
        }
        if (b.value === "personal") {
          return 1;
        }
        return a.label.localeCompare(b.label);
      });
    const unique: { value: string; label: string }[] = [];
    const seen = new Set<string>();
    for (const option of baseOptions) {
      if (seen.has(option.value)) {
        continue;
      }
      seen.add(option.value);
      unique.push(option);
    }
    return [{ value: "all", label: "All tenants" }, ...unique];
  }, [tenantsQuery.data]);

  const [selectedScope, setSelectedScope] = useState<string>("all");
  const [hasManualScope, setHasManualScope] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [providerFilter, setProviderFilter] = useState("all");
  const [typeFilter, setTypeFilter] = useState("all");

  useEffect(() => {
    if (hasManualScope || !scopeOptions.length) {
      return;
    }
    const preferred =
      scopeOptions.find((option) => option.value === "personal") ??
      scopeOptions.find((option) => option.value !== "all");
    if (preferred && selectedScope !== preferred.value) {
      setSelectedScope(preferred.value);
    }
  }, [hasManualScope, scopeOptions, selectedScope]);

  const handleScopeChange = (value: string) => {
    setSelectedScope(value);
    setHasManualScope(true);
  };

  const selectedScopeLabel =
    scopeOptions.find((option) => option.value === selectedScope)?.label ??
    "All tenants";

  const modelsQuery = useQuery({
    queryKey: ["user-models", selectedScope],
    queryFn: () =>
      listUserModels(selectedScope === "all" ? undefined : selectedScope),
  });
  const { resolvedTheme } = useTheme();

  const models = modelsQuery.data ?? [];

  // Extract unique providers and types for filters
  const providerOptions = useMemo(() => {
    const unique = Array.from(new Set(models.map((m) => m.provider)));
    return unique.sort((a, b) => a.localeCompare(b));
  }, [models]);

  const typeOptions = useMemo(() => {
    const unique = Array.from(new Set(models.map((m) => m.model_type || "llm")));
    return unique.sort((a, b) => a.localeCompare(b));
  }, [models]);

  // Filter models
  const filteredModels = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return models.filter((model) => {
      const matchesTerm =
        !term ||
        model.alias.toLowerCase().includes(term) ||
        model.provider.toLowerCase().includes(term);
      const matchesProvider =
        providerFilter === "all" || model.provider === providerFilter;
      const matchesType =
        typeFilter === "all" || (model.model_type || "llm") === typeFilter;
      return matchesTerm && matchesProvider && matchesType;
    });
  }, [models, searchTerm, providerFilter, typeFilter]);

  const hasActiveFilters =
    searchTerm.trim() !== "" ||
    providerFilter !== "all" ||
    typeFilter !== "all";

  const clearFilters = () => {
    setSearchTerm("");
    setProviderFilter("all");
    setTypeFilter("all");
  };

  const content = useMemo(() => {
    if (modelsQuery.isLoading) {
      return <Skeleton className="h-64 w-full" />;
    }
    if (!models.length) {
      return (
        <p className="text-sm text-muted-foreground">
          No models available. Contact your administrator if this seems
          unexpected.
        </p>
      );
    }
    if (!filteredModels.length) {
      return (
        <div className="flex flex-col items-center gap-2 py-8">
          <p className="text-sm text-muted-foreground">
            No models match your search or filters.
          </p>
          <Button variant="outline" size="sm" onClick={clearFilters}>
            Clear filters
          </Button>
        </div>
      );
    }
    return <ModelTable models={filteredModels} theme={resolvedTheme} />;
  }, [modelsQuery.isLoading, models, filteredModels, resolvedTheme]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Model Catalog"
        description="Review pricing and recent performance for the models currently exposed to your API keys."
      />
      <Card>
        <CardHeader>
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
              <div>
                <CardTitle>Catalog overview</CardTitle>
                <p className="text-sm text-muted-foreground">
                  Throughput and latency reflect activity over the past 24 hours.
                </p>
                <p className="text-xs text-muted-foreground">
                  Showing models available to: {selectedScopeLabel}.
                </p>
              </div>
              <Select
                value={selectedScope}
                onValueChange={handleScopeChange}
                disabled={!scopeOptions.length}
              >
                <SelectTrigger
                  id="user-model-scope"
                  className="w-full md:w-56 lg:w-64"
                >
                  <SelectValue placeholder="Select tenant scope" />
                </SelectTrigger>
                <SelectContent align="end">
                  {scopeOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Search and Filters */}
            <div className="flex flex-col gap-3 md:flex-row md:items-center">
              <div className="relative flex-1 md:max-w-sm">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Search models..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9"
                />
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <Select value={providerFilter} onValueChange={setProviderFilter}>
                  <SelectTrigger className="w-32">
                    <SelectValue placeholder="Provider" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All providers</SelectItem>
                    {providerOptions.map((provider) => (
                      <SelectItem key={provider} value={provider}>
                        {provider}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Select value={typeFilter} onValueChange={setTypeFilter}>
                  <SelectTrigger className="w-32">
                    <SelectValue placeholder="Type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All types</SelectItem>
                    {typeOptions.map((type) => (
                      <SelectItem key={type} value={type}>
                        {formatModelTypeLabel(type)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {hasActiveFilters && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={clearFilters}
                    className="h-9"
                  >
                    <X className="mr-1 h-4 w-4" />
                    Clear
                  </Button>
                )}
              </div>
              <div className="ml-auto text-sm text-muted-foreground">
                {filteredModels.length} of {models.length} models
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent>{content}</CardContent>
      </Card>
    </div>
  );
}

function ModelTable({
  models,
  theme,
}: {
  models: UserModel[];
  theme: "light" | "dark";
}) {
  return (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Model</TableHead>
            <TableHead>Pricing</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Performance</TableHead>
            <TableHead>Status</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {models.map((model) => {
            const icon = getProviderIcon(model.provider, theme);
            return (
              <TableRow key={model.alias}>
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
                      <span className="text-xs text-muted-foreground">
                        {model.provider}
                      </span>
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  <PricingDisplay model={model} />
                </TableCell>
                <TableCell>
                  <ModelTypeBadge type={model.model_type} />
                </TableCell>
                <TableCell>
                  <PerformanceDisplay model={model} />
                </TableCell>
                <TableCell>
                  <Badge
                    className={statusToneClass(toneFromStatus(model.status))}
                  >
                    {formatStatusLabel(model.status)}
                  </Badge>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
  );
}

function PricingDisplay({ model }: { model: UserModel }) {
  const hasTiers = model.pricing_tiers && Object.keys(model.pricing_tiers).length > 0;

  if (hasTiers) {
    const tierEntries = Object.entries(model.pricing_tiers!);
    const standardTiers = tierEntries.find(([key]) => key === "standard");
    const batchTiers = tierEntries.find(([key]) => key === "batch");

    return (
      <div className="flex flex-col gap-0.5">
        {standardTiers ? (
          <TierPriceRow tiers={standardTiers[1]} />
        ) : (
          <LegacyPriceRow model={model} />
        )}
        {batchTiers && (
          <span className="text-xs text-muted-foreground">
            + Batch pricing
          </span>
        )}
      </div>
    );
  }

  return <LegacyPriceRow model={model} />;
}

function TierPriceRow({ tiers }: { tiers: PricingTier[] }) {
  const inputTier = tiers.find((t) => t.metadata?.type === "input" || t.unit?.includes("input"));
  const outputTier = tiers.find((t) => t.metadata?.type === "output" || t.unit?.includes("output"));

  // If we can identify input/output tiers, show them
  if (inputTier && outputTier) {
    return (
      <div className="flex flex-col text-sm">
        <span>
          {currencyFormatter.format(inputTier.price_per_unit ?? 0)} / 1M input
        </span>
        <span className="text-xs text-muted-foreground">
          {currencyFormatter.format(outputTier.price_per_unit ?? 0)} / 1M output
        </span>
      </div>
    );
  }

  // Otherwise show first tier
  const first = tiers[0];
  if (first) {
    return (
      <div className="text-sm">
        {currencyFormatter.format(first.price_per_unit ?? 0)} / {formatUnitLabel(first.unit)}
      </div>
    );
  }

  return <span className="text-sm text-muted-foreground">—</span>;
}

function LegacyPriceRow({ model }: { model: UserModel }) {
  return (
    <div className="flex flex-col text-sm">
      <span>
        {currencyFormatter.format(model.price_input)} / 1M input
      </span>
      <span className="text-xs text-muted-foreground">
        {currencyFormatter.format(model.price_output)} / 1M output
      </span>
    </div>
  );
}

function ModelTypeBadge({ type }: { type: string }) {
  const audioLabel = audioChipForType(type);

  return (
    <div className="flex flex-wrap items-center gap-1">
      <Badge variant="outline">{formatModelTypeLabel(type)}</Badge>
      {audioLabel && (
        <Badge variant="secondary" className="text-xs">
          {audioLabel}
        </Badge>
      )}
    </div>
  );
}

function PerformanceDisplay({ model }: { model: UserModel }) {
  const throughput = formatThroughput(model.throughput_tokens_per_second);
  const latency = formatLatency(model.avg_latency_ms);

  if (throughput === "—" && latency === "—") {
    return <span className="text-sm text-muted-foreground">No data</span>;
  }

  return (
    <div className="flex flex-col text-sm">
      {throughput !== "—" && (
        <span>{throughput}</span>
      )}
      {latency !== "—" && (
        <span className="text-xs text-muted-foreground">{latency} avg</span>
      )}
    </div>
  );
}

function formatThroughput(value?: number) {
  if (!value || value <= 0) {
    return "—";
  }
  return `${throughputFormatter.format(value)} tok/s`;
}

function formatLatency(ms?: number) {
  if (!ms || ms <= 0) {
    return "—";
  }
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`;
  }
  return `${ms.toFixed(0)}ms`;
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
      return "STT";
    case "audio_speech":
      return "TTS";
    default:
      return null;
  }
}

function formatUnitLabel(unit?: string) {
  switch (unit) {
    case "tokens_per_million":
      return "1M tokens";
    case "tokens_per_thousand":
      return "1k tokens";
    case "per_image":
      return "image";
    case "per_megapixel":
      return "megapixel";
    case "per_minute":
      return "minute";
    case "per_second":
      return "second";
    case "per_million_characters":
      return "1M chars";
    default:
      return unit ?? "unit";
  }
}
