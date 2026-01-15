import { useMemo } from "react";
import { Brain, DollarSign, TrendingDown, TrendingUp, Zap } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatUsageUSD, formatTokensShort } from "@/features/usage";
import { cn } from "@/lib/utils";

export interface SpendingInsight {
  type: "comparison" | "model" | "trend" | "tip";
  icon: "trending-up" | "trending-down" | "dollar" | "brain" | "zap";
  title: string;
  description: string;
  isPositive?: boolean;
}

export interface SpendingInsightsProps {
  currentSpend: number;
  previousSpend?: number;
  topModel?: { alias: string; spend: number; tokens: number };
  totalTokens?: number;
  loading?: boolean;
  className?: string;
}

function getIconComponent(icon: SpendingInsight["icon"]) {
  switch (icon) {
    case "trending-up":
      return TrendingUp;
    case "trending-down":
      return TrendingDown;
    case "dollar":
      return DollarSign;
    case "brain":
      return Brain;
    case "zap":
      return Zap;
  }
}

export function SpendingInsights({
  currentSpend,
  previousSpend,
  topModel,
  totalTokens,
  loading,
  className,
}: SpendingInsightsProps) {
  const insights = useMemo(() => {
    const result: SpendingInsight[] = [];

    // Spending comparison insight
    if (previousSpend !== undefined && previousSpend > 0) {
      const change = ((currentSpend - previousSpend) / previousSpend) * 100;
      const isIncrease = change > 5;
      const isDecrease = change < -5;

      if (isIncrease) {
        result.push({
          type: "comparison",
          icon: "trending-up",
          title: `${Math.abs(change).toFixed(0)}% more than last period`,
          description: `You've spent ${formatUsageUSD(currentSpend - previousSpend)} more compared to the previous period.`,
          isPositive: false,
        });
      } else if (isDecrease) {
        result.push({
          type: "comparison",
          icon: "trending-down",
          title: `${Math.abs(change).toFixed(0)}% less than last period`,
          description: `You've saved ${formatUsageUSD(previousSpend - currentSpend)} compared to the previous period.`,
          isPositive: true,
        });
      }
    }

    // Top model insight
    if (topModel && topModel.spend > 0) {
      const modelPercent =
        currentSpend > 0 ? (topModel.spend / currentSpend) * 100 : 0;
      result.push({
        type: "model",
        icon: "brain",
        title: `Most used: ${topModel.alias}`,
        description: `${topModel.alias} accounts for ${modelPercent.toFixed(0)}% of your spend (${formatTokensShort(topModel.tokens)} tokens).`,
      });
    }

    // Cost efficiency tip
    if (totalTokens && totalTokens > 0 && currentSpend > 0) {
      const costPerMillion = (currentSpend / totalTokens) * 1_000_000;
      result.push({
        type: "tip",
        icon: "zap",
        title: `Average cost: ${formatUsageUSD(costPerMillion)}/M tokens`,
        description:
          "Consider using smaller models for simple tasks to reduce costs.",
      });
    }

    return result;
  }, [currentSpend, previousSpend, topModel, totalTokens]);

  if (loading) {
    return (
      <Card className={className}>
        <CardHeader>
          <Skeleton className="h-5 w-32" />
        </CardHeader>
        <CardContent className="space-y-4">
          {[1, 2].map((i) => (
            <div key={i} className="flex gap-3">
              <Skeleton className="h-8 w-8 rounded-full" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-4 w-48" />
                <Skeleton className="h-3 w-full" />
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    );
  }

  if (!insights.length) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Spending Insights</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Not enough data to generate insights. Keep using the API to see
            personalized recommendations.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={cn("animate-fade-in-up", className)}>
      <CardHeader>
        <CardTitle className="text-sm font-medium">Spending Insights</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {insights.map((insight, index) => {
          const Icon = getIconComponent(insight.icon);
          return (
            <div key={index} className="flex gap-3">
              <div
                className={cn(
                  "flex h-8 w-8 shrink-0 items-center justify-center rounded-full",
                  insight.isPositive === true && "bg-success/10 text-success",
                  insight.isPositive === false &&
                    "bg-destructive/10 text-destructive",
                  insight.isPositive === undefined &&
                    "bg-muted text-muted-foreground"
                )}
              >
                <Icon className="h-4 w-4" />
              </div>
              <div className="space-y-1">
                <p className="text-sm font-medium">{insight.title}</p>
                <p className="text-xs text-muted-foreground">
                  {insight.description}
                </p>
              </div>
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
