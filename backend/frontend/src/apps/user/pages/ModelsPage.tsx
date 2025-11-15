import { useMemo } from "react";
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
import { formatModelTypeLabel } from "@/features/models/types";
import { listUserModels, type UserModel } from "@/api/user/models";
import { useTheme } from "@/providers/ThemeProvider";
import { getProviderIcon } from "@/features/models/provider-icons";

const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 4,
});

const throughputFormatter = new Intl.NumberFormat(undefined, {
  maximumFractionDigits: 1,
});

export function UserModelsPage() {
  const modelsQuery = useQuery({
    queryKey: ["user-models"],
    queryFn: listUserModels,
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
          No models available. Contact your administrator if this seems unexpected.
        </p>
      );
    }
    return <ModelTable models={models} theme={resolvedTheme} />;
  }, [modelsQuery.isLoading, models, resolvedTheme]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Model catalog</h1>
        <p className="text-sm text-muted-foreground">
          Review pricing and recent performance for the models currently exposed to your API keys.
        </p>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>Catalog overview</CardTitle>
          <p className="text-sm text-muted-foreground">
            Throughput and latency reflect activity over the past 24 hours.
          </p>
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
                  {currencyFormatter.format(model.price_input)} / 1M input tokens
                </span>
                <span>
                  {currencyFormatter.format(model.price_output)} / 1M output tokens
                </span>
              </div>
            </TableCell>
              <TableCell>
                <Badge variant="outline">{formatModelTypeLabel(model.model_type)}</Badge>
              </TableCell>
            <TableCell className="text-sm">
              {formatThroughput(model.throughput_tokens_per_second)}
            </TableCell>
            <TableCell className="text-sm">
              {formatLatency(model.avg_latency_ms)}
            </TableCell>
              <TableCell>
                <Badge className={statusClassName(model.status)}>
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

function statusClassName(status: string) {
  switch (status) {
    case "online":
      return "bg-emerald-500 text-white hover:bg-emerald-500";
    case "degraded":
      return "bg-amber-500/80 text-black hover:bg-amber-500";
    case "offline":
    case "disabled":
      return "bg-destructive text-destructive-foreground hover:bg-destructive";
    default:
      return "bg-muted text-foreground";
  }
}

function formatStatusLabel(status: string) {
  if (!status) {
    return "Unknown";
  }
  return status.charAt(0).toUpperCase() + status.slice(1);
}
