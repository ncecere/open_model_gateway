import { useMemo, useState } from "react";
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
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Label } from "@/components/ui/label";
import { PageHeader } from "@/components/layouts";
import {
  DateRangePresets,
  EnhancedSummaryCard,
  QueryAlert,
  UsageDailySkeleton,
  UsageDailyTable,
  formatTokensShort,
  formatUsageUSD,
  useUsageRange,
} from "@/features/usage";
import { formatUsageDate } from "@/lib/dates";
import { useDefaultSelection } from "@/hooks/useDefaultSelection";
import { useUserUsageQuery } from "../hooks/useUserData";
import { listUserModels } from "@/api/user/models";
import {
  getUserModelDailyUsage,
  getUserTenantDailyUsage,
} from "@/api/user/usage";
import type {
  ModelDailyTenantUsage,
  ModelDailyUsageDay,
  TenantDailyUsageDay,
  TenantDailyUsageKey,
} from "@/api/usage";

export function UserUsagePage() {
  const {
    startInput,
    endInput,
    setStartInput,
    setEndInput,
    range: activeRange,
    rangeError,
    rangeDisplay,
  } = useUsageRange({ defaultDays: 6 });
  const usageQuery = useUserUsageQuery(
    activeRange && !rangeError
      ? { start: activeRange.start, end: activeRange.end }
      : {},
    { enabled: Boolean(activeRange && !rangeError) },
  );
  const usage = usageQuery.data;
  const timezone = usage?.timezone ?? "UTC";
  const totals = usage?.totals;
  const dailyRows = useMemo(() => {
    if (usage?.selected_scope?.series?.length) {
      return usage.selected_scope.series;
    }
    return usage?.personal_series ?? [];
  }, [usage?.personal_series, usage?.selected_scope]);

  const tenantOptions = useMemo(() => {
    const options: { value: string; label: string; hint?: string }[] = [];
    const seen = new Set<string>();
    const push = (id?: string, label?: string, hint?: string) => {
      if (!id || seen.has(id)) {
        return;
      }
      seen.add(id);
      options.push({ value: id, label: label ?? id, hint });
    };
    if (usage?.personal?.tenant_id) {
      push(
        usage.personal.tenant_id,
        usage.personal.name ?? "Personal",
        usage.personal.status,
      );
    }
    usage?.memberships.forEach((tenant) =>
      push(tenant.tenant_id, tenant.name, tenant.status),
    );
    return options;
  }, [usage?.personal, usage?.memberships]);

  const [selectedTenantId, setSelectedTenantId] = useState<string | undefined>(
    undefined,
  );
  useDefaultSelection({
    items: tenantOptions,
    selected: selectedTenantId,
    onChange: setSelectedTenantId,
    getValue: (option) => option.value,
  });

  const userModelsQuery = useQuery({
    queryKey: ["user-models"],
    queryFn: () => listUserModels(),
  });
  const modelOptions = useMemo(
    () =>
      (userModelsQuery.data ?? [])
        .filter((model) => model.enabled)
        .map((model) => ({
          value: model.alias,
          label: model.alias,
          hint: model.provider,
        })),
    [userModelsQuery.data],
  );

  const [selectedModelAlias, setSelectedModelAlias] = useState<
    string | undefined
  >(undefined);
  useDefaultSelection({
    items: modelOptions,
    selected: selectedModelAlias,
    onChange: setSelectedModelAlias,
    getValue: (option) => option.value,
  });

  const selectedTenantOption = tenantOptions.find(
    (option) => option.value === selectedTenantId,
  );
  const selectedModelOption = modelOptions.find(
    (option) => option.value === selectedModelAlias,
  );

  const globalRangeKey = activeRange
    ? `${activeRange.start}-${activeRange.end}`
    : "invalid";

  const tenantDailyQuery = useQuery({
    queryKey: ["user-tenant-daily", selectedTenantId, globalRangeKey],
    queryFn: () =>
      getUserTenantDailyUsage({
        tenantId: selectedTenantId as string,
        start: activeRange?.start as string,
        end: activeRange?.end as string,
      }),
    enabled: Boolean(selectedTenantId && activeRange && !rangeError),
  });

  const modelDailyQuery = useQuery({
    queryKey: ["user-model-daily", selectedModelAlias, globalRangeKey],
    queryFn: () =>
      getUserModelDailyUsage({
        modelAlias: selectedModelAlias as string,
        start: activeRange?.start as string,
        end: activeRange?.end as string,
      }),
    enabled: Boolean(selectedModelAlias && activeRange && !rangeError),
  });

  const hasGlobalError = Boolean(usageQuery.isError || userModelsQuery.isError);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Usage"
        description="Track your API activity across personal and shared tenants."
      />

      <DateRangePresets
        startInput={startInput}
        endInput={endInput}
        onStartChange={setStartInput}
        onEndChange={setEndInput}
        rangeError={rangeError}
        rangeDisplay={rangeDisplay}
        timezone={timezone}
        idPrefix="user-usage"
      />

      {hasGlobalError ? (
        <div className="space-y-2">
          <QueryAlert
            error={usageQuery.isError ? (usageQuery.error as Error) : null}
            onRetry={usageQuery.refetch}
          />
          <QueryAlert
            error={
              userModelsQuery.isError ? (userModelsQuery.error as Error) : null
            }
            onRetry={userModelsQuery.refetch}
          />
        </div>
      ) : null}

      <Tabs defaultValue="overview" className="space-y-6">
        <TabsList className="w-fit">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="tenants">Tenants</TabsTrigger>
          <TabsTrigger value="models">Models</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          <section className="grid gap-4 md:grid-cols-3">
            <EnhancedSummaryCard
              title="Total requests"
              value={totals ? totals.requests.toLocaleString() : "—"}
              description="Across the selected window."
              loading={usageQuery.isLoading}
              sparklineData={dailyRows?.map((p) => ({ value: p.requests })) ?? []}
              sparklineColor="hsl(var(--chart-1))"
              upIsGood={true}
            />
            <EnhancedSummaryCard
              title="Total spend"
              value={
                totals
                  ? formatUsageUSD(totals.cost_usd, totals.cost_cents, {
                      minimumFractionDigits: 4,
                      maximumFractionDigits: 4,
                    })
                  : "—"
              }
              description="Usage-based fees"
              loading={usageQuery.isLoading}
              sparklineData={dailyRows?.map((p) => ({
                value: p.cost_usd ?? (p.cost_cents ? p.cost_cents / 100 : 0),
              })) ?? []}
              sparklineColor="hsl(var(--chart-3))"
              upIsGood={false}
            />
            <EnhancedSummaryCard
              title="Tokens processed"
              value={totals ? formatTokensShort(totals.tokens) : "—"}
              description="Prompt + completion"
              loading={usageQuery.isLoading}
              sparklineData={dailyRows?.map((p) => ({ value: p.tokens })) ?? []}
              sparklineColor="hsl(var(--chart-2))"
              upIsGood={true}
            />
          </section>

          <Card>
            <CardHeader>
              <CardTitle>Daily breakdown</CardTitle>
              <p className="text-sm text-muted-foreground">
                Activity for your selected scope. Times shown in {timezone}.
              </p>
            </CardHeader>
            <CardContent>
              {usageQuery.isLoading ? (
                <div className="space-y-3">
                  <Skeleton className="h-12 w-full" />
                  <Skeleton className="h-12 w-full" />
                  <Skeleton className="h-12 w-full" />
                </div>
              ) : dailyRows.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No usage recorded.
                </p>
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
                    {dailyRows.map((point) => (
                      <TableRow key={point.date}>
                        <TableCell>
                          {formatUsageDate(point.date, timezone)}
                        </TableCell>
                        <TableCell className="text-right">
                          {point.requests.toLocaleString()}
                        </TableCell>
                        <TableCell className="text-right">
                          {point.tokens.toLocaleString()}
                        </TableCell>
                        <TableCell className="text-right">
                          {formatUsageUSD(point.cost_usd, point.cost_cents, {
                            minimumFractionDigits: 4,
                            maximumFractionDigits: 4,
                          })}
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
                Inspect a specific tenant (personal or shared) with per-key
                breakdowns.
              </p>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="max-w-sm space-y-2">
                <Label>Tenant</Label>
                <Select
                  value={selectedTenantId}
                  onValueChange={setSelectedTenantId}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select tenant" />
                  </SelectTrigger>
                  <SelectContent>
                    {tenantOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        <div className="flex items-center justify-between gap-2">
                          <span>{option.label}</span>
                          {option.hint ? (
                            <span className="text-xs text-muted-foreground">
                              {option.hint}
                            </span>
                          ) : null}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {selectedTenantOption?.hint ? (
                  <p className="text-xs text-muted-foreground">
                    {selectedTenantOption.hint}
                  </p>
                ) : null}
              </div>
              <QueryAlert
                error={
                  tenantDailyQuery.isError
                    ? (tenantDailyQuery.error as Error)
                    : null
                }
                onRetry={tenantDailyQuery.refetch}
              />
              {!selectedTenantId ? (
                <p className="text-sm text-muted-foreground">
                  Select a tenant to begin.
                </p>
              ) : rangeError ? (
                <p className="text-sm text-muted-foreground">{rangeError}</p>
              ) : tenantDailyQuery.isLoading ? (
                <UsageDailySkeleton />
              ) : tenantDailyQuery.data && tenantDailyQuery.data.days.length ? (
                <>
                  <p className="text-xs text-muted-foreground">
                    Times shown in {tenantDailyQuery.data.timezone}. Daily
                    totals include all of your keys for this tenant.
                  </p>
                  <TenantDailyList
                    days={tenantDailyQuery.data.days}
                    timezone={tenantDailyQuery.data.timezone}
                  />
                </>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No activity in this range.
                </p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="models" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Model daily usage</CardTitle>
              <p className="text-sm text-muted-foreground">
                See how your tenants are using a specific model alias.
              </p>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="max-w-sm space-y-2">
                <Label>Model</Label>
                <Select
                  value={selectedModelAlias}
                  onValueChange={setSelectedModelAlias}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select model" />
                  </SelectTrigger>
                  <SelectContent>
                    {modelOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        <div className="flex items-center justify-between gap-2">
                          <span>{option.label}</span>
                          {option.hint ? (
                            <span className="text-xs text-muted-foreground">
                              {option.hint}
                            </span>
                          ) : null}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {selectedModelOption?.hint ? (
                  <p className="text-xs text-muted-foreground">
                    {selectedModelOption.hint}
                  </p>
                ) : null}
              </div>
              <QueryAlert
                error={
                  modelDailyQuery.isError
                    ? (modelDailyQuery.error as Error)
                    : null
                }
                onRetry={modelDailyQuery.refetch}
              />
              {!selectedModelAlias ? (
                <p className="text-sm text-muted-foreground">
                  Select a model to begin.
                </p>
              ) : rangeError ? (
                <p className="text-sm text-muted-foreground">{rangeError}</p>
              ) : modelDailyQuery.isLoading ? (
                <UsageDailySkeleton />
              ) : modelDailyQuery.data && modelDailyQuery.data.days.length ? (
                <>
                  <p className="text-xs text-muted-foreground">
                    Times shown in {modelDailyQuery.data.timezone}. Daily totals
                    only include tenants you belong to.
                  </p>
                  <ModelDailyList
                    days={modelDailyQuery.data.days}
                    timezone={modelDailyQuery.data.timezone}
                  />
                </>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No activity in this range.
                </p>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function TenantDailyList({
  days,
  timezone,
}: {
  days: TenantDailyUsageDay[];
  timezone: string;
}) {
  return (
    <UsageDailyTable<TenantDailyUsageKey, TenantDailyUsageDay>
      days={days}
      timezone={timezone}
      emptyMessage="No key activity for this day."
      renderBreakdown={(key) => (
        <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 rounded border bg-background/80 px-3 py-2">
          <div>
            <p className="font-medium">
              {key.api_key_name || key.api_key_prefix || "Unnamed key"}
            </p>
            <p className="text-xs text-muted-foreground">
              {key.api_key_prefix || key.api_key_id}
            </p>
          </div>
          <span className="text-right">{key.requests.toLocaleString()}</span>
          <span className="text-right">{key.tokens.toLocaleString()}</span>
          <span className="text-right">
            {formatUsageUSD(key.cost_usd, key.cost_cents, {
              minimumFractionDigits: 4,
              maximumFractionDigits: 4,
            })}
          </span>
        </div>
      )}
      breakdownHeaders={["API key", "Requests", "Tokens", "Spend"]}
      getBreakdown={(day) => day.keys}
      currencyOptions={{ minimumFractionDigits: 4, maximumFractionDigits: 4 }}
    />
  );
}

function ModelDailyList({
  days,
  timezone,
}: {
  days: ModelDailyUsageDay[];
  timezone: string;
}) {
  return (
    <UsageDailyTable<ModelDailyTenantUsage, ModelDailyUsageDay>
      days={days}
      timezone={timezone}
      emptyMessage="No tenant activity for this model on this day."
      renderBreakdown={(tenant) => (
        <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 rounded border bg-background/80 px-3 py-2">
          <div>
            <p className="font-medium">
              {tenant.tenant_name || tenant.tenant_id || "Tenant"}
            </p>
            <p className="text-xs text-muted-foreground">{tenant.tenant_id}</p>
          </div>
          <span className="text-right">{tenant.requests.toLocaleString()}</span>
          <span className="text-right">{tenant.tokens.toLocaleString()}</span>
          <span className="text-right">
            {formatUsageUSD(tenant.cost_usd, tenant.cost_cents, {
              minimumFractionDigits: 4,
              maximumFractionDigits: 4,
            })}
          </span>
        </div>
      )}
      breakdownHeaders={["Tenant", "Requests", "Tokens", "Spend"]}
      getBreakdown={(day) => day.tenants}
      currencyOptions={{ minimumFractionDigits: 4, maximumFractionDigits: 4 }}
    />
  );
}
