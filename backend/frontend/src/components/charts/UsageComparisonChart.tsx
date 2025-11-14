import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  XAxis,
  YAxis,
} from "recharts";

import type { UsageComparisonSeries } from "@/api/usage";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { formatUsageDate } from "@/lib/dates";

const COLOR_PALETTE = [
  "hsl(var(--chart-1))",
  "hsl(var(--chart-2))",
  "hsl(var(--chart-3))",
  "hsl(var(--chart-4))",
  "hsl(var(--chart-5))",
  "hsl(var(--chart-6, 210 40% 65%))",
  "hsl(var(--chart-7, 120 45% 55%))",
  "hsl(var(--chart-8, 330 60% 60%))",
];

const DEFAULT_COLOR = "hsl(var(--muted-foreground))";
const PAD_DAYS = 2;
const MS_IN_DAY = 24 * 60 * 60 * 1000;

export type UsageComparisonMetric = "requests" | "tokens" | "spend";

const METRIC_CONFIG: Record<UsageComparisonMetric, { label: string }> = {
  requests: { label: "Requests" },
  tokens: { label: "Tokens" },
  spend: { label: "Spend" },
};

type DecoratedSeries = UsageComparisonSeries & {
  color: string;
  pointMap: Map<string, number>;
  chartKey: string;
};

function valueForMetric(
  metric: UsageComparisonMetric,
  point?: { requests: number; tokens: number; cost_cents: number; cost_usd?: number },
) {
  if (!point) {
    return 0;
  }
  switch (metric) {
    case "requests":
      return point.requests;
    case "tokens":
      return point.tokens;
    case "spend":
      if (typeof point.cost_usd === "number") {
        return point.cost_usd;
      }
      return point.cost_cents / 100;
    default:
      return 0;
  }
}

function valueForMetricTotals(
  metric: UsageComparisonMetric,
  totals: { requests: number; tokens: number; cost_cents: number; cost_usd?: number },
) {
  switch (metric) {
    case "requests":
      return totals.requests;
    case "tokens":
      return totals.tokens;
    case "spend":
      if (typeof totals.cost_usd === "number") {
        return totals.cost_usd;
      }
      return totals.cost_cents / 100;
    default:
      return 0;
  }
}

function formatMetricValue(metric: UsageComparisonMetric, value: number) {
  if (metric === "spend") {
    return `$${value.toFixed(2)}`;
  }
  return value.toLocaleString();
}

function decorateSeries(
  series: UsageComparisonSeries[],
  metric: UsageComparisonMetric,
): DecoratedSeries[] {
  return series.map((entry, index) => {
    const color = COLOR_PALETTE[index] ?? DEFAULT_COLOR;
    const pointMap = new Map<string, number>();
    entry.points.forEach((point) => {
      pointMap.set(point.date, valueForMetric(metric, point));
    });
    const chartKey = `series_${index}`;
    return { ...entry, color, pointMap, chartKey };
  });
}

type ActiveRange = { startMs: number; endMs: number };

function deriveActiveRange(series: UsageComparisonSeries[]): ActiveRange | null {
  let min = Number.POSITIVE_INFINITY;
  let max = Number.NEGATIVE_INFINITY;
  series.forEach((entry) => {
    if (entry.active_start) {
      const ts = Date.parse(entry.active_start);
      if (!Number.isNaN(ts)) {
        min = Math.min(min, ts);
      }
    }
    if (entry.active_end) {
      const ts = Date.parse(entry.active_end);
      if (!Number.isNaN(ts)) {
        max = Math.max(max, ts);
      }
    }
  });
  if (!Number.isFinite(min) || !Number.isFinite(max) || min === Number.POSITIVE_INFINITY || max === Number.NEGATIVE_INFINITY) {
    return null;
  }
  return { startMs: min, endMs: max };
}

type FilterResult = { rows: Record<string, number | string>[]; activeStartISO: string; activeEndISO: string };

