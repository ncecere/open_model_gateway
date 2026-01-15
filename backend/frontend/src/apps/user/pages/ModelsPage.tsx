import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

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
  useEffect(() => {
    if (hasManualScope || !scopeOptions.length) {
      return;
    }
    // Prefer personal scope when available so users start on their default tenant.
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
    return <ModelTable models={models} theme={resolvedTheme} />;
  }, [modelsQuery.isLoading, models, resolvedTheme]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Model Catalog"
        description="Review pricing and recent performance for the models currently exposed to your API keys."
      />
      <Card>
        <CardHeader>
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
          <TableHead>Alias</TableHead>
          <TableHead>Pricing</TableHead>
          <TableHead>Model type</TableHead>
          <TableHead>Throughput</TableHead>
          <TableHead>Latency</TableHead>
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
              <TableCell className="text-sm">
                <div className="flex flex-col">
                  <span>
                    {currencyFormatter.format(model.price_input)} / 1M input
                    tokens
                  </span>
                  <span>
                    {currencyFormatter.format(model.price_output)} / 1M output
                    tokens
                  </span>
                </div>
              </TableCell>
              <TableCell>
                <Badge variant="outline">
                  {formatModelTypeLabel(model.model_type)}
                </Badge>
              </TableCell>
              <TableCell className="text-sm">
                {formatThroughput(model.throughput_tokens_per_second)}
              </TableCell>
              <TableCell className="text-sm">
                {formatLatency(model.avg_latency_ms)}
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
    return `${(ms / 1000).toFixed(2)} s`;
  }
  return `${ms.toFixed(0)} ms`;
}

function formatStatusLabel(status: string) {
  if (!status) {
    return "Unknown";
  }
  return status.charAt(0).toUpperCase() + status.slice(1);
}
