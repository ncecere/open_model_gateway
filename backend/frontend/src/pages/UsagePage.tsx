import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { ChevronDown, Download } from "lucide-react";

import {
  useModelDailyUsage,
  useTenantDailyUsage,
  useUsageBreakdown,
  useUsageComparison,
  useUsageOverview,
  useUserDailyUsage,
} from "@/api/hooks/useUsage";
import { listTenants } from "@/api/tenants";
import { listUsers } from "@/api/users";
import { listModelCatalog, type ModelCatalogEntry } from "@/api/model-catalog";
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
import { Input } from "@/components/ui/input";
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
import { Separator } from "@/components/ui/separator";
import { UsageBreakdownChart } from "@/components/charts/UsageBreakdownChart";
import type { UsageBreakdownDatum } from "@/components/charts/UsageBreakdownChart";
import {
  UsageComparisonChart,
  type UsageComparisonMetric,
} from "@/components/charts/UsageComparisonChart";
import { SummaryCard } from "@/ui/kit/Cards";
import { QueryAlert } from "@/features/usage";
import { formatUsageDate } from "@/lib/dates";
import { formatTokensShort } from "@/lib/numbers";
import { cn } from "@/lib/utils";

const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

const formatSpendValue = (usd?: number, cents?: number) =>
  currencyFormatter.format(
    typeof usd === "number"
      ? usd
      : typeof cents === "number"
        ? cents / 100
        : 0,
  );

const METRIC_OPTIONS: { value: UsageComparisonMetric; label: string }[] = [
  { value: "spend", label: "Spend" },
  { value: "tokens", label: "Tokens" },
  { value: "requests", label: "Requests" },
];

