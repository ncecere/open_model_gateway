import { ArrowRight, TrendingDown, TrendingUp, Minus } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export interface ComparisonMetric {
  label: string;
  current: number;
  previous: number;
  format?: (value: number) => string;
  /** Whether higher values are good (true) or bad (false). Default: false for costs */
  higherIsGood?: boolean;
}

export interface MetricDifferenceCardProps {
  title: string;
  description?: string;
  currentLabel: string;
  previousLabel: string;
  metrics: ComparisonMetric[];
  loading?: boolean;
  className?: string;
}

function defaultFormat(value: number): string {
  return value.toLocaleString();
}

function calculateChange(current: number, previous: number): number {
  if (previous === 0) {
    if (current === 0) return 0;
    return current > 0 ? 100 : -100;
  }
  return ((current - previous) / previous) * 100;
}

function formatChange(change: number): string {
  const sign = change > 0 ? "+" : "";
  return `${sign}${change.toFixed(1)}%`;
}

export function MetricDifferenceCard({
  title,
  description,
  currentLabel,
  previousLabel,
  metrics,
  loading,
  className,
}: MetricDifferenceCardProps) {
  if (loading) {
    return (
      <Card className={className}>
        <CardHeader>
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-4 w-48" />
        </CardHeader>
        <CardContent className="space-y-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={cn("animate-fade-in-up", className)}>
      <CardHeader>
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {description && (
          <p className="text-xs text-muted-foreground">{description}</p>
        )}
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Header row */}
        <div className="grid grid-cols-4 gap-2 text-xs font-medium text-muted-foreground">
          <div>Metric</div>
          <div className="text-right">{previousLabel}</div>
          <div className="text-right">{currentLabel}</div>
          <div className="text-right">Change</div>
        </div>

        {/* Metric rows */}
        {metrics.map((metric, index) => {
          const format = metric.format ?? defaultFormat;
          const change = calculateChange(metric.current, metric.previous);
          const isImproved = metric.higherIsGood
            ? change > 0
            : change < 0;
          const isWorse = metric.higherIsGood
            ? change < 0
            : change > 0;
          const isFlat = Math.abs(change) < 0.5;

          return (
            <div
              key={index}
              className="grid grid-cols-4 gap-2 items-center py-2 border-t"
            >
              <div className="text-sm font-medium">{metric.label}</div>
              <div className="text-right text-sm text-muted-foreground">
                {format(metric.previous)}
              </div>
              <div className="text-right text-sm font-medium">
                {format(metric.current)}
              </div>
              <div
                className={cn(
                  "flex items-center justify-end gap-1 text-sm font-medium",
                  isFlat && "text-muted-foreground",
                  isImproved && "text-success",
                  isWorse && "text-destructive"
                )}
              >
                {isFlat ? (
                  <Minus className="h-3 w-3" />
                ) : change > 0 ? (
                  <TrendingUp className="h-3 w-3" />
                ) : (
                  <TrendingDown className="h-3 w-3" />
                )}
                {formatChange(change)}
              </div>
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}

// Compact inline comparison display
export interface InlineComparisonProps {
  current: number;
  previous: number;
  format?: (value: number) => string;
  higherIsGood?: boolean;
  showPrevious?: boolean;
  className?: string;
}

export function InlineComparison({
  current,
  previous,
  format = defaultFormat,
  higherIsGood = false,
  showPrevious = true,
  className,
}: InlineComparisonProps) {
  const change = calculateChange(current, previous);
  const isImproved = higherIsGood ? change > 0 : change < 0;
  const isWorse = higherIsGood ? change < 0 : change > 0;
  const isFlat = Math.abs(change) < 0.5;

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <span className="font-medium">{format(current)}</span>
      {showPrevious && (
        <>
          <ArrowRight className="h-3 w-3 text-muted-foreground" />
          <span className="text-muted-foreground">{format(previous)}</span>
        </>
      )}
      <span
        className={cn(
          "text-xs font-medium",
          isFlat && "text-muted-foreground",
          isImproved && "text-success",
          isWorse && "text-destructive"
        )}
      >
        {formatChange(change)}
      </span>
    </div>
  );
}