function applyPaddedRangeFilter(
  rows: Record<string, number | string>[],
  range: ActiveRange | null,
  paddingDays: number,
): FilterResult | null {
  if (!range || rows.length === 0) {
    return null;
  }
  const earliestTs = Date.parse(rows[0].date as string);
  const latestTs = Date.parse(rows[rows.length - 1].date as string);
  if (Number.isNaN(earliestTs) || Number.isNaN(latestTs)) {
    return null;
  }
  const padding = paddingDays * MS_IN_DAY;
  const paddedStart = Math.max(range.startMs - padding, earliestTs);
  const paddedEnd = Math.min(range.endMs + padding, latestTs);
  if (paddedStart > paddedEnd) {
    return null;
  }
  const filtered = rows.filter((row) => {
    const ts = Date.parse(row.date as string);
    if (Number.isNaN(ts)) {
      return false;
    }
    return ts >= paddedStart && ts <= paddedEnd;
  });
  if (filtered.length === 0 || filtered.length === rows.length) {
    return null;
  }
  return {
    rows: filtered,
    activeStartISO: new Date(range.startMs).toISOString(),
    activeEndISO: new Date(range.endMs).toISOString(),
  };
}

function applyFallbackRangeFilter(
  rows: Record<string, number | string>[],
  decorated: DecoratedSeries[],
): FilterResult | null {
  if (rows.length === 0) {
    return null;
  }
  const firstActiveIndex = rows.findIndex((row) =>
    decorated.some((entry) => {
      const value = row[entry.chartKey];
      return typeof value === "number" && value > 0;
    }),
  );
  let lastActiveIndex = -1;
  for (let idx = rows.length - 1; idx >= 0; idx -= 1) {
    const row = rows[idx];
    const hasValue = decorated.some((entry) => {
      const value = row[entry.chartKey];
      return typeof value === "number" && value > 0;
    });
    if (hasValue) {
      lastActiveIndex = idx;
      break;
    }
  }
  if (firstActiveIndex <= 0 || lastActiveIndex < firstActiveIndex) {
    return null;
  }
  const startIndex = Math.max(firstActiveIndex - 2, 0);
  const endIndex = Math.min(lastActiveIndex + 2, rows.length - 1);
  const sliced = rows.slice(startIndex, endIndex + 1);
  if (sliced.length === 0 || sliced.length === rows.length) {
    return null;
  }
  const firstISO = rows[firstActiveIndex].date as string;
  const lastISO = rows[lastActiveIndex].date as string;
  return {
    rows: sliced,
    activeStartISO: new Date(firstISO).toISOString(),
    activeEndISO: new Date(lastISO).toISOString(),
  };
}

