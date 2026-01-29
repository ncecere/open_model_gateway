import { useMemo } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useUsageBreakdown } from "@/api/hooks/useUsage";
import { TokensDonutChart } from "@/components/charts/TokensDonutChart";
import { formatTokensShort } from "@/lib/numbers";
import type { QueryParams } from "@/components/ui/period-selector";

interface BreakdownTabsProps {
  queryParams: QueryParams;
  periodLabel: string;
}

import { currencyFormatter } from "@/lib/formatters";

export function BreakdownTabs({ queryParams, periodLabel }: BreakdownTabsProps) {
  const modelBreakdown = useUsageBreakdown(
    { group: "model", ...queryParams, limit: 10 },
    { staleTime: 60_000 }
  );

  const tenantBreakdown = useUsageBreakdown(
    { group: "tenant", ...queryParams, limit: 10 },
    { staleTime: 60_000 }
  );

  const userBreakdown = useUsageBreakdown(
    { group: "user", ...queryParams, limit: 10 },
    { staleTime: 60_000 }
  );

  const modelData = useMemo(() => {
    return (modelBreakdown.data?.items ?? []).map((item) => ({
      id: item.id,
      label: item.label,
      tokens: item.tokens,
      requests: item.requests,
      cost_usd: item.cost_usd ?? item.cost_cents / 100,
    }));
  }, [modelBreakdown.data?.items]);

  const tenantData = useMemo(() => {
    return (tenantBreakdown.data?.items ?? []).map((item) => ({
      id: item.id,
      label: item.label,
      tokens: item.tokens,
      requests: item.requests,
      cost_usd: item.cost_usd ?? item.cost_cents / 100,
    }));
  }, [tenantBreakdown.data?.items]);

  const userData = useMemo(() => {
    return (userBreakdown.data?.items ?? []).map((item) => ({
      id: item.id,
      label: item.label,
      tokens: item.tokens,
      requests: item.requests,
      cost_usd: item.cost_usd ?? item.cost_cents / 100,
    }));
  }, [userBreakdown.data?.items]);

  return (
    <Card className="animate-fade-in-up stagger-6">
      <CardHeader>
        <CardTitle className="text-base font-semibold">Usage Breakdown</CardTitle>
        <p className="text-sm text-muted-foreground">
          Distribution by dimension for {periodLabel.toLowerCase()}
        </p>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="model" className="space-y-4">
          <TabsList>
            <TabsTrigger value="model">By Model</TabsTrigger>
            <TabsTrigger value="tenant">By Tenant</TabsTrigger>
            <TabsTrigger value="user">By User</TabsTrigger>
          </TabsList>

          <TabsContent value="model" className="space-y-4">
            <div className="grid gap-4 lg:grid-cols-2">
              <TokensDonutChart
                data={modelData}
                loading={modelBreakdown.isLoading}
                height={240}
                showLegend={false}
              />
              <TopItemsList
                items={modelData}
                loading={modelBreakdown.isLoading}
              />
            </div>
          </TabsContent>

          <TabsContent value="tenant" className="space-y-4">
            <div className="grid gap-4 lg:grid-cols-2">
              <TokensDonutChart
                data={tenantData}
                loading={tenantBreakdown.isLoading}
                height={240}
                showLegend={false}
              />
              <TopItemsList
                items={tenantData}
                loading={tenantBreakdown.isLoading}
              />
            </div>
          </TabsContent>

          <TabsContent value="user" className="space-y-4">
            <div className="grid gap-4 lg:grid-cols-2">
              <TokensDonutChart
                data={userData}
                loading={userBreakdown.isLoading}
                height={240}
                showLegend={false}
              />
              <TopItemsList
                items={userData}
                loading={userBreakdown.isLoading}
              />
            </div>
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}

interface TopItemsListProps {
  items: {
    id: string;
    label: string;
    tokens: number;
    requests: number;
    cost_usd: number;
  }[];
  loading?: boolean;
}

function TopItemsList({ items, loading }: TopItemsListProps) {
  if (loading) {
    return (
      <div className="space-y-2">
        {[...Array(5)].map((_, i) => (
          <div
            key={i}
            className="h-12 animate-pulse rounded-md bg-muted"
          />
        ))}
      </div>
    );
  }

  if (!items.length) {
    return (
      <div className="flex h-[240px] items-center justify-center text-sm text-muted-foreground">
        No data available
      </div>
    );
  }

  const maxTokens = Math.max(...items.map((item) => item.tokens));

  return (
    <div className="space-y-2 max-h-[240px] overflow-y-auto pr-1">
      {items.slice(0, 8).map((item, index) => {
        const percentage = maxTokens > 0 ? (item.tokens / maxTokens) * 100 : 0;
        return (
          <div
            key={item.id}
            className="relative overflow-hidden rounded-md bg-muted/50 px-3 py-2"
          >
            <div
              className="absolute inset-y-0 left-0 bg-primary/10"
              style={{ width: `${percentage}%` }}
            />
            <div className="relative flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-xs font-medium text-muted-foreground w-5">
                  {index + 1}.
                </span>
                <span className="text-sm font-medium truncate">
                  {item.label}
                </span>
              </div>
              <div className="text-right text-xs text-muted-foreground shrink-0">
                <div className="font-medium text-foreground">
                  {formatTokensShort(item.tokens)}
                </div>
                <div>
                  {item.requests.toLocaleString()} reqs · {currencyFormatter.format(item.cost_usd)}
                </div>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
