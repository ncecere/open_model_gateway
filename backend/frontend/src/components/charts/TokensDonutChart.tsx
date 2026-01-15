import { useMemo } from "react";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";

import { Skeleton } from "@/components/ui/skeleton";
import { formatTokensShort } from "@/lib/numbers";

export interface TokenBreakdownItem {
  id: string;
  label: string;
  tokens: number;
  requests?: number;
  cost_usd?: number;
}

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

interface TokensDonutChartProps {
  data: TokenBreakdownItem[];
  loading?: boolean;
  height?: number;
  showLegend?: boolean;
}

export function TokensDonutChart({
  data,
  loading,
  height = 280,
  showLegend = true,
}: TokensDonutChartProps) {
  if (loading) {
    return <Skeleton className="w-full" style={{ height }} />;
  }

  const chartData = useMemo(() => {
    const sorted = [...data].sort((a, b) => b.tokens - a.tokens);
    const top7 = sorted.slice(0, 7);
    const others = sorted.slice(7);
    const otherTokens = others.reduce((sum, item) => sum + item.tokens, 0);

    const result = top7.map((item, index) => ({
      ...item,
      color: COLORS[index % COLORS.length],
    }));

    if (otherTokens > 0) {
      result.push({
        id: "other",
        label: "Other",
        tokens: otherTokens,
        color: COLORS[7],
      });
    }

    return result;
  }, [data]);

  const totalTokens = useMemo(
    () => chartData.reduce((sum, item) => sum + item.tokens, 0),
    [chartData]
  );

  if (!chartData.length || totalTokens === 0) {
    return (
      <div
        className="flex items-center justify-center text-sm text-muted-foreground"
        style={{ height }}
      >
        No token usage data available.
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4" style={{ height }}>
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
              dataKey="tokens"
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
                const item = payload[0].payload as TokenBreakdownItem & { color: string };
                const percentage = ((item.tokens / totalTokens) * 100).toFixed(1);
                return (
                  <div className="rounded-lg border bg-popover px-3 py-2 text-xs shadow-sm">
                    <div className="flex items-center gap-2 mb-1">
                      <span
                        className="h-2 w-2 rounded-full"
                        style={{ backgroundColor: item.color }}
                      />
                      <span className="font-medium">{item.label}</span>
                    </div>
                    <div className="text-muted-foreground">
                      <div>{formatTokensShort(item.tokens)} tokens ({percentage}%)</div>
                      {item.requests != null && (
                        <div>{item.requests.toLocaleString()} requests</div>
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
              {formatTokensShort(totalTokens)}
            </text>
            <text
              x="50%"
              y="56%"
              textAnchor="middle"
              dominantBaseline="middle"
              className="fill-muted-foreground text-xs"
            >
              total tokens
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
