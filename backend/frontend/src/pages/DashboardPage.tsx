import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { listTenants } from "@/api/tenants";
import { listModelCatalog } from "@/api/model-catalog";
import type { ModelCatalogEntry } from "@/api/model-catalog";
import { fetchHealth } from "@/api/health";
import { useUsageOverview } from "@/api/hooks/useUsage";
import { formatTokensShort } from "@/lib/numbers";
import { MetricCard } from "@/ui/kit/Cards";
import { Globe } from "lucide-react";
import PostgresIcon from "@/assets/system/postgres.svg";
import RedisIcon from "@/assets/system/redis.svg";
import AnthropicIconDark from "@/assets/providers/anthropic_dark.svg";
import AnthropicIconLight from "@/assets/providers/anthropic_light.svg";
import BedrockIcon from "@/assets/providers/bedrock.svg";
import VertexIcon from "@/assets/providers/vertexai.svg";
import AzureIcon from "@/assets/providers/azure.svg";
import OpenAIIconLight from "@/assets/providers/openai_light.svg";
import OpenAIIconDark from "@/assets/providers/openai_dark.svg";
import OpenAICompatIcon from "@/assets/providers/openai_compatable.svg";
import { useTheme } from "@/providers/ThemeProvider";

const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
});

const providerIcons: Record<
  string,
  { light: string; dark: string }
> = {
  anthropic: {
    light: AnthropicIconLight,
    dark: AnthropicIconDark,
  },
  bedrock: {
    light: BedrockIcon,
    dark: BedrockIcon,
  },
  vertex: {
    light: VertexIcon,
    dark: VertexIcon,
  },
  azure: {
    light: AzureIcon,
    dark: AzureIcon,
  },
  openai: {
    light: OpenAIIconLight,
    dark: OpenAIIconDark,
  },
  "openai-compatible": {
    light: OpenAICompatIcon,
    dark: OpenAICompatIcon,
  },
  openai_compatible: {
    light: OpenAICompatIcon,
    dark: OpenAICompatIcon,
  },
};

const formatSpendAmount = (usd?: number, cents?: number) =>
  currencyFormatter.format(
    typeof usd === "number"
      ? usd
      : typeof cents === "number"
        ? cents / 100
        : 0,
  );

export function DashboardPage() {
  const { resolvedTheme } = useTheme();
  const tenantsQuery = useQuery({
    queryKey: ["tenants", "dashboard"],
    queryFn: () => listTenants({ limit: 50 }),
  });

  const modelsQuery = useQuery({
    queryKey: ["model-catalog"],
    queryFn: listModelCatalog,
  });

  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    staleTime: 30_000,
  });

  const usageQuery = useUsageOverview({ period: "7d" });
  const activeTenants =
    tenantsQuery.data?.tenants.filter((t) => t.status === "active").length ?? 0;
  const totalTenants = tenantsQuery.data?.tenants.length ?? 0;
  const enabledModels =
    modelsQuery.data?.filter((model) => model.enabled).length ?? 0;
  const totalModels = modelsQuery.data?.length ?? 0;
  const usageSummary = usageQuery.data;

  return (
    <div className="space-y-6">
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          title="Active tenants"
          value={activeTenants}
          secondary={`${totalTenants} total`}
          loading={tenantsQuery.isLoading}
        />
        <MetricCard
          title="Models online"
          value={enabledModels}
          secondary={`${totalModels} configured`}
          loading={modelsQuery.isLoading}
        />
        <MetricCard
          title="Weekly requests"
          value={usageSummary?.total_requests ?? 0}
          secondary={`${((usageSummary?.total_requests ?? 0) / 7).toFixed(0)} avg / day`}
          loading={usageQuery.isLoading}
        />
        <MetricCard
          title="Weekly spend"
          value={
            usageSummary
              ? formatSpendAmount(
                  usageSummary.total_cost_usd,
                  usageSummary.total_cost_cents,
                )
              : "—"
          }
          secondary={`${formatTokensShort(
            usageSummary?.total_tokens ?? 0,
          )} tokens`}
          loading={usageQuery.isLoading}
        />
      </section>

      <section className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Gateway health</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {healthQuery.isLoading ? (
              <Skeleton className="h-24 w-full" />
            ) : (
              <div className="space-y-4">
                <div className="flex items-center justify-between gap-2 text-muted-foreground">
                  <div className="flex items-center gap-2 font-medium text-foreground">
                    <Globe className="h-8 w-8" aria-hidden />
                    <span>Global status</span>
                  </div>
                  <Badge
                    variant={
                      healthQuery.data?.status === "ok"
                        ? "secondary"
                        : "destructive"
                    }
                  >
                    {healthQuery.data?.status ?? "unknown"}
                  </Badge>
                </div>
                <div className="space-y-3">
                  <HealthStatusRow
                    label="Postgres"
                    icon={<img src={PostgresIcon} alt="" className="h-8 w-8" />}
                    check={healthQuery.data?.checks?.postgres}
                  />
                  <HealthStatusRow
                    label="Redis"
                    icon={<img src={RedisIcon} alt="" className="h-8 w-8" />}
                    check={healthQuery.data?.checks?.redis}
                  />
                </div>
                <p className="text-xs text-muted-foreground">
                  Last checked: {" "}
                  {new Date(healthQuery.dataUpdatedAt).toLocaleTimeString()}
                </p>
              </div>
            )}
          </CardContent>
        </Card>
        <ModelHealthCard
          models={modelsQuery.data ?? []}
          loading={modelsQuery.isLoading}
          theme={resolvedTheme}
        />
      </section>

    </div>
  );
}

