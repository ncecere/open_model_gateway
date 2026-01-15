import { Area, AreaChart, ResponsiveContainer } from "recharts";

export interface SparklineData {
  value: number;
}

interface SparklineProps {
  data: SparklineData[] | number[];
  color?: string;
  height?: number;
  width?: number | string;
  fillOpacity?: number;
  strokeWidth?: number;
  className?: string;
}

export function Sparkline({
  data,
  color = "hsl(var(--chart-1))",
  height = 32,
  width = "100%",
  fillOpacity = 0.2,
  strokeWidth = 1.5,
  className,
}: SparklineProps) {
  // Normalize data to { value: number }[]
  const normalizedData = data.map((item) =>
    typeof item === "number" ? { value: item } : item
  );

  if (!normalizedData.length) {
    return null;
  }

  return (
    <div className={className} style={{ width, height }}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart
          data={normalizedData}
          margin={{ top: 2, right: 0, bottom: 2, left: 0 }}
        >
          <defs>
            <linearGradient id={`sparkline-gradient-${color.replace(/[^a-z0-9]/gi, "")}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={color} stopOpacity={fillOpacity} />
              <stop offset="95%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <Area
            type="monotone"
            dataKey="value"
            stroke={color}
            strokeWidth={strokeWidth}
            fill={`url(#sparkline-gradient-${color.replace(/[^a-z0-9]/gi, "")})`}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

// Utility to extract sparkline data from usage points
export function extractSparklineData(
  points: Array<{ requests?: number; tokens?: number; cost_usd?: number; cost_cents?: number }>,
  metric: "requests" | "tokens" | "spend"
): SparklineData[] {
  return points.map((point) => {
    if (metric === "requests") {
      return { value: point.requests ?? 0 };
    }
    if (metric === "tokens") {
      return { value: point.tokens ?? 0 };
    }
    // spend
    return { value: point.cost_usd ?? (point.cost_cents ? point.cost_cents / 100 : 0) };
  });
}
