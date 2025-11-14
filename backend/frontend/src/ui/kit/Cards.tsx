import type { ReactNode } from "react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

interface MetricCardProps {
  title: string;
  value: ReactNode;
  secondary?: ReactNode;
  loading?: boolean;
}

export function MetricCard({ title, value, secondary, loading }: MetricCardProps) {
  return (
    <Card>
      <CardHeader className="space-y-1">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-1">
        {loading ? (
          <Skeleton className="h-8 w-24" />
        ) : (
          <p className="text-2xl font-semibold tracking-tight">{value}</p>
        )}
        {secondary ? (
          <p className="text-sm text-muted-foreground">{secondary}</p>
        ) : null}
      </CardContent>
    </Card>
  );
}

interface SummaryCardProps {
  title: string;
  value: ReactNode;
  description?: ReactNode;
  loading?: boolean;
}

export function SummaryCard({ title, value, description, loading }: SummaryCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <Skeleton className="h-10 w-32" />
        ) : (
          <p className="text-3xl font-semibold tracking-tight">{value}</p>
        )}
        {description ? (
          <p className="mt-2 text-sm text-muted-foreground">{description}</p>
        ) : null}
      </CardContent>
    </Card>
  );
}
