import { useMemo, useState } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { isAxiosError } from "axios";
import { Download, RefreshCcw } from "lucide-react";
import { PageHeader } from "@/components/layouts";

import { api } from "@/api/client";
import {
  useModelDailyUsage,
  useTenantDailyUsage,
  useUsageBreakdown,
  useUsageComparison,
  useUsageOverview,
  useUserDailyUsage,
} from "@/api/hooks/useUsage";
import type {
  ModelDailyTenantUsage,
  ModelDailyUsageDay,
  TenantDailyUsageDay,
  TenantDailyUsageKey,
  UsageBreakdownItem,
  UsageBreakdownResponse,
  UsageBreakdownSeriesPoint,
  UsageComparisonSeries,
  UserDailyTenantUsage,
  UserDailyUsageDay,
} from "@/api/usage";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { UsageBreakdownChart } from "@/components/charts/UsageBreakdownChart";
import type { UsageBreakdownDatum } from "@/components/charts/UsageBreakdownChart";
import {
  UsageComparisonChart,
  type UsageComparisonMetric,
} from "@/components/charts/UsageComparisonChart";
import { SummaryCard } from "@/ui/kit/Cards";
import { ChartCard } from "@/ui/kit/ChartCard";
import {
  QueryAlert,
  UsageDailySkeleton,
  UsageDailyTable,
  UsageFilters,
  formatTokensShort,
  formatUsageUSD,
  parseSpendValue,
  useUsageRange,
} from "@/features/usage";
import { formatUsageDate } from "@/lib/dates";
import { getBrowserTimezone } from "@/lib/timezone";
import { cn } from "@/lib/utils";
import { useDirectoryData } from "@/providers/DirectoryProvider";
import { useToast } from "@/hooks/use-toast";
import { useDefaultSelection } from "@/hooks/useDefaultSelection";

const EXPORT_POLL_INTERVAL_MS = 2000;
const EXPORT_POLL_ATTEMPTS = 30;

type UsageExportResponse = {
  id: string;
  status: string;
  download_url?: string;
  error?: string;
};

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

const METRIC_OPTIONS: { value: UsageComparisonMetric; label: string }[] = [
  { value: "spend", label: "Spend" },
  { value: "tokens", label: "Tokens" },
  { value: "requests", label: "Requests" },
];

