import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  Building2,
  Boxes,
  Activity,
  DollarSign,
  Globe,
  ExternalLink,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/layouts";
import { listTenants } from "@/api/tenants";
import { listModelCatalog } from "@/api/model-catalog";
import type { ModelCatalogEntry } from "@/api/model-catalog";
import { fetchHealth } from "@/api/health";
import { useUsageOverview } from "@/api/hooks/useUsage";
import { formatTokensShort } from "@/lib/numbers";
import { MetricCard, StatusDot } from "@/ui/kit/Cards";
import PostgresIcon from "@/assets/system/postgres.svg";
import RedisIcon from "@/assets/system/redis.svg";
import { useTheme } from "@/providers/ThemeProvider";
import { getProviderIcon } from "@/features/models/provider-icons";

const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
});

const formatSpendAmount = (usd?: number, cents?: number) =>
  currencyFormatter.format(
    typeof usd === "number"
      ? usd
      : typeof cents === "number"
        ? cents / 100
        : 0
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
      <PageHeader
        title="Dashboard"
        description="Overview of your gateway infrastructure and usage"
      />

      {/* KPI Cards */}
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          title="Active Tenants"
          value={activeTenants}
          secondary={`${totalTenants} total`}
          loading={tenantsQuery.isLoading}
          icon={Building2}
          className="stagger-1"
        />
        <MetricCard
          title="Models Online"
          value={enabledModels}
          secondary={`${totalModels} configured`}
          loading={modelsQuery.isLoading}
          icon={Boxes}
          status={enabledModels > 0 ? "success" : undefined}
          className="stagger-2"
        />
        <MetricCard
          title="Weekly Requests"
          value={(usageSummary?.total_requests ?? 0).toLocaleString()}
          secondary={`${((usageSummary?.total_requests ?? 0) / 7).toFixed(0)} avg/day`}
          loading={usageQuery.isLoading}
          icon={Activity}
          className="stagger-3"
        />
        <MetricCard
          title="Weekly Spend"
          value={
            usageSummary
              ? formatSpendAmount(
                  usageSummary.total_cost_usd,
                  usageSummary.total_cost_cents
                )
              : "$0.00"
          }
          secondary={`${formatTokensShort(usageSummary?.total_tokens ?? 0)} tokens`}
          loading={usageQuery.isLoading}
          icon={DollarSign}
          className="stagger-4"
        />
      </section>

      {/* Health Cards */}
      <section className="grid gap-4 lg:grid-cols-2">
        <Card className="animate-fade-in-up stagger-5">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
            <CardTitle className="text-base font-semibold">Gateway Health</CardTitle>
            <StatusDot
              status={healthQuery.data?.status === "ok" ? "online" : "warning"}
              pulse
            />
          </CardHeader>
          <CardContent className="space-y-4">
            {healthQuery.isLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : (
              <>
                {/* Global Status */}
                <div className="flex items-center justify-between rounded-lg bg-muted/50 p-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-background">
                      <Globe className="h-5 w-5 text-muted-foreground" />
                    </div>
                    <div>
                      <p className="text-sm font-medium">Global Status</p>
                      <p className="text-xs text-muted-foreground">
                        All systems operational
                      </p>
                    </div>
                  </div>
                  <Badge
                    variant={
                      healthQuery.data?.status === "ok"
                        ? "success-soft"
                        : "warning-soft"
                    }
                  >
                    {formatStatusLabel(healthQuery.data?.status)}
                  </Badge>
                </div>

                {/* Infrastructure Services */}
                <div className="space-y-2">
                  <HealthStatusRow
                    label="Postgres"
                    icon={<img src={PostgresIcon} alt="" className="h-6 w-6" />}
                    check={healthQuery.data?.checks?.postgres}
                  />
                  <HealthStatusRow
                    label="Redis"
                    icon={<img src={RedisIcon} alt="" className="h-6 w-6" />}
                    check={healthQuery.data?.checks?.redis}
                  />
                </div>

                <p className="text-xs text-muted-foreground">
                  Last checked:{" "}
                  {new Date(healthQuery.dataUpdatedAt).toLocaleTimeString()}
                </p>
              </>
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

  const totalEnabled = providerStats.reduce((sum, p) => sum + p.enabled, 0);
  const totalModels = providerStats.reduce((sum, p) => sum + p.total, 0);

  return (
    <Card className="animate-fade-in-up stagger-6">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
        <div>
          <CardTitle className="text-base font-semibold">Provider Health</CardTitle>
          <p className="mt-1 text-sm text-muted-foreground">
            {totalEnabled}/{totalModels} models enabled
          </p>
        </div>
        <Link
          to="/provider-health"
          className="text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ExternalLink className="h-4 w-4" />
        </Link>
      </CardHeader>
      <CardContent className="space-y-3">
        {loading ? (
          <div className="space-y-3">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : providerStats.length === 0 ? (
          <div className="rounded-lg bg-muted/50 p-4 text-center">
            <p className="text-sm text-muted-foreground">
              No models configured yet.
            </p>
            <Link
              to="/models"
              className="mt-2 inline-block text-sm text-primary hover:underline"
            >
              Add your first model
            </Link>
          </div>
        ) : (
          <div className="space-y-2">
            {providerStats.map((group) => {
              const icon = getProviderIcon(group.provider, theme);
              const tone =
                group.enabled === group.total
                  ? "success"
                  : group.enabled === 0
                    ? "danger"
                    : "warning";
              const badgeVariant =
                tone === "success"
                  ? "success-soft"
                  : tone === "warning"
                    ? "warning-soft"
                    : "destructive-soft";

              return (
                <div
                  key={group.provider}
                  className="flex items-center justify-between rounded-lg bg-muted/30 p-3 transition-colors hover:bg-muted/50"
                >
                  <div className="flex items-center gap-3">
                    {icon ? (
                      <div className="flex h-8 w-8 items-center justify-center rounded bg-background">
                        <img src={icon} alt="" className="h-5 w-5" />
                      </div>
                    ) : (
                      <div className="flex h-8 w-8 items-center justify-center rounded bg-muted">
                        <Boxes className="h-4 w-4 text-muted-foreground" />
                      </div>
                    )}
                    <div>
                      <p className="text-sm font-medium">{group.label}</p>
                      <p className="text-xs text-muted-foreground">
                        {group.enabled}/{group.total} enabled
                      </p>
                    </div>
                  </div>
                  <Badge variant={badgeVariant}>
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
    case "openrouter":
      return "OpenRouter";
    case "groq":
      return "Groq";
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
  const badgeVariant =
    status === "ok"
      ? "success-soft"
      : status === "error"
        ? "destructive-soft"
        : "muted";

  return (
    <div className="flex items-center justify-between rounded-lg bg-muted/30 p-3 transition-colors hover:bg-muted/50">
      <div className="flex items-center gap-3">
        {icon && (
          <div className="flex h-8 w-8 items-center justify-center rounded bg-background">
            {icon}
          </div>
        )}
        <div>
          <p className="text-sm font-medium">{label}</p>
          <p className="text-xs text-muted-foreground">
            {check?.error
              ? check.error
              : check?.latency_ms != null
                ? `${check.latency_ms}ms latency`
                : "No samples"}
          </p>
        </div>
      </div>
      <Badge variant={badgeVariant}>{formatStatusLabel(status)}</Badge>
    </div>
  );
}

function formatStatusLabel(status?: string | null) {
  if (!status) return "Unknown";
  if (status === "ok") return "Healthy";
  return status.charAt(0).toUpperCase() + status.slice(1);
}
