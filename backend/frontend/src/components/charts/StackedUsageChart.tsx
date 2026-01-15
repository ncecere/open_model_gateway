import { useMemo } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  XAxis,
  YAxis,
} from "recharts";

import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import { formatUsageDate } from "@/lib/dates";
import { cn } from "@/lib/utils";

export interface StackedSeriesData {
  id: string;
  label: string;
  points: Array<{
    date: string;
    value: number;
  }>;
}

export type StackedMetric = "spend" | "tokens" | "requests";

interface StackedUsageChartProps {
  series: StackedSeriesData[];
  metric: StackedMetric;
  timezone?: string;
  loading?: boolean;
  height?: number;
  className?: string;
}

const CHART_COLORS = [
  "hsl(var(--chart-1))",
  "hsl(var(--chart-2))",
  "hsl(var(--chart-3))",
  "hsl(var(--chart-4))",
  "hsl(var(--chart-5))",
];

function formatMetricValue(value: number, metric: StackedMetric): string {
  switch (metric) {
    case "spend":
      return `$${value.toFixed(2)}`;
    case "tokens":
      if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
      if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
      return value.toLocaleString();
    case "requests":
      return value.toLocaleString();
  }
}

export function StackedUsageChart({
  series,
  metric,
  timezone = "UTC",
  loading,
  height = 320,
  className,
}: StackedUsageChartProps) {
  if (loading) {
    return <Skeleton className={cn("w-full", className)} style={{ height }} />;
  }

  // Transform series data into stacked format
  const { chartData, chartConfig, seriesKeys } = useMemo(() => {
    if (!series.length) {
      return { chartData: [], chartConfig: {} as ChartConfig, seriesKeys: [] };
    }

    // Collect all unique dates
    const allDates = new Set<string>();
    series.forEach((s) => s.points.forEach((p) => allDates.add(p.date)));
    const sortedDates = Array.from(allDates).sort();

    // Build chart data with all series values
    const data = sortedDates.map((date) => {
      const row: Record<string, number | string> = { date };
      series.forEach((s) => {
        const point = s.points.find((p) => p.date === date);
        row[s.id] = point?.value ?? 0;
      });
      return row;
    });

    // Build chart config
    const config: ChartConfig = {};
    const keys: string[] = [];
    series.forEach((s, i) => {
      config[s.id] = {
        label: s.label,
        color: CHART_COLORS[i % CHART_COLORS.length],
      };
      keys.push(s.id);
    });

    return { chartData: data, chartConfig: config, seriesKeys: keys };
  }, [series]);

  if (!chartData.length) {
    return (
      <div
        className={cn(
          "flex items-center justify-center text-sm text-muted-foreground",
          className
        )}
        style={{ height }}
      >
        No data available for the selected period.
      </div>
    );
  }

  const formatDate = (value: string) => formatUsageDate(value, timezone);

  // Calculate max value for Y-axis
  const maxValue = Math.max(
    ...chartData.map((row) =>
      seriesKeys.reduce((sum, key) => sum + (Number(row[key]) || 0), 0)
    ),
    1
  );

  return (
    <div className={className} style={{ height }}>
      <ChartContainer config={chartConfig} className="h-full w-full">
        <ResponsiveContainer>
          <AreaChart
            data={chartData}
            margin={{ left: 12, right: 12, top: 16, bottom: 8 }}
          >
            <defs>
              {seriesKeys.map((key, i) => (
                <linearGradient
                  key={key}
                  id={`stacked-gradient-${key}`}
                  x1="0"
                  y1="0"
                  x2="0"
                  y2="1"
                >
                  <stop
                    offset="5%"
                    stopColor={CHART_COLORS[i % CHART_COLORS.length]}
                    stopOpacity={0.6}
                  />
                  <stop
                    offset="95%"
                    stopColor={CHART_COLORS[i % CHART_COLORS.length]}
                    stopOpacity={0.1}
                  />
                </linearGradient>
              ))}
            </defs>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
            <XAxis
              dataKey="date"
              axisLine={false}
              tickLine={false}
              interval={Math.max(0, Math.floor(chartData.length / 7) - 1)}
              tickFormatter={(value) => formatDate(value as string)}
            />
            <YAxis
              axisLine={false}
              tickLine={false}
              tickFormatter={(value) => formatMetricValue(value, metric)}
              domain={[0, maxValue]}
            />
            <ChartTooltip
              cursor={{ strokeDasharray: "3 3" }}
              content={
                <ChartTooltipContent
                  indicator="dot"
                  labelFormatter={(value) => formatDate(value?.toString() ?? "")}
                  valueFormatter={(payload) =>
                    formatMetricValue(Number(payload.value ?? 0), metric)
                  }
                />
              }
            />
            {seriesKeys.map((key, i) => (
              <Area
                key={key}
                type="monotone"
                dataKey={key}
                stackId="1"
                stroke={CHART_COLORS[i % CHART_COLORS.length]}
                fill={`url(#stacked-gradient-${key})`}
                strokeWidth={2}
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      </ChartContainer>
    </div>
  );
}