export function UsagePage() {
  const {
    tenants,
    users,
    models,
    tenantsLoading,
  } = useDirectoryData();
  const { toast } = useToast();

  const [selectedTenantId, setSelectedTenantId] = useState<string | undefined>();
  const [selectedUserId, setSelectedUserId] = useState<string | undefined>();
  const [selectedModelAlias, setSelectedModelAlias] = useState<string | undefined>();
  const [selectedTopTenant, setSelectedTopTenant] = useState<string | undefined>();
  const [selectedTopModel, setSelectedTopModel] = useState<string | undefined>();
  const [selectedTopUser, setSelectedTopUser] = useState<string | undefined>();
  const [tenantMetric, setTenantMetric] = useState<UsageComparisonMetric>("spend");
  const [modelMetric, setModelMetric] = useState<UsageComparisonMetric>("spend");
  const [userMetric, setUserMetric] = useState<UsageComparisonMetric>("spend");
  const [exporting, setExporting] = useState(false);

  useDefaultSelection({
    items: tenants,
    selected: selectedTenantId,
    onChange: setSelectedTenantId,
    getValue: (tenant) => tenant.id,
  });
  useDefaultSelection({
    items: users,
    selected: selectedUserId,
    onChange: setSelectedUserId,
    getValue: (user) => user.id,
  });
  useDefaultSelection({
    items: models,
    selected: selectedModelAlias,
    onChange: setSelectedModelAlias,
    getValue: (model) => model.alias,
  });

  const {
    startInput,
    endInput,
    setStartInput,
    setEndInput,
    range: activeRange,
    rangeError,
    rangeDisplay,
  } = useUsageRange({ defaultDays: 6 });

  const overviewQueryEnabled = Boolean(activeRange && !rangeError);
  const usageQuery = useUsageOverview(
    overviewQueryEnabled
      ? { start: activeRange!.start, end: activeRange!.end }
      : undefined,
    { enabled: overviewQueryEnabled },
  );

  const tenantDailyQuery = useTenantDailyUsage(
    selectedTenantId && activeRange && !rangeError
      ? {
          tenantId: selectedTenantId,
          start: activeRange.start,
          end: activeRange.end,
        }
      : undefined,
  );

  const userDailyQuery = useUserDailyUsage(
    selectedUserId && activeRange && !rangeError
      ? {
          userId: selectedUserId,
          start: activeRange.start,
          end: activeRange.end,
        }
      : undefined,
  );

  const modelDailyQuery = useModelDailyUsage(
    selectedModelAlias && activeRange && !rangeError
      ? {
          modelAlias: selectedModelAlias,
          start: activeRange.start,
          end: activeRange.end,
        }
      : undefined,
  );
  const breakdownEnabled = overviewQueryEnabled;
  const tenantBreakdown = useUsageBreakdown(
    {
      group: "tenant",
      limit: 5,
      entityId: selectedTopTenant,
      ...(activeRange && !rangeError ? { start: activeRange.start, end: activeRange.end } : {}),
    },
    { enabled: breakdownEnabled },
  );
  const modelBreakdown = useUsageBreakdown(
    {
      group: "model",
      limit: 5,
      entityId: selectedTopModel,
      ...(activeRange && !rangeError ? { start: activeRange.start, end: activeRange.end } : {}),
    },
    { enabled: breakdownEnabled },
  );
  const userBreakdown = useUsageBreakdown(
    {
      group: "user",
      limit: 5,
      entityId: selectedTopUser,
      ...(activeRange && !rangeError ? { start: activeRange.start, end: activeRange.end } : {}),
    },
    { enabled: breakdownEnabled },
  );

  const topTenantIds = useMemo(
    () => (tenantBreakdown.data?.items ?? []).slice(0, 5).map((item) => item.id),
    [tenantBreakdown.data?.items],
  );
  const topModelIds = useMemo(
    () => (modelBreakdown.data?.items ?? []).slice(0, 5).map((item) => item.id),
    [modelBreakdown.data?.items],
  );
  const topUserIds = useMemo(
    () => (userBreakdown.data?.items ?? []).slice(0, 5).map((item) => item.id),
    [userBreakdown.data?.items],
  );

  const tenantComparison = useUsageComparison(
    activeRange && !rangeError
      ? { tenantIds: topTenantIds, start: activeRange.start, end: activeRange.end }
      : { tenantIds: topTenantIds, period: "7d" },
    { enabled: breakdownEnabled && Boolean(activeRange && !rangeError && topTenantIds.length) },
  );
  const modelComparison = useUsageComparison(
    activeRange && !rangeError
      ? { modelAliases: topModelIds, start: activeRange.start, end: activeRange.end }
      : { modelAliases: topModelIds, period: "7d" },
    { enabled: breakdownEnabled && Boolean(activeRange && !rangeError && topModelIds.length) },
  );
  const userComparison = useUsageComparison(
    activeRange && !rangeError
      ? { userIds: topUserIds, start: activeRange.start, end: activeRange.end }
      : { userIds: topUserIds, period: "7d" },
    { enabled: breakdownEnabled && Boolean(activeRange && !rangeError && topUserIds.length) },
  );

  const tenantComparisonSeries = useMemo(() => {
    const series = tenantComparison.data?.series ?? [];
    if (!series.length) {
      return [];
    }
    const map = new Map(series.map((entry) => [entry.id, entry]));
    return topTenantIds.map((id) => map.get(id)).filter(Boolean) as UsageComparisonSeries[];
  }, [tenantComparison.data?.series, topTenantIds]);
  const modelComparisonSeries = useMemo(() => {
    const series = modelComparison.data?.series ?? [];
    if (!series.length) {
      return [];
    }
    const map = new Map(series.map((entry) => [entry.id, entry]));
    return topModelIds.map((id) => map.get(id)).filter(Boolean) as UsageComparisonSeries[];
  }, [modelComparison.data?.series, topModelIds]);
  const userComparisonSeries = useMemo(() => {
    const series = userComparison.data?.series ?? [];
    if (!series.length) {
      return [];
    }
    const map = new Map(series.map((entry) => [entry.id, entry]));
    return topUserIds.map((id) => map.get(id)).filter(Boolean) as UsageComparisonSeries[];
  }, [userComparison.data?.series, topUserIds]);

  useDefaultSelection({
    items: tenantBreakdown.data?.items ?? [],
    selected: selectedTopTenant,
    onChange: setSelectedTopTenant,
    getValue: (item) => item.id,
    selectFirst: false,
  });
  useDefaultSelection({
    items: modelBreakdown.data?.items ?? [],
    selected: selectedTopModel,
    onChange: setSelectedTopModel,
    getValue: (item) => item.id,
    selectFirst: false,
  });
  useDefaultSelection({
    items: userBreakdown.data?.items ?? [],
    selected: selectedTopUser,
    onChange: setSelectedTopUser,
    getValue: (item) => item.id,
    selectFirst: false,
  });

  const selectedTenant = tenants.find((tenant) => tenant.id === selectedTenantId);
  const selectedUser = users.find((user) => user.id === selectedUserId);
  const selectedModel = models.find((model) => model.alias === selectedModelAlias);

  const handleExport = async () => {
    if (exporting) {
      return;
    }
    setExporting(true);

    const basePayload: Record<string, unknown> = {
      timezone: getBrowserTimezone(),
      format: "csv",
      granularity: "daily",
    };
    if (activeRange && !rangeError) {
      basePayload.start = activeRange.start;
      basePayload.end = activeRange.end;
    } else {
      basePayload.period = "7d";
    }

    const createExport = async (tenantIds?: string[]) => {
      const payload = { ...basePayload } as Record<string, unknown>;
      if (tenantIds?.length) {
        payload.tenant_ids = tenantIds;
      }
      const { data } = await api.post<UsageExportResponse>("/usage-exports", payload);
      return data;
    };

    try {
      let exportResp: UsageExportResponse | undefined;
      try {
        exportResp = await createExport();
      } catch (err) {
        if (
          isAxiosError(err) &&
          err.response?.status === 400 &&
          selectedTenantId
        ) {
          const message =
            err.response?.data?.error ||
            err.response?.data?.message ||
            err.message;
          if (typeof message === "string" && message.includes("tenant_ids required")) {
            exportResp = await createExport([selectedTenantId]);
          } else {
            throw err;
          }
        } else {
          throw err;
        }
      }

      if (!exportResp) {
        return;
      }

      toast({
        title: "Export queued",
        description: "Preparing a CSV export. We'll open it once ready.",
      });

      let latest: UsageExportResponse | null = null;
      for (let attempt = 0; attempt < EXPORT_POLL_ATTEMPTS; attempt += 1) {
        await sleep(EXPORT_POLL_INTERVAL_MS);
        const { data } = await api.get<UsageExportResponse>(
          `/usage-exports/${exportResp.id}`,
        );
        latest = data;
        if (data.status === "ready") {
          break;
        }
        if (data.status === "failed") {
          throw new Error(data.error || "Export failed");
        }
      }

      if (latest?.status === "ready") {
        const rawUrl = latest.download_url || `/admin/usage-exports/${exportResp.id}/content`;
        const downloadUrl = rawUrl.startsWith("http")
          ? rawUrl
          : new URL(rawUrl, window.location.origin).toString();
        window.open(downloadUrl, "_blank");
        toast({
          title: "Export ready",
          description: "Your CSV export is downloading.",
        });
      } else {
        toast({
          title: "Export queued",
          description: "The export is still processing. Check back in a moment.",
        });
      }
    } catch (err) {
      if (!isAxiosError(err)) {
        toast({
          variant: "destructive",
          title: "Export failed",
          description:
            err instanceof Error ? err.message : "Unable to prepare export.",
        });
      }
    } finally {
      setExporting(false);
    }
  };

  const timezone = usageQuery.data?.timezone ?? "UTC";
  const totalRequests = usageQuery.data?.total_requests ?? 0;
  const totalTokens = usageQuery.data?.total_tokens ?? 0;
  const totalCostUsd = usageQuery.data?.total_cost_usd;
  const totalCostCents = usageQuery.data?.total_cost_cents;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Usage"
        description="Track platform-wide consumption and drill into tenants, users, or model trends."
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={() => usageQuery.refetch()}
              loading={usageQuery.isFetching}
            >
              <RefreshCcw className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              onClick={handleExport}
              disabled={Boolean(rangeError) || exporting}
            >
              <Download className="mr-2 h-4 w-4" /> Export CSV
            </Button>
          </div>
        }
      />

      <div className="flex flex-col gap-2 md:flex-row md:items-center">
        <UsageFilters
          startInput={startInput}
          endInput={endInput}
          onStartChange={setStartInput}
          onEndChange={setEndInput}
          rangeError={rangeError}
          rangeDisplay={rangeDisplay}
          timezone={timezone}
          idPrefix="usage"
        />
      </div>

      {rangeError ? (
        <p className="text-sm text-destructive">Adjust the date range above to continue.</p>
      ) : null}

      <Tabs defaultValue="overview" className="space-y-6">
        <TabsList className="w-fit">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="tenants">Tenants</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
          <TabsTrigger value="models">Models</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <SummaryCard
              title="Total requests"
              value={usageQuery.data ? totalRequests.toLocaleString() : "—"}
              description={rangeDisplay ?? "Across the selected window."}
              loading={usageQuery.isLoading}
            />
            <SummaryCard
              title="Total spend"
              value={
                usageQuery.data
                  ? formatUsageUSD(totalCostUsd, totalCostCents)
                  : "—"
              }
              description="Usage-based fees"
              loading={usageQuery.isLoading}
            />
            <SummaryCard
              title="Tokens processed"
              value={usageQuery.data ? formatTokensShort(totalTokens) : "—"}
              description="Prompt + completion"
              loading={usageQuery.isLoading}
            />
            <SummaryCard
              title="Tenants covered"
              value={tenants.length}
              description="Across the selected window"
              loading={tenantsLoading}
            />
          </section>

          {breakdownEnabled ? (
            <div className="flex flex-col gap-4">
              <TopUsageCard
                title="Top tenants"
                description="Daily usage for the selected tenant."
                breakdown={tenantBreakdown}
                comparisonSeries={tenantComparisonSeries}
                comparisonIsLoading={tenantComparison.isFetching && !tenantComparison.data}
                metric={tenantMetric}
                onMetricChange={setTenantMetric}
                selectedId={selectedTopTenant}
                onSelectId={setSelectedTopTenant}
                timezone={timezone}
              />
              <TopUsageCard
                title="Top users"
                description="Track which user accounts consume the most budget."
                breakdown={userBreakdown}
                comparisonSeries={userComparisonSeries}
                comparisonIsLoading={userComparison.isFetching && !userComparison.data}
                metric={userMetric}
                onMetricChange={setUserMetric}
                selectedId={selectedTopUser}
                onSelectId={setSelectedTopUser}
                timezone={timezone}
              />
              <TopUsageCard
                title="Top models"
                description="Daily demand for your routed model aliases."
                breakdown={modelBreakdown}
                comparisonSeries={modelComparisonSeries}
                comparisonIsLoading={modelComparison.isFetching && !modelComparison.data}
                metric={modelMetric}
                onMetricChange={setModelMetric}
                selectedId={selectedTopModel}
                onSelectId={setSelectedTopModel}
                timezone={timezone}
              />
            </div>
          ) : null}

          <Card>
            <CardHeader>
              <CardTitle>Daily breakdown</CardTitle>
              <p className="text-sm text-muted-foreground">
                Aggregated usage for the selected date range. Times shown in {timezone}.
              </p>
            </CardHeader>
            <CardContent>
              <QueryAlert
                error={usageQuery.isError ? (usageQuery.error as Error) : null}
                onRetry={usageQuery.refetch}
              />
              {usageQuery.isLoading ? (
                <UsageDailySkeleton />
              ) : (usageQuery.data?.points ?? []).length === 0 ? (
                <p className="text-sm text-muted-foreground">No usage recorded.</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Date</TableHead>
                      <TableHead className="text-right">Requests</TableHead>
                      <TableHead className="text-right">Tokens</TableHead>
                      <TableHead className="text-right">Cost</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {usageQuery.data?.points.map((point) => (
                      <TableRow key={point.date}>
                        <TableCell>{formatUsageDate(point.date, timezone)}</TableCell>
                        <TableCell className="text-right">{point.requests.toLocaleString()}</TableCell>
                        <TableCell className="text-right">{point.tokens.toLocaleString()}</TableCell>
                        <TableCell className="text-right">
                          {formatUsageUSD(point.cost_usd, point.cost_cents)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="tenants" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Tenant daily usage</CardTitle>
              <p className="text-sm text-muted-foreground">
                Select a tenant to review daily totals with per-key breakdowns.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Tenant</Label>
                <Select value={selectedTenantId} onValueChange={setSelectedTenantId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select tenant" />
                  </SelectTrigger>
                  <SelectContent>
                    {tenants.map((tenant) => (
                      <SelectItem key={tenant.id} value={tenant.id}>
                        {tenant.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {selectedTenant ? (
                  <p className="text-xs text-muted-foreground">
                    Status: {selectedTenant.status} · Budget limit{" "}
                    {formatUsageUSD(selectedTenant.budget_limit_usd ?? 0, undefined)} · Used{" "}
                    {formatUsageUSD(selectedTenant.budget_used_usd ?? 0, undefined)}
                  </p>
                ) : null}
              </div>
              <QueryAlert
                error={tenantDailyQuery.isError ? (tenantDailyQuery.error as Error) : null}
                onRetry={tenantDailyQuery.refetch}
              />
              {!selectedTenantId ? (
                <p className="text-sm text-muted-foreground">Select a tenant to begin.</p>
              ) : tenantDailyQuery.isLoading ? (
                <UsageDailySkeleton />
              ) : tenantDailyQuery.data && tenantDailyQuery.data.days.length ? (
                <TenantDailyList
                  days={tenantDailyQuery.data.days}
                  timezone={tenantDailyQuery.data.timezone}
                />
              ) : (
                <p className="text-sm text-muted-foreground">No activity for this range.</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="users" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>User daily usage</CardTitle>
              <p className="text-sm text-muted-foreground">
                Compare per-user activity across tenants for the selected range.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>User</Label>
                <Select value={selectedUserId} onValueChange={setSelectedUserId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select user" />
                  </SelectTrigger>
                  <SelectContent>
                    {users.map((user) => (
                      <SelectItem key={user.id} value={user.id}>
                        {user.name?.trim() || user.email}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {selectedUser ? (
                  <p className="text-xs text-muted-foreground">{selectedUser.email}</p>
                ) : null}
              </div>
              <QueryAlert
                error={userDailyQuery.isError ? (userDailyQuery.error as Error) : null}
                onRetry={userDailyQuery.refetch}
              />
              {!selectedUserId ? (
                <p className="text-sm text-muted-foreground">Select a user to begin.</p>
              ) : userDailyQuery.isLoading ? (
                <UsageDailySkeleton />
              ) : userDailyQuery.data && userDailyQuery.data.days.length ? (
                <UserDailyList
                  days={userDailyQuery.data.days}
                  timezone={userDailyQuery.data.timezone}
                />
              ) : (
                <p className="text-sm text-muted-foreground">No activity for this range.</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="models" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Model daily usage</CardTitle>
              <p className="text-sm text-muted-foreground">
                Review per-tenant usage for a specific model alias.
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Model</Label>
                <Select value={selectedModelAlias} onValueChange={setSelectedModelAlias}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select model" />
                  </SelectTrigger>
                  <SelectContent>
                    {models.map((model) => (
                      <SelectItem key={model.alias} value={model.alias}>
                        {model.alias}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {selectedModel ? (
                  <p className="text-xs text-muted-foreground">
                    Provider: {selectedModel.provider}
                  </p>
                ) : null}
              </div>
              <QueryAlert
                error={modelDailyQuery.isError ? (modelDailyQuery.error as Error) : null}
                onRetry={modelDailyQuery.refetch}
              />
              {!selectedModelAlias ? (
                <p className="text-sm text-muted-foreground">Select a model to begin.</p>
              ) : modelDailyQuery.isLoading ? (
                <UsageDailySkeleton />
              ) : modelDailyQuery.data && modelDailyQuery.data.days.length ? (
                <ModelDailyList
                  days={modelDailyQuery.data.days}
                  timezone={modelDailyQuery.data.timezone}
                />
              ) : (
                <p className="text-sm text-muted-foreground">No activity for this range.</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function TenantDailyList({ days, timezone }: { days: TenantDailyUsageDay[]; timezone: string }) {
  return (
    <UsageDailyTable<TenantDailyUsageKey, TenantDailyUsageDay>
      days={days}
      timezone={timezone}
      emptyMessage="No key activity for this day."
      renderBreakdown={(key) => (
        <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 rounded border bg-background/80 px-3 py-2">
          <div>
            <p className="font-medium">{key.api_key_name || key.api_key_prefix || "Unnamed key"}</p>
            <p className="text-xs text-muted-foreground">{key.api_key_prefix || key.api_key_id}</p>
          </div>
          <span className="text-right">{key.requests.toLocaleString()}</span>
          <span className="text-right">{key.tokens.toLocaleString()}</span>
          <span className="text-right">{formatUsageUSD(key.cost_usd, key.cost_cents)}</span>
        </div>
      )}
      breakdownHeaders={["API key", "Requests", "Tokens", "Spend"]}
      getBreakdown={(day) => day.keys}
    />
  );
}

function UserDailyList({ days, timezone }: { days: UserDailyUsageDay[]; timezone: string }) {
  return (
    <UsageDailyTable<UserDailyTenantUsage, UserDailyUsageDay>
      days={days}
      timezone={timezone}
      emptyMessage="No tenant activity for this day."
      renderBreakdown={(tenant) => (
        <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 rounded border bg-background/80 px-3 py-2">
          <div>
            <p className="font-medium">{tenant.tenant_name || tenant.tenant_id || "Tenant"}</p>
            <p className="text-xs text-muted-foreground">{tenant.tenant_id}</p>
          </div>
          <span className="text-right">{tenant.requests.toLocaleString()}</span>
          <span className="text-right">{tenant.tokens.toLocaleString()}</span>
          <span className="text-right">{formatUsageUSD(tenant.cost_usd, tenant.cost_cents)}</span>
        </div>
      )}
      breakdownHeaders={["Tenant", "Requests", "Tokens", "Spend"]}
      getBreakdown={(day) => day.tenants}
    />
  );
}

function ModelDailyList({ days, timezone }: { days: ModelDailyUsageDay[]; timezone: string }) {
  return (
    <UsageDailyTable<ModelDailyTenantUsage, ModelDailyUsageDay>
      days={days}
      timezone={timezone}
      emptyMessage="No tenant activity for this model on this day."
      renderBreakdown={(tenant) => (
        <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 rounded border bg-background/80 px-3 py-2">
          <div>
            <p className="font-medium">{tenant.tenant_name || tenant.tenant_id || "Tenant"}</p>
            <p className="text-xs text-muted-foreground">{tenant.tenant_id}</p>
          </div>
          <span className="text-right">{tenant.requests.toLocaleString()}</span>
          <span className="text-right">{tenant.tokens.toLocaleString()}</span>
          <span className="text-right">{formatUsageUSD(tenant.cost_usd, tenant.cost_cents)}</span>
        </div>
      )}
      breakdownHeaders={["Tenant", "Requests", "Tokens", "Spend"]}
      getBreakdown={(day) => day.tenants}
    />
  );
}

interface TopUsageCardProps {
  title: string;
  description: string;
  breakdown: UseQueryResult<UsageBreakdownResponse>;
  comparisonSeries?: UsageComparisonSeries[];
  comparisonIsLoading?: boolean;
  metric: UsageComparisonMetric;
  onMetricChange: (metric: UsageComparisonMetric) => void;
  selectedId?: string;
  onSelectId: (id: string) => void;
  timezone: string;
}

function TopUsageCard({
  title,
  description,
  breakdown,
  comparisonSeries,
  comparisonIsLoading,
  metric,
  onMetricChange,
  selectedId,
  onSelectId,
  timezone,
}: TopUsageCardProps) {
  const items = breakdown.data?.items ?? [];
  const series = breakdown.data?.series ?? null;
  const chartData = buildChartData(series?.points);
  const resolvedId = selectedId ?? series?.id ?? items[0]?.id ?? "";
  const hasComparison = (comparisonSeries?.length ?? 0) > 0;

  const toolbar = (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <Select value={metric} onValueChange={(value) => onMetricChange(value as UsageComparisonMetric)}>
        <SelectTrigger>
          <SelectValue placeholder="Metric" />
        </SelectTrigger>
        <SelectContent>
          {METRIC_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );

  return (
    <ChartCard
      title={title}
      description={description}
      toolbar={toolbar}
      isLoading={breakdown.isLoading}
    >
      <QueryAlert
        error={breakdown.isError ? (breakdown.error as Error) : null}
        onRetry={breakdown.refetch}
      />
      {!breakdown.isLoading && (
        <>
          {comparisonIsLoading ? (
            <Skeleton className="h-64 w-full" />
          ) : hasComparison ? (
            <UsageComparisonChart
              series={comparisonSeries!}
              metric={metric}
              timezone={timezone}
              activeSeriesId={resolvedId}
            />
          ) : (
            <UsageBreakdownChart data={chartData} metric={metric} timezone={timezone} />
          )}
          <TopUsageList items={items} metric={metric} resolvedId={resolvedId} onSelectId={onSelectId} />
        </>
      )}
    </ChartCard>
  );
}

interface TopUsageListProps {
  items: UsageBreakdownItem[];
  metric: UsageComparisonMetric;
  resolvedId?: string;
  onSelectId: (id: string) => void;
}

function TopUsageList({ items, metric, resolvedId, onSelectId }: TopUsageListProps) {
  if (!items.length) {
    return <p className="text-sm text-muted-foreground">No usage recorded for this range.</p>;
  }
  const values = items.map((item) => metricValue(metric, item));
  const maxValue = Math.max(...values, 0);

  return (
    <div className="space-y-2">
      {items.map((item, index) => {
        const value = values[index];
        const percent = maxValue > 0 ? (value / maxValue) * 100 : 0;
        return (
          <button
            key={item.id}
            type="button"
            onClick={() => onSelectId(item.id)}
            className={cn(
              "w-full rounded-md border p-3 text-left transition",
              resolvedId === item.id ? "border-primary bg-primary/5" : "border-border hover:bg-muted/50",
            )}
          >
            <div className="flex items-center justify-between gap-4">
              <span className="font-medium truncate">{item.label || "Unnamed"}</span>
              <span className="text-sm font-semibold">{formatMetricDisplay(metric, value)}</span>
            </div>
            <div className="mt-2 h-1 w-full rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-primary transition-all"
                style={{ width: `${percent}%` }}
              />
            </div>
          </button>
        );
      })}
    </div>
  );
}

function buildChartData(points?: UsageBreakdownSeriesPoint[]): UsageBreakdownDatum[] {
  if (!points) {
    return [];
  }
  return points.map((point) => ({
    label: point.date,
    requests: point.requests,
    tokens: point.tokens,
    spend: parseSpendValue(point.cost_usd, point.cost_cents),
  }));
}

function metricValue(metric: UsageComparisonMetric, item: UsageBreakdownItem): number {
  switch (metric) {
    case "requests":
      return item.requests;
    case "tokens":
      return item.tokens;
    case "spend":
    default:
      return parseSpendValue(item.cost_usd, item.cost_cents);
  }
}

function formatMetricDisplay(metric: UsageComparisonMetric, value: number) {
  if (metric === "spend") {
    return `$${value.toFixed(2)}`;
  }
  return value.toLocaleString();
}
