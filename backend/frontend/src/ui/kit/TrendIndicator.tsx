import { TrendingUp, TrendingDown, Minus } from "lucide-react";
import { cn } from "@/lib/utils";

export type TrendDirection = "up" | "down" | "flat";

export interface TrendIndicatorProps {
  /** Percentage change (e.g., 12.5 for +12.5%, -8.3 for -8.3%) */
  change: number;
  /** Optional threshold below which to show as "flat" (default: 0.5%) */
  flatThreshold?: number;
  /** Whether "up" is good (green) or bad (red). Default: false (up = bad for costs) */
  upIsGood?: boolean;
  /** Show the percentage value. Default: true */
  showValue?: boolean;
  /** Additional label text */
  label?: string;
  /** Size variant */
  size?: "sm" | "md" | "lg";
  className?: string;
}

export function TrendIndicator({
  change,
  flatThreshold = 0.5,
  upIsGood = false,
  showValue = true,
  label,
  size = "sm",
  className,
}: TrendIndicatorProps) {
  const direction = getDirection(change, flatThreshold);
  const colorClass = getColorClass(direction, upIsGood);
  const Icon = getIcon(direction);

  const sizeClasses = {
    sm: "text-xs gap-0.5",
    md: "text-sm gap-1",
    lg: "text-base gap-1",
  };

  const iconSizes = {
    sm: "h-3 w-3",
    md: "h-4 w-4",
    lg: "h-5 w-5",
  };

  const formattedValue = formatChange(change);

  return (
    <span
      className={cn(
        "inline-flex items-center font-medium",
        sizeClasses[size],
        colorClass,
        className
      )}
      title={label}
    >
      <Icon className={iconSizes[size]} />
      {showValue && <span>{formattedValue}</span>}
      {label && <span className="text-muted-foreground ml-1">{label}</span>}
    </span>
  );
}

function getDirection(change: number, threshold: number): TrendDirection {
  if (Math.abs(change) < threshold) return "flat";
  return change > 0 ? "up" : "down";
}

function getColorClass(direction: TrendDirection, upIsGood: boolean): string {
  if (direction === "flat") return "text-muted-foreground";

  const isPositive = direction === "up";
  const isGood = upIsGood ? isPositive : !isPositive;

  return isGood ? "text-success" : "text-destructive";
}

function getIcon(direction: TrendDirection) {
  switch (direction) {
    case "up":
      return TrendingUp;
    case "down":
      return TrendingDown;
    default:
      return Minus;
  }
}

function formatChange(change: number): string {
  const sign = change > 0 ? "+" : "";
  return `${sign}${change.toFixed(1)}%`;
}

// Utility function to calculate percentage change between two values
export function calculateTrendChange(current: number, previous: number): number {
  if (previous === 0) {
    if (current === 0) return 0;
    return current > 0 ? 100 : -100;
  }
  return ((current - previous) / previous) * 100;
}

// Calculate trend from an array of data points (compare first half vs second half)
export function calculateTrendFromPoints(
  points: Array<{ value: number }> | number[]
): number {
  if (points.length < 2) return 0;

  const values = points.map((p) => (typeof p === "number" ? p : p.value));
  const midpoint = Math.floor(values.length / 2);

  const firstHalf = values.slice(0, midpoint);
  const secondHalf = values.slice(midpoint);

  const firstAvg = firstHalf.reduce((a, b) => a + b, 0) / firstHalf.length;
  const secondAvg = secondHalf.reduce((a, b) => a + b, 0) / secondHalf.length;

  return calculateTrendChange(secondAvg, firstAvg);
}
