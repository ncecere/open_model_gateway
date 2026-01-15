import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  XAxis,
  YAxis,
} from "recharts";

import type { ChartConfig } from "@/components/ui/chart";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";

export interface RequestsChartDatum {
  date: string;
  requests: number;
  tokens?: number;
  cost_usd?: number;
  cost_cents?: number;
}

const chartConfig = {
  requests: { label: "Requests", color: "hsl(var(--chart-1))" },
} satisfies ChartConfig;

interface RequestsAreaChartProps {
  data: RequestsChartDatum[];
  loading?: boolean;
  height?: number;
  showGrid?: boolean;
}

export function RequestsAreaChart({
  data,
  loading,
  height = 280,
  showGrid = true,
}: RequestsAreaChartProps) {
  if (loading) {
    return <Skeleton className="w-full" style={{ height }} />;
  }

  if (!data.length) {
    return (
      <div
        className="flex items-center justify-center text-sm text-muted-foreground"
        style={{ height }}
      >
        No data available for the selected period.
      </div>
    );
  }

  const formatDate = (value: string) => {
    const date = new Date(value);
    return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  };

  const maxValue = Math.max(...data.map((item) => item.requests), 1);

  return (
    <ChartContainer config={chartConfig} className="w-full" style={{ height }}>
      <ResponsiveContainer>
        <AreaChart data={data} margin={{ left: 0, right: 12, top: 16, bottom: 8 }}>
          <defs>
            <linearGradient id="requests-gradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="hsl(var(--chart-1))" stopOpacity={0.4} />
              <stop offset="95%" stopColor="hsl(var(--chart-1))" stopOpacity={0.05} />
            </linearGradient>
          </defs>
          {showGrid && <CartesianGrid strokeDasharray="3 3" className="stroke-muted/50" />}
          <XAxis
            dataKey="date"
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 11 }}
            tickFormatter={formatDate}
            interval={Math.max(0, Math.floor(data.length / 7) - 1)}
          />
          <YAxis
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 11 }}
            tickFormatter={(value) => value.toLocaleString()}
            domain={[0, maxValue]}
            width={50}
          />
          <ChartTooltip
            cursor={{ strokeDasharray: "3 3" }}
            content={
              <ChartTooltipContent
                indicator="dot"
                labelFormatter={(value) => formatDate(value?.toString() ?? "")}
                valueFormatter={(payload) =>
                  Number(payload.value ?? 0).toLocaleString()
                }
              />
            }
          />
          <Area
            type="monotone"
            dataKey="requests"
            stroke="hsl(var(--chart-1))"
            fill="url(#requests-gradient)"
            strokeWidth={2}
          />
        </AreaChart>
      </ResponsiveContainer>
    </ChartContainer>
  );
}
