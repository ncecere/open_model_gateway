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

export interface SpendChartDatum {
  date: string;
  cost_usd?: number;
  cost_cents?: number;
}

const chartConfig = {
  spend: { label: "Spend", color: "hsl(var(--chart-3))" },
} satisfies ChartConfig;

interface SpendAreaChartProps {
  data: SpendChartDatum[];
  loading?: boolean;
  height?: number;
  showGrid?: boolean;
}

export function SpendAreaChart({
  data,
  loading,
  height = 280,
  showGrid = true,
}: SpendAreaChartProps) {
  if (loading) {
    return <Skeleton className="w-full" style={{ height }} />;
  }

  // Normalize data to always use cost_usd
  const normalizedData = data.map((item) => ({
    ...item,
    spend: item.cost_usd ?? (item.cost_cents ? item.cost_cents / 100 : 0),
  }));

  if (!normalizedData.length) {
    return (
      <div
        className="flex items-center justify-center text-sm text-muted-foreground"
        style={{ height }}
      >
        No spend data available for the selected period.
      </div>
    );
  }

  const formatDate = (value: string) => {
    const date = new Date(value);
    return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  };

  const formatCurrency = (value: number) => {
    if (value >= 1000) {
      return `$${(value / 1000).toFixed(1)}K`;
    }
    if (value >= 1) {
      return `$${value.toFixed(0)}`;
    }
    return `$${value.toFixed(2)}`;
  };

  const maxValue = Math.max(...normalizedData.map((item) => item.spend), 0.01);

  return (
    <ChartContainer config={chartConfig} className="w-full" style={{ height }}>
      <ResponsiveContainer>
        <AreaChart data={normalizedData} margin={{ left: 0, right: 12, top: 16, bottom: 8 }}>
          <defs>
            <linearGradient id="spend-gradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="hsl(var(--chart-3))" stopOpacity={0.4} />
              <stop offset="95%" stopColor="hsl(var(--chart-3))" stopOpacity={0.05} />
            </linearGradient>
          </defs>
          {showGrid && <CartesianGrid strokeDasharray="3 3" className="stroke-muted/50" />}
          <XAxis
            dataKey="date"
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 11 }}
            tickFormatter={formatDate}
            interval={Math.max(0, Math.floor(normalizedData.length / 7) - 1)}
          />
          <YAxis
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 11 }}
            tickFormatter={formatCurrency}
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
                  `$${Number(payload.value ?? 0).toFixed(4)}`
                }
              />
            }
          />
          <Area
            type="monotone"
            dataKey="spend"
            stroke="hsl(var(--chart-3))"
            fill="url(#spend-gradient)"
            strokeWidth={2}
          />
        </AreaChart>
      </ResponsiveContainer>
    </ChartContainer>
  );
}
