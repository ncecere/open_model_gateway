import type { ReactNode } from "react";
import { TrendingUp, TrendingDown, Minus, type LucideIcon } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface TrendIndicator {
  direction: "up" | "down" | "flat";
  value: string;
  label?: string;
}

interface MetricCardProps {
  title: string;
  value: ReactNode;
  secondary?: ReactNode;
  loading?: boolean;
  icon?: LucideIcon;
  trend?: TrendIndicator;
  status?: "success" | "warning" | "destructive" | "info";
  className?: string;
}

export function MetricCard({
  title,
  value,
  secondary,
  loading,
  icon: Icon,
  trend,
  status,
  className,
}: MetricCardProps) {
  return (
    <Card
      className={cn(
        "animate-fade-in-up",
        status && statusBorderClasses[status],
        className
      )}
    >
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          {title}
        </CardTitle>
        {Icon && (
          <Icon className="h-4 w-4 text-muted-foreground" />
        )}
      </CardHeader>
      <CardContent className="space-y-1">
        {loading ? (
          <>
            <Skeleton className="h-8 w-24" />
            <Skeleton className="h-4 w-16" />
          </>
        ) : (
          <>
            <div className="flex items-baseline gap-2">
              <p className="text-2xl font-semibold tracking-tight">{value}</p>
              {trend && <TrendBadge trend={trend} />}
            </div>
            {secondary && (
              <p className="text-sm text-muted-foreground">{secondary}</p>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}

const statusBorderClasses = {
  success: "border-l-2 border-l-success",
  warning: "border-l-2 border-l-warning",
  destructive: "border-l-2 border-l-destructive",
  info: "border-l-2 border-l-info",
};

function TrendBadge({ trend }: { trend: TrendIndicator }) {
  const { direction, value, label } = trend;

  const trendColors = {
    up: "text-success",
    down: "text-destructive",
    flat: "text-muted-foreground",
  };

  const TrendIcon = {
    up: TrendingUp,
    down: TrendingDown,
    flat: Minus,
  }[direction];

  return (
    <span
      className={cn(
        "inline-flex items-center gap-0.5 text-xs font-medium",
        trendColors[direction]
      )}
      title={label}
    >
      <TrendIcon className="h-3 w-3" />
      {value}
    </span>
  );
}

interface SummaryCardProps {
  title: string;
  value: ReactNode;
  description?: ReactNode;
  loading?: boolean;
  className?: string;
}

export function SummaryCard({
  title,
  value,
  description,
  loading,
  className,
}: SummaryCardProps) {
  return (
    <Card className={cn("animate-fade-in-up", className)}>
      <CardHeader>
        <CardTitle className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <Skeleton className="h-10 w-32" />
        ) : (
          <p className="text-3xl font-semibold tracking-tight">{value}</p>
        )}
        {description && (
          <p className="mt-2 text-sm text-muted-foreground">{description}</p>
        )}
      </CardContent>
    </Card>
  );
}

// Status Dot component for use in cards and tables
interface StatusDotProps {
  status: "online" | "offline" | "warning" | "unknown";
  pulse?: boolean;
  className?: string;
}

export function StatusDot({ status, pulse = false, className }: StatusDotProps) {
  const statusColors = {
    online: "bg-success",
    offline: "bg-destructive",
    warning: "bg-warning",
    unknown: "bg-muted-foreground",
  };

  return (
    <span
      className={cn(
        "relative inline-flex h-2 w-2 rounded-full",
        statusColors[status],
        className
      )}
    >
      {pulse && status === "online" && (
        <span
          className={cn(
            "absolute inline-flex h-full w-full animate-ping rounded-full opacity-75",
            statusColors[status]
          )}
        />
      )}
    </span>
  );
}