export function UsagePage() {
  const tenantsQuery = useQuery({
    queryKey: ["tenants", "usage"],
    queryFn: () => listTenants({ limit: 500 }),
  });
  const tenants = tenantsQuery.data?.tenants ?? [];

  const usersQuery = useQuery({
    queryKey: ["users", "usage"],
    queryFn: () => listUsers({ limit: 500 }),
  });
  const users = usersQuery.data?.users ?? [];

  const modelsQuery = useQuery({
    queryKey: ["model-catalog"],
    queryFn: listModelCatalog,
  });
  const models: ModelCatalogEntry[] = modelsQuery.data ?? [];

  const [selectedTenantId, setSelectedTenantId] = useState<string | undefined>();
  const [selectedUserId, setSelectedUserId] = useState<string | undefined>();
  const [selectedModelAlias, setSelectedModelAlias] = useState<string | undefined>();
  const [selectedTopTenant, setSelectedTopTenant] = useState<string | undefined>();
  const [selectedTopModel, setSelectedTopModel] = useState<string | undefined>();
  const [selectedTopUser, setSelectedTopUser] = useState<string | undefined>();
  const [tenantMetric, setTenantMetric] = useState<UsageComparisonMetric>("spend");
  const [modelMetric, setModelMetric] = useState<UsageComparisonMetric>("spend");
  const [userMetric, setUserMetric] = useState<UsageComparisonMetric>("spend");

  useEffect(() => {
    if (!tenants.length) {
      setSelectedTenantId(undefined);
      return;
    }
    if (!selectedTenantId || !tenants.find((tenant) => tenant.id === selectedTenantId)) {
      setSelectedTenantId(tenants[0].id);
    }
  }, [tenants, selectedTenantId]);

  useEffect(() => {
    if (!users.length) {
      setSelectedUserId(undefined);
      return;
    }
    if (!selectedUserId || !users.find((user) => user.id === selectedUserId)) {
      setSelectedUserId(users[0].id);
    }
  }, [users, selectedUserId]);

  useEffect(() => {
    if (!models.length) {
      setSelectedModelAlias(undefined);
      return;
    }
    if (!selectedModelAlias || !models.find((model) => model.alias === selectedModelAlias)) {
      setSelectedModelAlias(models[0].alias);
    }
  }, [models, selectedModelAlias]);

  const [startInput, setStartInput] = useState(() =>
    formatDateInput(addDays(startOfToday(), -6)),
  );
  const [endInput, setEndInput] = useState(() => formatDateInput(startOfToday()));
  const rangeResult = useMemo(
    () => deriveRangeISO(startInput, endInput),
    [startInput, endInput],
  );
  const activeRange = rangeResult?.range;
  const rangeError = rangeResult?.error;
  const rangeDisplay =
    activeRange && !rangeError
      ? `${formatInputDisplay(startInput)} – ${formatInputDisplay(endInput)}`
      : null;

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

  useEffect(() => {
    const items = tenantBreakdown.data?.items ?? [];
    if (!items.length) {
      if (selectedTopTenant !== undefined) {
        setSelectedTopTenant(undefined);
      }
      return;
    }
    if (selectedTopTenant && !items.some((item) => item.id === selectedTopTenant)) {
      setSelectedTopTenant(undefined);
    }
  }, [tenantBreakdown.data?.items, selectedTopTenant]);

  useEffect(() => {
    const items = modelBreakdown.data?.items ?? [];
    if (!items.length) {
      if (selectedTopModel !== undefined) {
        setSelectedTopModel(undefined);
      }
      return;
    }
    if (selectedTopModel && !items.some((item) => item.id === selectedTopModel)) {
      setSelectedTopModel(undefined);
    }
  }, [modelBreakdown.data?.items, selectedTopModel]);

  useEffect(() => {
    const items = userBreakdown.data?.items ?? [];
    if (!items.length) {
      if (selectedTopUser !== undefined) {
        setSelectedTopUser(undefined);
      }
      return;
    }
    if (selectedTopUser && !items.some((item) => item.id === selectedTopUser)) {
      setSelectedTopUser(undefined);
    }
  }, [userBreakdown.data?.items, selectedTopUser]);

  const selectedTenant = tenants.find((tenant) => tenant.id === selectedTenantId);
  const selectedUser = users.find((user) => user.id === selectedUserId);
  const selectedModel = models.find((model) => model.alias === selectedModelAlias);

  const handleExport = () => {
    const url = new URL("/usage/export", window.location.origin);
    if (activeRange && !rangeError) {
      url.searchParams.set("start", activeRange.start);
      url.searchParams.set("end", activeRange.end);
    } else {
      url.searchParams.set("period", "7d");
    }
    window.open(url.toString(), "_blank");
  };

  const timezone = usageQuery.data?.timezone ?? "UTC";
  const totalRequests = usageQuery.data?.total_requests ?? 0;
  const totalTokens = usageQuery.data?.total_tokens ?? 0;
  const totalCostUsd = usageQuery.data?.total_cost_usd;
  const totalCostCents = usageQuery.data?.total_cost_cents;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Usage</h1>
          <p className="text-sm text-muted-foreground">
            Track platform-wide consumption and drill into tenants, users, or model trends.
          </p>
        </div>
        <div className="w-full space-y-2 md:max-w-xl">
          <div className="flex flex-col gap-2 md:flex-row md:items-center">
            <div className="grid flex-1 gap-2 sm:grid-cols-2">
              <div className="space-y-1">
                <Label htmlFor="usage-start">Start date</Label>
                <Input
                  id="usage-start"
                  type="date"
                  value={startInput}
                  onChange={(event) => {
                    setStartInput(event.target.value);
                  }}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="usage-end">End date</Label>
                <Input
                  id="usage-end"
                  type="date"
                  value={endInput}
                  onChange={(event) => {
                    setEndInput(event.target.value);
                  }}
                />
              </div>
            </div>
            <Button variant="outline" onClick={handleExport} className="md:self-end">
              <Download className="mr-2 h-4 w-4" /> Export CSV
            </Button>
          </div>
          {rangeError ? (
            <p className="text-xs text-destructive">{rangeError}</p>
          ) : (
            <p className="text-xs text-muted-foreground">
              Times shown in {timezone}. Range: {rangeDisplay}
            </p>
          )}
        </div>
      </div>

      <Separator />

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
                  ? formatSpendValue(totalCostUsd, totalCostCents)
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
              loading={tenantsQuery.isLoading}
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
                <DailySkeleton />
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
                          {formatSpendValue(point.cost_usd, point.cost_cents)}
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
                    {formatSpendValue(selectedTenant.budget_limit_usd ?? 0, undefined)} · Used{" "}
                    {formatSpendValue(selectedTenant.budget_used_usd ?? 0, undefined)}
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
                <DailySkeleton />
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
                <DailySkeleton />
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
                <DailySkeleton />
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
    <ExpandableDailyList<TenantDailyUsageKey, TenantDailyUsageDay>
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
          <span className="text-right">{formatSpendValue(key.cost_usd, key.cost_cents)}</span>
        </div>
      )}
      breakdownHeaders={["API key", "Requests", "Tokens", "Spend"]}
      getBreakdown={(day) => day.keys}
    />
  );
}

function UserDailyList({ days, timezone }: { days: UserDailyUsageDay[]; timezone: string }) {
  return (
    <ExpandableDailyList<UserDailyTenantUsage, UserDailyUsageDay>
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
          <span className="text-right">{formatSpendValue(tenant.cost_usd, tenant.cost_cents)}</span>
        </div>
      )}
      breakdownHeaders={["Tenant", "Requests", "Tokens", "Spend"]}
      getBreakdown={(day) => day.tenants}
    />
  );
}

