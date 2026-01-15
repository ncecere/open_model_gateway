import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Sparkline, type SparklineData } from "@/components/charts/Sparkline";
import {
  TrendIndicator,
  calculateTrendFromPoints,
} from "@/ui/kit/TrendIndicator";
import { cn } from "@/lib/utils";

export interface EnhancedSummaryCardProps {
  /** Card title (displayed in uppercase) */
  title: string;
  /** Primary value display */
  value: ReactNode;
  /** Secondary description text */
  description?: ReactNode;
  /** Data points for sparkline visualization */
  sparklineData?: SparklineData[] | number[];
  /** Pre-calculated trend percentage. If not provided, will calculate from sparklineData */
  trendChange?: number;
  /** Whether "up" trends should be colored green (default: false for costs) */
  upIsGood?: boolean;
  /** Previous period value for comparison display */
  previousValue?: ReactNode;
  /** Loading state */
  loading?: boolean;
  /** Optional icon */
  icon?: LucideIcon;
  /** Sparkline color (CSS variable or hex) */
  sparklineColor?: string;
  /** Additional CSS classes */
  className?: string;
}

export function EnhancedSummaryCard({
  title,
  value,
  description,
  sparklineData,
  trendChange,
  upIsGood = false,
  previousValue,
  loading,
  icon: Icon,
  sparklineColor = "hsl(var(--chart-1))",
  className,
}: EnhancedSummaryCardProps) {
  // Calculate trend from sparkline data if not provided explicitly
  const calculatedTrend =
    trendChange ?? (sparklineData ? calculateTrendFromPoints(sparklineData) : 0);

  const hasSparkline = sparklineData && sparklineData.length > 1;
  const hasTrend = calculatedTrend !== 0 || trendChange !== undefined;

  return (
    <Card className={cn("animate-fade-in-up overflow-hidden", className)}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          {title}
        </CardTitle>
        {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
      </CardHeader>
      <CardContent className="space-y-2">
        {loading ? (
          <>
            <Skeleton className="h-8 w-28" />
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-8 w-full mt-2" />
          </>
        ) : (
          <>
            <div className="flex items-baseline gap-3">
              <p className="text-3xl font-semibold tracking-tight">{value}</p>
              {hasTrend && (
                <TrendIndicator
                  change={calculatedTrend}
                  upIsGood={upIsGood}
                  size="md"
                />
              )}
            </div>

            {previousValue && (
              <p className="text-sm text-muted-foreground">
                vs {previousValue} last period
              </p>
            )}

            {description && !previousValue && (
              <p className="text-sm text-muted-foreground">{description}</p>
            )}

            {hasSparkline && (
              <div className="pt-2">
                <Sparkline
                  data={sparklineData}
                  color={sparklineColor}
                  height={36}
                  fillOpacity={0.15}
                  strokeWidth={1.5}
                />
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}

// Variant for displaying key metrics in a more compact format
export interface CompactMetricCardProps {
  title: string;
  value: ReactNode;
  trend?: {
    change: number;
    upIsGood?: boolean;
  };
  loading?: boolean;
  className?: string;
}

export function CompactMetricCard({
  title,
  value,
  trend,
  loading,
  className,
}: CompactMetricCardProps) {
  return (
    <div className={cn("space-y-1", className)}>
      <p className="text-xs font-medium text-muted-foreground">{title}</p>
      {loading ? (
        <Skeleton className="h-6 w-16" />
      ) : (
        <div className="flex items-baseline gap-2">
          <p className="text-xl font-semibold">{value}</p>
          {trend && (
            <TrendIndicator
              change={trend.change}
              upIsGood={trend.upIsGood}
              size="sm"
            />
          )}
        </div>
      )}
    </div>
  );
}
