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
import { formatUsageDate } from "@/lib/dates";

export interface UsageBreakdownDatum {
  label: string;
  requests: number;
  tokens: number;
  spend: number;
}
const chartConfig = {
  requests: { label: "Requests", color: "hsl(var(--chart-1))" },
  tokens: { label: "Tokens", color: "hsl(var(--chart-2))" },
  spend: { label: "Spend", color: "hsl(var(--chart-3))" },
} satisfies ChartConfig;

type MetricKey = keyof typeof chartConfig;

export function UsageBreakdownChart({
	data,
	metric,
	timezone = "UTC",
}: {
	data: UsageBreakdownDatum[];
	metric: MetricKey;
	timezone?: string;
}) {
  if (!data.length) {
    return (
      <div className="flex h-64 items-center justify-center text-sm text-muted-foreground">
        No usage recorded for the selected window.
      </div>
    );
  }

  const metricConfig = chartConfig[metric];
  const maxValue = Math.max(...data.map((item) => item[metric] as number), 1);
	const formatDate = (value: string) => formatUsageDate(value, timezone);

  return (
    <ChartContainer config={chartConfig} className="h-80 w-full">
      <ResponsiveContainer>
        <AreaChart data={data} margin={{ left: 12, right: 12, top: 16, bottom: 8 }}>
          <defs>
            <linearGradient id={`usage-gradient-${metric}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={metricConfig.color} stopOpacity={0.4} />
              <stop offset="95%" stopColor={metricConfig.color} stopOpacity={0.05} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
          <XAxis
            dataKey="label"
            axisLine={false}
            tickLine={false}
            interval={Math.max(0, Math.floor(data.length / 7) - 1)}
            tickFormatter={(value) => formatDate(value as string)}
          />
          <YAxis
            axisLine={false}
            tickLine={false}
            tickFormatter={(value) => value.toLocaleString()}
            domain={[0, maxValue]}
          />
          <ChartTooltip
            cursor={{ strokeDasharray: "3 3" }}
            content={
              <ChartTooltipContent
                indicator="dot"
                labelFormatter={(value) => formatDate(value?.toString() ?? "")}
                valueFormatter={(payload) =>
                  metric === "spend"
                    ? `$${Number(payload.value ?? 0).toFixed(2)}`
                    : Number(payload.value ?? 0).toLocaleString()
                }
              />
            }
          />
          <Area
            type="monotone"
            dataKey={metric}
            stroke={metricConfig.color}
            fill={`url(#usage-gradient-${metric})`}
            strokeWidth={2}
          />
        </AreaChart>
      </ResponsiveContainer>
    </ChartContainer>
  );
}
