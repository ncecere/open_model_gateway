import { useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { ChevronDown } from "lucide-react";

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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SummaryCard } from "@/ui/kit/Cards";
import { QueryAlert } from "@/features/usage";
import { formatTokensShort } from "@/lib/numbers";
import { formatUsageDate } from "@/lib/dates";
import { useUserUsageQuery } from "../hooks/useUserData";
import { listUserModels } from "@/api/user/models";
import { getUserModelDailyUsage, getUserTenantDailyUsage } from "@/api/user/usage";
import type {
  ModelDailyTenantUsage,
  ModelDailyUsageDay,
  TenantDailyUsageDay,
  TenantDailyUsageKey,
} from "@/api/usage";


const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 4,
  maximumFractionDigits: 4,
});

const formatSpendValue = (usd?: number, cents?: number) =>
  currencyFormatter.format(
    typeof usd === "number"
      ? usd
      : typeof cents === "number"
        ? cents / 100
        : 0,
  );

export function UserUsagePage() {
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
      push(usage.personal.tenant_id, usage.personal.name ?? "Personal", usage.personal.status);
    }
    usage?.memberships.forEach((tenant) =>
      push(tenant.tenant_id, tenant.name, tenant.status),
    );
    return options;
  }, [usage?.personal, usage?.memberships]);

  const [selectedTenantId, setSelectedTenantId] = useState<string | undefined>(undefined);
  useEffect(() => {
    if (!tenantOptions.length) {
      setSelectedTenantId(undefined);
      return;
    }
    if (!selectedTenantId || !tenantOptions.find((option) => option.value === selectedTenantId)) {
      setSelectedTenantId(tenantOptions[0].value);
    }
  }, [tenantOptions, selectedTenantId]);

  const userModelsQuery = useQuery({
    queryKey: ["user-models"],
    queryFn: listUserModels,
  });
  const modelOptions = useMemo(
    () =>
      (userModelsQuery.data ?? [])
        .filter((model) => model.enabled)
        .map((model) => ({ value: model.alias, label: model.alias, hint: model.provider })),
    [userModelsQuery.data],
  );

  const [selectedModelAlias, setSelectedModelAlias] = useState<string | undefined>(undefined);
  useEffect(() => {
    if (!modelOptions.length) {
      setSelectedModelAlias(undefined);
      return;
    }
    if (!selectedModelAlias || !modelOptions.find((option) => option.value === selectedModelAlias)) {
      setSelectedModelAlias(modelOptions[0].value);
    }
  }, [modelOptions, selectedModelAlias]);

  const selectedTenantOption = tenantOptions.find((option) => option.value === selectedTenantId);
  const selectedModelOption = modelOptions.find((option) => option.value === selectedModelAlias);

  const globalRangeKey = activeRange ? `${activeRange.start}-${activeRange.end}` : "invalid";

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
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Usage</h1>
          <p className="text-sm text-muted-foreground">
            Track your API activity across personal and shared tenants.
          </p>
        </div>
        <div className="w-full space-y-2 md:max-w-xl">
          <div className="grid gap-2 sm:grid-cols-2">
            <div className="space-y-1">
              <Label htmlFor="user-usage-start">Start date</Label>
              <Input
                id="user-usage-start"
                type="date"
                value={startInput}
                onChange={(event) => setStartInput(event.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="user-usage-end">End date</Label>
              <Input
                id="user-usage-end"
                type="date"
                value={endInput}
                onChange={(event) => setEndInput(event.target.value)}
              />
            </div>
          </div>
          {rangeError ? (
            <p className="text-xs text-destructive">{rangeError}</p>
          ) : (
            <p className="text-xs text-muted-foreground">
              Times shown in {timezone}. Range: {rangeDisplay ?? "Select dates"}
            </p>
          )}
        </div>
      </div>

      {rangeError ? (
        <p className="text-sm text-destructive">Adjust the date range above to continue.</p>
      ) : null}

      {hasGlobalError ? (
        <div className="space-y-2">
          <QueryAlert error={usageQuery.isError ? (usageQuery.error as Error) : null} onRetry={usageQuery.refetch} />
          <QueryAlert error={userModelsQuery.isError ? (userModelsQuery.error as Error) : null} onRetry={userModelsQuery.refetch} />
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
            <SummaryCard
              title="Total requests"
              value={totals ? totals.requests.toLocaleString() : "—"}
              description="Across the selected window."
              loading={usageQuery.isLoading}
            />
            <SummaryCard
              title="Total spend"
              value={totals ? formatSpendValue(totals.cost_usd, totals.cost_cents) : "—"}
              description="Usage-based fees"
              loading={usageQuery.isLoading}
            />
            <SummaryCard
              title="Tokens processed"
              value={totals ? formatTokensShort(totals.tokens) : "—"}
              description="Prompt + completion"
              loading={usageQuery.isLoading}
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
                    {dailyRows.map((point) => (
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
                Inspect a specific tenant (personal or shared) with per-key breakdowns.
              </p>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="max-w-sm space-y-2">
                <Label>Tenant</Label>
                <Select value={selectedTenantId} onValueChange={setSelectedTenantId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select tenant" />
                  </SelectTrigger>
                  <SelectContent>
                    {tenantOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        <div className="flex items-center justify-between gap-2">
                          <span>{option.label}</span>
                          {option.hint ? (
                            <span className="text-xs text-muted-foreground">{option.hint}</span>
                          ) : null}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {selectedTenantOption?.hint ? (
                  <p className="text-xs text-muted-foreground">{selectedTenantOption.hint}</p>
                ) : null}
              </div>
              <QueryAlert
                error={tenantDailyQuery.isError ? (tenantDailyQuery.error as Error) : null}
                onRetry={tenantDailyQuery.refetch}
              />
              {!selectedTenantId ? (
                <p className="text-sm text-muted-foreground">Select a tenant to begin.</p>
              ) : rangeError ? (
                <p className="text-sm text-muted-foreground">{rangeError}</p>
              ) : tenantDailyQuery.isLoading ? (
                <DailySkeleton />
              ) : tenantDailyQuery.data && tenantDailyQuery.data.days.length ? (
                <>
                  <p className="text-xs text-muted-foreground">
                    Times shown in {tenantDailyQuery.data.timezone}. Daily totals include all of your keys
                    for this tenant.
                  </p>
                  <TenantDailyList
                    days={tenantDailyQuery.data.days}
                    timezone={tenantDailyQuery.data.timezone}
                  />
                </>
              ) : (
                <p className="text-sm text-muted-foreground">No activity in this range.</p>
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
                <Select value={selectedModelAlias} onValueChange={setSelectedModelAlias}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select model" />
                  </SelectTrigger>
                  <SelectContent>
                    {modelOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        <div className="flex items-center justify-between gap-2">
                          <span>{option.label}</span>
                          {option.hint ? (
                            <span className="text-xs text-muted-foreground">{option.hint}</span>
                          ) : null}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {selectedModelOption?.hint ? (
                  <p className="text-xs text-muted-foreground">{selectedModelOption.hint}</p>
                ) : null}
              </div>
              <QueryAlert
                error={modelDailyQuery.isError ? (modelDailyQuery.error as Error) : null}
                onRetry={modelDailyQuery.refetch}
              />
              {!selectedModelAlias ? (
                <p className="text-sm text-muted-foreground">Select a model to begin.</p>
              ) : rangeError ? (
                <p className="text-sm text-muted-foreground">{rangeError}</p>
              ) : modelDailyQuery.isLoading ? (
                <DailySkeleton />
              ) : modelDailyQuery.data && modelDailyQuery.data.days.length ? (
                <>
                  <p className="text-xs text-muted-foreground">
                    Times shown in {modelDailyQuery.data.timezone}. Daily totals only include tenants you
                    belong to.
                  </p>
                  <ModelDailyList
                    days={modelDailyQuery.data.days}
                    timezone={modelDailyQuery.data.timezone}
                  />
                </>
              ) : (
                <p className="text-sm text-muted-foreground">No activity in this range.</p>
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
