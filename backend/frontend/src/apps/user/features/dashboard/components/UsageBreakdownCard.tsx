import { useState, useMemo } from "react";
import {
  Bar,
  BarChart,
  ResponsiveContainer,
  XAxis,
  YAxis,
  Tooltip,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { formatTokensShort } from "@/lib/numbers";

interface UsagePoint {
  date: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
}

interface UsageBreakdownCardProps {
  series: UsagePoint[];
  scopeName: string;
  loading?: boolean;
}

type MetricKey = "requests" | "tokens" | "spend";

const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

export function UsageBreakdownCard({
  series,
  scopeName,
  loading,
}: UsageBreakdownCardProps) {
  const [metric, setMetric] = useState<MetricKey>("requests");

  const chartData = useMemo(() => {
    return series.map((point) => ({
      date: new Date(point.date).toLocaleDateString(undefined, {
        month: "short",
        day: "numeric",
      }),
      requests: point.requests,
      tokens: point.tokens,
      spend: point.cost_usd ?? point.cost_cents / 100,
    }));
  }, [series]);

  const formatValue = (value: number) => {
    switch (metric) {
      case "requests":
        return value.toLocaleString();
      case "tokens":
        return formatTokensShort(value);
      case "spend":
        return currencyFormatter.format(value);
    }
  };

  const getLabel = () => {
    switch (metric) {
      case "requests":
        return "Requests";
      case "tokens":
        return "Tokens";
      case "spend":
        return "Spend";
    }
  };

  if (loading) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle>Usage Breakdown</CardTitle>
          <Skeleton className="h-9 w-32" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[200px] w-full" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <div>
          <CardTitle>Usage Breakdown</CardTitle>
          <p className="text-sm text-muted-foreground">
            Daily {getLabel().toLowerCase()} for {scopeName}
          </p>
        </div>
        <Select value={metric} onValueChange={(v) => setMetric(v as MetricKey)}>
          <SelectTrigger className="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="requests">Requests</SelectItem>
            <SelectItem value="tokens">Tokens</SelectItem>
            <SelectItem value="spend">Spend</SelectItem>
          </SelectContent>
        </Select>
      </CardHeader>
      <CardContent>
        {chartData.length === 0 ? (
          <div className="flex h-[200px] items-center justify-center text-sm text-muted-foreground">
            No usage data available
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={chartData}>
              <XAxis
                dataKey="date"
                tick={{ fontSize: 11 }}
                tickLine={false}
                axisLine={false}
              />
              <YAxis
                tick={{ fontSize: 11 }}
                tickLine={false}
                axisLine={false}
                tickFormatter={(value) =>
                  metric === "spend"
                    ? `$${value}`
                    : metric === "tokens"
                      ? formatTokensShort(value)
                      : value.toLocaleString()
                }
                width={50}
              />
              <Tooltip
                content={({ active, payload }) => {
                  if (!active || !payload?.length) return null;
                  const data = payload[0].payload;
                  return (
                    <div className="rounded-lg border bg-popover px-3 py-2 text-xs shadow-sm">
                      <div className="font-medium mb-1">{data.date}</div>
                      <div className="text-muted-foreground">
                        {getLabel()}: {formatValue(data[metric])}
                      </div>
                    </div>
                  );
                }}
              />
              <Bar
                dataKey={metric}
                fill="hsl(var(--primary))"
                radius={[4, 4, 0, 0]}
              />
            </BarChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  );
}