function ModelHealthCard({
  models,
  loading,
  theme,
}: {
  models: ModelCatalogEntry[];
  loading?: boolean;
  theme: "light" | "dark";
}) {
  const providerStats = useMemo(() => {
    const stats = new Map<string, { total: number; enabled: number }>();
    models.forEach((model) => {
      const entry = stats.get(model.provider) ?? { total: 0, enabled: 0 };
      entry.total += 1;
      if (model.enabled) {
        entry.enabled += 1;
      }
      stats.set(model.provider, entry);
    });
    return Array.from(stats.entries()).map(([provider, values]) => ({
      provider,
      label: formatProviderLabel(provider),
      ...values,
    }));
  }, [models]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Model health</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {loading ? (
          <Skeleton className="h-32 w-full" />
        ) : providerStats.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No models configured yet. Add a provider on the Models page.
          </p>
        ) : (
          <div className="space-y-3">
            {providerStats.map((group) => {
              const variant =
                group.enabled === group.total
                  ? "secondary"
                  : group.enabled === 0
                    ? "destructive"
                    : "outline";
              const iconSet = providerIcons[group.provider];
              const icon = iconSet ? iconSet[theme] : null;
              return (
                <div key={group.provider} className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    {icon ? (
                      <img src={icon} alt="" className="h-8 w-8 shrink-0" />
                    ) : null}
                    <div>
                      <p className="font-medium">{group.label}</p>
                    <p className="text-xs text-muted-foreground">
                      {group.enabled}/{group.total} enabled
                    </p>
                    </div>
                  </div>
                  <Badge variant={variant}>
                    {group.enabled === group.total
                      ? "Healthy"
                      : group.enabled === 0
                        ? "Offline"
                        : "Partial"}
                  </Badge>
                </div>
              );
            })}
          </div>
        )}
        <p className="text-xs text-muted-foreground">
          Dig into detailed comparisons on the {" "}
          <Link to="/usage" className="text-primary underline">
            Usage page
          </Link>
          .
        </p>
      </CardContent>
    </Card>
  );
}

function formatProviderLabel(provider: string) {
  switch (provider) {
    case "azure":
      return "Azure OpenAI";
    case "openai":
      return "OpenAI";
    case "openai_compatible":
    case "openai-compatible":
      return "OpenAI-compatible";
    case "bedrock":
      return "Amazon Bedrock";
    case "vertex":
      return "Google Vertex";
    case "anthropic":
      return "Anthropic";
    default:
      return provider.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
  }
}

// Metric cards now live in @/ui/kit/Cards.tsx

function HealthStatusRow({
  label,
  check,
  icon,
}: {
  label: string;
  check?: { status?: string; latency_ms?: number; error?: string };
  icon?: React.ReactNode;
}) {
  const status = check?.status ?? "unknown";
  const variant =
    status === "ok" ? "secondary" : status === "error" ? "destructive" : "outline";
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-2">
        {icon ? <span className="text-muted-foreground">{icon}</span> : null}
        <div>
          <p className="font-medium">{label}</p>
        <p className="text-xs text-muted-foreground">
          {check?.error
            ? check.error
            : check?.latency_ms != null
              ? `${check.latency_ms} ms`
              : "No samples"}
        </p>
        </div>
      </div>
      <Badge variant={variant}>{status}</Badge>
    </div>
  );
}
