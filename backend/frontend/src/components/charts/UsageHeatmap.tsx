import { useMemo } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export interface HeatmapDataPoint {
  date: string;
  value: number;
}

interface UsageHeatmapProps {
  data: HeatmapDataPoint[];
  loading?: boolean;
  className?: string;
  valueLabel?: string;
  formatValue?: (value: number) => string;
}

const DAYS_OF_WEEK = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

function getIntensityClass(value: number, max: number): string {
  if (value === 0) return "bg-muted/40";
  const ratio = value / max;
  if (ratio < 0.25) return "bg-chart-1/20";
  if (ratio < 0.5) return "bg-chart-1/40";
  if (ratio < 0.75) return "bg-chart-1/60";
  return "bg-chart-1/80";
}

function formatDateShort(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

export function UsageHeatmap({
  data,
  loading,
  className,
  valueLabel = "Value",
  formatValue = (v) => v.toLocaleString(),
}: UsageHeatmapProps) {
  if (loading) {
    return <Skeleton className={cn("w-full h-32", className)} />;
  }

  const { grid, maxValue, weeks } = useMemo(() => {
    if (!data.length) {
      return { grid: [], maxValue: 0, weeks: [] };
    }

    // Sort data by date
    const sorted = [...data].sort(
      (a, b) => new Date(a.date).getTime() - new Date(b.date).getTime()
    );

    // Find the range
    const startDate = new Date(sorted[0].date);
    const endDate = new Date(sorted[sorted.length - 1].date);

    // Adjust start to beginning of week (Sunday)
    const adjustedStart = new Date(startDate);
    adjustedStart.setDate(adjustedStart.getDate() - adjustedStart.getDay());

    // Create a map for quick lookup
    const valueMap = new Map(data.map((d) => [d.date, d.value]));

    // Build grid
    const gridData: Array<{
      date: string;
      value: number;
      dayOfWeek: number;
      weekIndex: number;
    }> = [];

    let currentDate = new Date(adjustedStart);
    let weekIndex = 0;
    let max = 0;

    while (currentDate <= endDate) {
      const dateStr = currentDate.toISOString().split("T")[0];
      const value = valueMap.get(dateStr) ?? 0;
      max = Math.max(max, value);

      gridData.push({
        date: dateStr,
        value,
        dayOfWeek: currentDate.getDay(),
        weekIndex,
      });

      currentDate.setDate(currentDate.getDate() + 1);
      if (currentDate.getDay() === 0) {
        weekIndex++;
      }
    }

    // Generate week labels
    const weekLabels: string[] = [];
    let lastMonth = "";
    for (let i = 0; i <= weekIndex; i++) {
      const weekStart = new Date(adjustedStart);
      weekStart.setDate(weekStart.getDate() + i * 7);
      const month = weekStart.toLocaleDateString(undefined, { month: "short" });
      if (month !== lastMonth) {
        weekLabels.push(month);
        lastMonth = month;
      } else {
        weekLabels.push("");
      }
    }

    return { grid: gridData, maxValue: max || 1, weeks: weekLabels };
  }, [data]);

  if (!grid.length) {
    return (
      <div
        className={cn(
          "flex items-center justify-center h-32 text-sm text-muted-foreground",
          className
        )}
      >
        No activity data available.
      </div>
    );
  }

  // Group by week
  const weekCount = Math.max(...grid.map((g) => g.weekIndex)) + 1;

  return (
    <div className={cn("overflow-x-auto", className)}>
      <div className="inline-flex flex-col gap-1 min-w-fit">
        {/* Month labels */}
        <div className="flex gap-0.5 ml-8 mb-1">
          {weeks.map((label, i) => (
            <div
              key={i}
              className="w-3 text-[10px] text-muted-foreground"
              style={{ minWidth: "12px" }}
            >
              {label}
            </div>
          ))}
        </div>

        {/* Grid */}
        <div className="flex gap-0.5">
          {/* Day labels */}
          <div className="flex flex-col gap-0.5 mr-1">
            {DAYS_OF_WEEK.map((day, i) => (
              <div
                key={day}
                className={cn(
                  "h-3 text-[10px] text-muted-foreground leading-3 text-right pr-1",
                  i % 2 === 1 && "opacity-0"
                )}
              >
                {day}
              </div>
            ))}
          </div>

          {/* Cells */}
          {Array.from({ length: weekCount }).map((_, weekIdx) => (
            <div key={weekIdx} className="flex flex-col gap-0.5">
              {DAYS_OF_WEEK.map((_, dayIdx) => {
                const cell = grid.find(
                  (g) => g.weekIndex === weekIdx && g.dayOfWeek === dayIdx
                );

                if (!cell) {
                  return (
                    <div
                      key={dayIdx}
                      className="w-3 h-3 rounded-sm bg-transparent"
                    />
                  );
                }

                return (
                  <div
                    key={dayIdx}
                    className={cn(
                      "w-3 h-3 rounded-sm cursor-default transition-colors",
                      getIntensityClass(cell.value, maxValue)
                    )}
                    title={`${formatDateShort(cell.date)}: ${valueLabel} ${formatValue(cell.value)}`}
                  />
                );
              })}
            </div>
          ))}
        </div>

        {/* Legend */}
        <div className="flex items-center gap-2 mt-2 text-[10px] text-muted-foreground">
          <span>Less</span>
          <div className="flex gap-0.5">
            <div className="w-3 h-3 rounded-sm bg-muted/40" />
            <div className="w-3 h-3 rounded-sm bg-chart-1/20" />
            <div className="w-3 h-3 rounded-sm bg-chart-1/40" />
            <div className="w-3 h-3 rounded-sm bg-chart-1/60" />
            <div className="w-3 h-3 rounded-sm bg-chart-1/80" />
          </div>
          <span>More</span>
        </div>
      </div>
    </div>
  );
}
