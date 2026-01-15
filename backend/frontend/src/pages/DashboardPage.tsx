import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Building2,
  Boxes,
  Activity,
  DollarSign,
  TrendingUp,
  TrendingDown,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { PeriodSelector, usePeriodSelector } from "@/components/ui/period-selector";
import { PageHeader } from "@/components/layouts";
import { listTenants } from "@/api/tenants";
import { listModelCatalog } from "@/api/model-catalog";
import { useUsageOverview } from "@/api/hooks/useUsage";
import { MetricCard } from "@/ui/kit/Cards";
import { RequestsAreaChart } from "@/components/charts/RequestsAreaChart";
import { SpendAreaChart } from "@/components/charts/SpendAreaChart";
import { Sparkline, extractSparklineData } from "@/components/charts/Sparkline";
import { BreakdownTabs } from "@/features/dashboard";
import { cn } from "@/lib/utils";

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
  const periodSelector = usePeriodSelector("7d");
  const [selectedTenantId, setSelectedTenantId] = useState<string>("all");
  const queryParams = periodSelector.getQueryParams();

  const tenantsQuery = useQuery({
    queryKey: ["tenants", "dashboard"],
    queryFn: () => listTenants({ limit: 50 }),
  });

  const modelsQuery = useQuery({
    queryKey: ["model-catalog"],
    queryFn: listModelCatalog,
  });

  const usageQueryParams = useMemo(() => {
    const params = { ...queryParams };
    if (selectedTenantId !== "all") {
      return { ...params, tenantId: selectedTenantId };
    }
    return params;
  }, [queryParams, selectedTenantId]);

  const usageQuery = useUsageOverview(usageQueryParams);

  const selectedTenantName = useMemo(() => {
    if (selectedTenantId === "all") return null;
    return tenantsQuery.data?.tenants.find((t) => t.id === selectedTenantId)?.name ?? null;
  }, [selectedTenantId, tenantsQuery.data?.tenants]);

  const activeTenants =
    tenantsQuery.data?.tenants.filter((t) => t.status === "active").length ?? 0;
  const totalTenants = tenantsQuery.data?.tenants.length ?? 0;
  const enabledModels =
    modelsQuery.data?.filter((model) => model.enabled).length ?? 0;
  const totalModels = modelsQuery.data?.length ?? 0;
  const usageSummary = usageQuery.data;

  // Calculate trends from points data
  const { requestsTrend, spendTrend } = useMemo(() => {
    const points = usageSummary?.points ?? [];
    if (points.length < 2) {
      return { requestsTrend: null, spendTrend: null };
    }
    const midpoint = Math.floor(points.length / 2);
    const firstHalf = points.slice(0, midpoint);
    const secondHalf = points.slice(midpoint);

    const firstRequests = firstHalf.reduce((sum, p) => sum + p.requests, 0);
    const secondRequests = secondHalf.reduce((sum, p) => sum + p.requests, 0);
    const requestsTrend = firstRequests > 0
      ? ((secondRequests - firstRequests) / firstRequests) * 100
      : null;

    const getSpend = (p: typeof points[0]) => p.cost_usd ?? (p.cost_cents / 100);
    const firstSpend = firstHalf.reduce((sum, p) => sum + getSpend(p), 0);
    const secondSpend = secondHalf.reduce((sum, p) => sum + getSpend(p), 0);
    const spendTrend = firstSpend > 0
      ? ((secondSpend - firstSpend) / firstSpend) * 100
      : null;

    return { requestsTrend, spendTrend };
  }, [usageSummary?.points]);

  const periodLabel = periodSelector.getLabel();

  return (
    <div className="space-y-6">
      <PageHeader
        title="Dashboard"
        description={
          selectedTenantName
            ? `Viewing usage for ${selectedTenantName}`
            : "Overview of your gateway infrastructure and usage"
        }
        actions={
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <Select
              value={selectedTenantId}
              onValueChange={setSelectedTenantId}
            >
              <SelectTrigger className="w-full sm:w-48">
                <SelectValue placeholder="Filter by tenant" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All tenants</SelectItem>
                {tenantsQuery.data?.tenants.map((tenant) => (
                  <SelectItem key={tenant.id} value={tenant.id}>
                    {tenant.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <PeriodSelector
              value={periodSelector.value}
              onChange={periodSelector.onChange}
            />
          </div>
        }
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
          title="Requests"
          value={(usageSummary?.total_requests ?? 0).toLocaleString()}
          secondary={
            <div className="flex items-center gap-2">
              <span>{periodLabel}</span>
              {requestsTrend !== null && (
                <TrendBadge value={requestsTrend} />
              )}
            </div>
          }
          loading={usageQuery.isLoading}
          icon={Activity}
          className="stagger-3"
        >
          {usageSummary?.points && usageSummary.points.length > 1 && (
            <Sparkline
              data={extractSparklineData(usageSummary.points, "requests")}
              color="hsl(var(--chart-1))"
              height={40}
              className="mt-2"
            />
          )}
        </MetricCard>
        <MetricCard
          title="Spend"
          value={
            usageSummary
              ? formatSpendAmount(
                  usageSummary.total_cost_usd,
                  usageSummary.total_cost_cents
                )
              : "$0.00"
          }
          secondary={
            <div className="flex items-center gap-2">
              <span>{periodLabel}</span>
              {spendTrend !== null && (
                <TrendBadge value={spendTrend} />
              )}
            </div>
          }
          loading={usageQuery.isLoading}
          icon={DollarSign}
          className="stagger-4"
        >
          {usageSummary?.points && usageSummary.points.length > 1 && (
            <Sparkline
              data={extractSparklineData(usageSummary.points, "spend")}
              color="hsl(var(--chart-3))"
              height={40}
              className="mt-2"
            />
          )}
        </MetricCard>
      </section>

      {/* Charts Section */}
      <section className="grid gap-4 lg:grid-cols-2">
        <Card className="animate-fade-in-up stagger-5">
          <CardHeader>
            <CardTitle className="text-base font-semibold">Request Volume</CardTitle>
            <p className="text-sm text-muted-foreground">
              Daily breakdown for {periodLabel.toLowerCase()}
            </p>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue="requests" className="space-y-4">
              <TabsList>
                <TabsTrigger value="requests">Requests</TabsTrigger>
                <TabsTrigger value="spend">Spend</TabsTrigger>
              </TabsList>
              <TabsContent value="requests">
                <RequestsAreaChart
                  data={usageSummary?.points ?? []}
                  loading={usageQuery.isLoading}
                  height={240}
                />
              </TabsContent>
              <TabsContent value="spend">
                <SpendAreaChart
                  data={usageSummary?.points ?? []}
                  loading={usageQuery.isLoading}
                  height={240}
                />
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>

        <BreakdownTabs queryParams={usageQueryParams} periodLabel={periodLabel} />
      </section>
    </div>
  );
}

function TrendBadge({ value }: { value: number }) {
  const isPositive = value > 0;
  const isNeutral = Math.abs(value) < 1;

  if (isNeutral) {
    return null;
  }

  return (
    <span
      className={cn(
        "inline-flex items-center gap-0.5 text-xs font-medium",
        isPositive ? "text-success" : "text-destructive"
      )}
    >
      {isPositive ? (
        <TrendingUp className="h-3 w-3" />
      ) : (
        <TrendingDown className="h-3 w-3" />
      )}
      {Math.abs(value).toFixed(0)}%
    </span>
  );
}
