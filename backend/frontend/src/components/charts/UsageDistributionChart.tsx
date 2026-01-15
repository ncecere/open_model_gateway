import { useMemo } from "react";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";

import { Skeleton } from "@/components/ui/skeleton";
import { formatUsageUSD, formatTokensShort } from "@/features/usage";
import { cn } from "@/lib/utils";

export interface DistributionItem {
  id: string;
  label: string;
  value: number;
  requests?: number;
  tokens?: number;
  cost_usd?: number;
  cost_cents?: number;
}

export type DistributionMetric = "spend" | "tokens" | "requests";

const COLORS = [
  "hsl(var(--chart-1))",
  "hsl(var(--chart-2))",
  "hsl(var(--chart-3))",
  "hsl(var(--chart-4))",
  "hsl(var(--chart-5))",
  "hsl(var(--chart-6))",
  "hsl(var(--chart-7))",
  "hsl(var(--chart-8))",
];

interface UsageDistributionChartProps {
  data: DistributionItem[];
  metric: DistributionMetric;
  loading?: boolean;
  height?: number;
  showLegend?: boolean;
  title?: string;
  emptyMessage?: string;
  className?: string;
}

function formatValue(value: number, metric: DistributionMetric): string {
  switch (metric) {
    case "spend":
      return formatUsageUSD(value);
    case "tokens":
      return formatTokensShort(value);
    case "requests":
      return value.toLocaleString();
  }
}

function getMetricLabel(metric: DistributionMetric): string {
  switch (metric) {
    case "spend":
      return "total spend";
    case "tokens":
      return "total tokens";
    case "requests":
      return "total requests";
  }
}

export function UsageDistributionChart({
  data,
  metric,
  loading,
  height = 280,
  showLegend = true,
  emptyMessage,
  className,
}: UsageDistributionChartProps) {
  if (loading) {
    return <Skeleton className={cn("w-full", className)} style={{ height }} />;
  }

  const chartData = useMemo(() => {
    const sorted = [...data].sort((a, b) => b.value - a.value);
    const top7 = sorted.slice(0, 7);
    const others = sorted.slice(7);
    const otherValue = others.reduce((sum, item) => sum + item.value, 0);

    const result = top7.map((item, index) => ({
      ...item,
      color: COLORS[index % COLORS.length],
    }));

    if (otherValue > 0) {
      result.push({
        id: "other",
        label: "Other",
        value: otherValue,
        color: COLORS[7],
      });
    }

    return result;
  }, [data]);

  const totalValue = useMemo(
    () => chartData.reduce((sum, item) => sum + item.value, 0),
    [chartData]
  );

  if (!chartData.length || totalValue === 0) {
    return (
      <div
        className={cn(
          "flex items-center justify-center text-sm text-muted-foreground",
          className
        )}
        style={{ height }}
      >
        {emptyMessage ?? `No ${metric} data available.`}
      </div>
    );
  }

  return (
    <div className={cn("flex flex-col gap-4", className)} style={{ height }}>
      <div className="flex-1 min-h-0">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={chartData}
              cx="50%"
              cy="50%"
              innerRadius="60%"
              outerRadius="85%"
              paddingAngle={2}
              dataKey="value"
              nameKey="label"
              strokeWidth={0}
            >
              {chartData.map((entry) => (
                <Cell key={entry.id} fill={entry.color} />
              ))}
            </Pie>
            <Tooltip
              content={({ active, payload }) => {
                if (!active || !payload?.length) return null;
                const item = payload[0].payload as DistributionItem & {
                  color: string;
                };
                const percentage = ((item.value / totalValue) * 100).toFixed(1);
                return (
                  <div className="rounded-lg border bg-popover px-3 py-2 text-xs shadow-sm">
                    <div className="flex items-center gap-2 mb-1">
                      <span
                        className="h-2 w-2 rounded-full"
                        style={{ backgroundColor: item.color }}
                      />
                      <span className="font-medium">{item.label}</span>
                    </div>
                    <div className="text-muted-foreground space-y-0.5">
                      <div>
                        {formatValue(item.value, metric)} ({percentage}%)
                      </div>
                      {metric !== "requests" && item.requests != null && (
                        <div>{item.requests.toLocaleString()} requests</div>
                      )}
                      {metric !== "tokens" && item.tokens != null && (
                        <div>{formatTokensShort(item.tokens)} tokens</div>
                      )}
                      {metric !== "spend" && item.cost_usd != null && (
                        <div>{formatUsageUSD(item.cost_usd)} spend</div>
                      )}
                    </div>
                  </div>
                );
              }}
            />
            {/* Center text */}
            <text
              x="50%"
              y="48%"
              textAnchor="middle"
              dominantBaseline="middle"
              className="fill-foreground text-lg font-semibold"
            >
              {formatValue(totalValue, metric)}
            </text>
            <text
              x="50%"
              y="56%"
              textAnchor="middle"
              dominantBaseline="middle"
              className="fill-muted-foreground text-xs"
            >
              {getMetricLabel(metric)}
            </text>
          </PieChart>
        </ResponsiveContainer>
      </div>
      {showLegend && (
        <div className="flex flex-wrap gap-x-4 gap-y-1 justify-center">
          {chartData.slice(0, 6).map((item) => (
            <div key={item.id} className="flex items-center gap-1.5 text-xs">
              <span
                className="h-2 w-2 rounded-full shrink-0"
                style={{ backgroundColor: item.color }}
              />
              <span className="text-muted-foreground truncate max-w-[80px]">
                {item.label}
              </span>
            </div>
          ))}
          {chartData.length > 6 && (
            <span className="text-xs text-muted-foreground">
              +{chartData.length - 6} more
            </span>
          )}
        </div>
      )}
    </div>
  );
}