export function UsageComparisonChart({
  series,
  metric,
  timezone = "UTC",
  activeSeriesId,
}: {
  series: UsageComparisonSeries[];
  metric: UsageComparisonMetric;
  timezone?: string;
  activeSeriesId?: string;
}) {
  if (!series.length) {
    return (
      <div className="flex h-64 items-center justify-center text-sm text-muted-foreground">
        Select at least one entity to compare.
      </div>
    );
  }

  const decorated = decorateSeries(series, metric);
  const dates = new Set<string>();
  decorated.forEach((entry) => {
    entry.points.forEach((point) => {
      dates.add(point.date);
    });
  });
  const sortedDates = Array.from(dates).sort();
  const rows = sortedDates.map((date) => {
    const row: Record<string, number | string> = { date };
    decorated.forEach((entry) => {
      row[entry.chartKey] = entry.pointMap.get(date) ?? 0;
    });
    return row;
  });

  if (rows.length === 0 && decorated.length > 0) {
    const fallbackDate =
      decorated[0].points[decorated[0].points.length - 1]?.date ??
      decorated[0].points[0]?.date ??
      new Date().toISOString();
    rows.push({ date: fallbackDate });
    decorated.forEach((entry) => {
      rows[0][entry.chartKey] = 0;
    });
  }

  const formatDate = (value: string) => formatUsageDate(value, timezone);

  const chartConfig = decorated.reduce<Record<string, { label: string; color: string }>>(
    (acc, entry) => {
      acc[entry.chartKey] = { label: entry.label, color: entry.color };
      return acc;
    },
    {},
  );

  const hasPointActivity = rows.some((row) =>
    decorated.some((entry) => {
      const value = row[entry.chartKey];
      return typeof value === "number" && value > 0;
    }),
  );
  const hasTotalActivity = decorated.some(
    (entry) => valueForMetricTotals(metric, entry.totals) > 0,
  );
  const showEmptyState = !hasPointActivity && !hasTotalActivity;

  let visibleRows = rows;
  let visibleDates = sortedDates;
  let trimmedRange: { start: string; end: string } | null = null;
  const activeRange = deriveActiveRange(series);
  if (!showEmptyState && rows.length > 0) {
    const padded = activeRange
      ? applyPaddedRangeFilter(rows, activeRange, PAD_DAYS)
      : applyFallbackRangeFilter(rows, decorated);

    if (padded) {
      visibleRows = padded.rows;
      visibleDates = padded.rows.map((row) => row.date as string);
      trimmedRange = {
        start: padded.activeStartISO,
        end: padded.activeEndISO,
      };
    }
  }

  if (showEmptyState) {
    return (
      <div className="flex h-64 items-center justify-center rounded-md border border-dashed">
        <p className="text-sm text-muted-foreground">
          No usage recorded for the selected window.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-4">
        {decorated.map((entry) => (
          <div key={entry.id} className="flex items-center gap-2 text-sm">
            <span
              className="h-2.5 w-2.5 rounded-full"
              style={{ backgroundColor: entry.color }}
            />
            <div>
              <p className="font-medium leading-none">{entry.label}</p>
              <p className="text-xs text-muted-foreground">
                {METRIC_CONFIG[metric].label}: {formatMetricValue(metric, valueForMetricTotals(metric, entry.totals))}
              </p>
            </div>
          </div>
        ))}
      </div>
      <ChartContainer className="h-96 w-full" config={chartConfig}>
        <ResponsiveContainer>
          <LineChart data={visibleRows} margin={{ left: 12, right: 12, top: 20, bottom: 8 }}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
            <XAxis
              dataKey="date"
              axisLine={false}
              tickLine={false}
              tickFormatter={(value) => formatDate(value as string)}
              interval={Math.max(0, Math.floor(visibleDates.length / 7) - 1)}
            />
            <YAxis
              axisLine={false}
              tickLine={false}
              tickFormatter={(value) =>
                metric === "spend" ? `$${value.toFixed(2)}` : value.toLocaleString()
              }
            />
            <ChartTooltip
              content={
                <ChartTooltipContent
                  indicator="dot"
                  labelFormatter={(value) => formatDate(value?.toString() ?? "")}
                  valueFormatter={(payload) =>
                    formatMetricValue(metric, Number(payload.value ?? 0))
                  }
                />
              }
            />
            {decorated.map((entry) => (
              <Line
                key={entry.chartKey}
                type="monotone"
                dataKey={entry.chartKey}
                stroke={entry.color}
                strokeWidth={activeSeriesId && entry.id === activeSeriesId ? 3 : 2}
                strokeOpacity={activeSeriesId && entry.id !== activeSeriesId ? 0.35 : 1}
                dot={{ r: 2 }}
                activeDot={{ r: 4 }}
                isAnimationActive={false}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </ChartContainer>
      {trimmedRange ? (
        <p className="text-xs text-muted-foreground">
          Showing activity between {formatUsageDate(trimmedRange.start, timezone)} and{" "}
          {formatUsageDate(trimmedRange.end, timezone)}. No usage recorded outside this window.
        </p>
      ) : null}
    </div>
  );
}
