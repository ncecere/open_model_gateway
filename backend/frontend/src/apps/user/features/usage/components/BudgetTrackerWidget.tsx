import { useMemo } from "react";
import { AlertCircle, Clock, TrendingUp } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { formatUsageUSD } from "@/features/usage";
import { cn } from "@/lib/utils";

export interface BudgetTrackerWidgetProps {
  budgetLimitUsd: number | null | undefined;
  budgetUsedUsd: number;
  warningThreshold?: number;
  refreshSchedule?: string;
  dailyAverageSpend?: number;
  loading?: boolean;
  className?: string;
}

function formatScheduleLabel(schedule?: string): string {
  switch (schedule) {
    case "daily":
      return "Resets daily";
    case "weekly":
      return "Resets weekly";
    case "calendar_month":
      return "Resets monthly";
    case "calendar_quarter":
      return "Resets quarterly";
    case "calendar_year":
      return "Resets yearly";
    default:
      return "No reset schedule";
  }
}

function getDaysUntilReset(schedule?: string): number | null {
  if (!schedule) return null;

  const now = new Date();
  let resetDate: Date;

  switch (schedule) {
    case "daily":
      resetDate = new Date(now);
      resetDate.setDate(resetDate.getDate() + 1);
      resetDate.setHours(0, 0, 0, 0);
      break;
    case "weekly":
      resetDate = new Date(now);
      resetDate.setDate(resetDate.getDate() + (7 - resetDate.getDay()));
      resetDate.setHours(0, 0, 0, 0);
      break;
    case "calendar_month":
      resetDate = new Date(now.getFullYear(), now.getMonth() + 1, 1);
      break;
    case "calendar_quarter":
      const currentQuarter = Math.floor(now.getMonth() / 3);
      resetDate = new Date(now.getFullYear(), (currentQuarter + 1) * 3, 1);
      break;
    case "calendar_year":
      resetDate = new Date(now.getFullYear() + 1, 0, 1);
      break;
    default:
      return null;
  }

  const diffTime = resetDate.getTime() - now.getTime();
  return Math.ceil(diffTime / (1000 * 60 * 60 * 24));
}

export function BudgetTrackerWidget({
  budgetLimitUsd,
  budgetUsedUsd,
  warningThreshold = 0.8,
  refreshSchedule,
  dailyAverageSpend,
  loading,
  className,
}: BudgetTrackerWidgetProps) {
  if (loading) {
    return (
      <Card className={className}>
        <CardHeader>
          <Skeleton className="h-5 w-32" />
        </CardHeader>
        <CardContent className="space-y-4">
          <Skeleton className="h-2 w-full" />
          <Skeleton className="h-4 w-48" />
        </CardContent>
      </Card>
    );
  }

  const hasLimit = typeof budgetLimitUsd === "number" && budgetLimitUsd > 0;
  const usagePercent = hasLimit
    ? Math.min((budgetUsedUsd / budgetLimitUsd!) * 100, 100)
    : 0;
  const remaining = hasLimit ? budgetLimitUsd! - budgetUsedUsd : null;
  const isOverWarning = usagePercent >= warningThreshold * 100;
  const isOverBudget = usagePercent >= 100;
  const daysUntilReset = getDaysUntilReset(refreshSchedule);

  // Estimate days remaining at current rate
  const daysRemaining = useMemo(() => {
    if (!hasLimit || !dailyAverageSpend || dailyAverageSpend <= 0) return null;
    if (remaining === null || remaining <= 0) return 0;
    return Math.floor(remaining / dailyAverageSpend);
  }, [hasLimit, dailyAverageSpend, remaining]);

  const progressColor = isOverBudget
    ? "bg-destructive"
    : isOverWarning
      ? "bg-warning"
      : "bg-success";

  return (
    <Card className={cn("animate-fade-in-up", className)}>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center justify-between text-sm font-medium">
          <span>Budget Tracker</span>
          {isOverWarning && !isOverBudget && (
            <span className="flex items-center gap-1 text-warning text-xs">
              <AlertCircle className="h-3 w-3" />
              Warning
            </span>
          )}
          {isOverBudget && (
            <span className="flex items-center gap-1 text-destructive text-xs">
              <AlertCircle className="h-3 w-3" />
              Over budget
            </span>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {hasLimit ? (
          <>
            {/* Progress bar */}
            <div className="space-y-2">
              <Progress value={usagePercent} className="h-2" indicatorClassName={progressColor} />
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium">
                  {formatUsageUSD(budgetUsedUsd)} used
                </span>
                <span className="text-muted-foreground">
                  of {formatUsageUSD(budgetLimitUsd)}
                </span>
              </div>
            </div>

            {/* Remaining */}
            {remaining !== null && remaining > 0 && (
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Remaining</span>
                <span className={cn("font-medium", isOverWarning && "text-warning")}>
                  {formatUsageUSD(remaining)}
                </span>
              </div>
            )}

            {/* Days remaining estimate */}
            {daysRemaining !== null && (
              <div className="flex items-center gap-2 text-sm">
                <TrendingUp className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">
                  {daysRemaining > 0 ? (
                    <>
                      ~{daysRemaining} day{daysRemaining !== 1 ? "s" : ""} remaining at
                      current rate
                    </>
                  ) : (
                    "Budget depleted at current rate"
                  )}
                </span>
              </div>
            )}

            {/* Reset countdown */}
            {daysUntilReset !== null && (
              <div className="flex items-center gap-2 text-sm border-t pt-3">
                <Clock className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">
                  {formatScheduleLabel(refreshSchedule)} ·{" "}
                  {daysUntilReset === 0
                    ? "today"
                    : daysUntilReset === 1
                      ? "tomorrow"
                      : `in ${daysUntilReset} days`}
                </span>
              </div>
            )}
          </>
        ) : (
          <p className="text-sm text-muted-foreground">
            No budget limit configured for this scope.
          </p>
        )}
      </CardContent>
    </Card>
  );
}