function ModelDailyList({ days, timezone }: { days: ModelDailyUsageDay[]; timezone: string }) {
  return (
    <ExpandableDailyList<ModelDailyTenantUsage, ModelDailyUsageDay>
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
          <span className="text-right">{formatSpendValue(tenant.cost_usd, tenant.cost_cents)}</span>
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

  return (
    <Card>
      <CardHeader className="space-y-4">
        <div className="space-y-1">
          <CardTitle>{title}</CardTitle>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
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
      </CardHeader>
      <CardContent className="space-y-4">
        <QueryAlert
          error={breakdown.isError ? (breakdown.error as Error) : null}
          onRetry={breakdown.refetch}
        />
        {breakdown.isLoading ? (
          <Skeleton className="h-64 w-full" />
        ) : (
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
      </CardContent>
    </Card>
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
    spend: deriveSpendValue(point.cost_usd, point.cost_cents),
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
      return deriveSpendValue(item.cost_usd, item.cost_cents);
  }
}

function formatMetricDisplay(metric: UsageComparisonMetric, value: number) {
  if (metric === "spend") {
    return `$${value.toFixed(2)}`;
  }
  return value.toLocaleString();
}

function deriveSpendValue(costUSD?: number, costCents?: number) {
  if (typeof costUSD === "number" && !Number.isNaN(costUSD) && costUSD !== 0) {
    return costUSD;
  }
  if (typeof costCents === "number") {
    return costCents / 100;
  }
  return 0;
}

type DailySummary = {
  date: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
};

interface ExpandableDailyListProps<T, D extends DailySummary = DailySummary> {
  days: D[];
  timezone: string;
  breakdownHeaders: string[];
  getBreakdown: (day: D) => T[];
  renderBreakdown: (item: T) => ReactNode;
  emptyMessage: string;
}

function ExpandableDailyList<T, D extends DailySummary>({
  days,
  timezone,
  breakdownHeaders,
  getBreakdown,
  renderBreakdown,
  emptyMessage,
}: ExpandableDailyListProps<T, D>) {
  const [openState, setOpenState] = useState<Record<string, boolean>>({});

  if (!days.length) {
    return null;
  }

  const toggle = (date: string) => {
    setOpenState((prev) => ({ ...prev, [date]: !prev[date] }));
  };

  return (
    <div className="divide-y rounded-md border">
      {days.map((day) => {
        const isOpen = openState[day.date] ?? false;
        const breakdown = getBreakdown(day);
        return (
          <div key={day.date}>
            <button
              type="button"
              onClick={() => toggle(day.date)}
              className="flex w-full items-center gap-4 p-4 text-left"
            >
              <ChevronDown className={`h-4 w-4 flex-none transition ${isOpen ? "rotate-180" : ""}`} />
              <div className="flex flex-1 flex-col gap-1">
                <p className="font-medium">{formatUsageDate(day.date, timezone)}</p>
                <p className="text-xs text-muted-foreground">
                  {breakdown.length ? `${breakdown.length} entries` : emptyMessage}
                </p>
              </div>
              <div className="flex flex-none gap-6 text-sm text-muted-foreground">
                <span className="w-20 text-right">{day.requests.toLocaleString()} req</span>
                <span className="w-24 text-right">{day.tokens.toLocaleString()} tokens</span>
                <span className="w-20 text-right">{formatSpendValue(day.cost_usd, day.cost_cents)}</span>
              </div>
            </button>
            {isOpen ? (
              <div className="space-y-3 border-t bg-muted/30 p-4 text-sm">
                {breakdown.length ? (
                  <div className="space-y-2">
                    <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 text-xs uppercase text-muted-foreground">
                      {breakdownHeaders.map((header) => (
                        <span key={header}>{header}</span>
                      ))}
                    </div>
                    {breakdown.map((item, idx) => (
                      <div key={idx}>{renderBreakdown(item)}</div>
                    ))}
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground">{emptyMessage}</p>
                )}
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

function DailySkeleton() {
  return (
    <div className="space-y-2">
      {[...Array(3)].map((_, idx) => (
        <Skeleton key={idx} className="h-16 w-full" />
      ))}
    </div>
  );
}

function startOfToday() {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), now.getDate());
}

function addDays(date: Date, amount: number) {
  return new Date(date.getTime() + amount * 24 * 60 * 60 * 1000);
}

function formatDateInput(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function parseDateInput(value: string): Date | null {
  if (!value) return null;
  const [yearStr, monthStr, dayStr] = value.split("-");
  const year = Number(yearStr);
  const month = Number(monthStr);
  const day = Number(dayStr);
  if (!year || !month || !day) {
    return null;
  }
  return new Date(year, month - 1, day);
}

function formatInputDisplay(value: string) {
  const parsed = parseDateInput(value);
  if (!parsed) {
    return "Select dates";
  }
  return parsed.toLocaleDateString();
}

function deriveRangeISO(startInput: string, endInput: string) {
  const startDate = parseDateInput(startInput);
  const endDate = parseDateInput(endInput);
  if (!startDate || !endDate) {
    return { error: "Select a valid start and end date." };
  }
  if (endDate.getTime() < startDate.getTime()) {
    return { error: "End date must be after start date." };
  }
  const startISO = startDate.toISOString();
  const endISO = addDays(endDate, 1).toISOString();
  return { range: { start: startISO, end: endISO } };
}
