import type { ReactNode } from "react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

interface ChartCardProps {
  title: string;
  description?: string;
  toolbar?: ReactNode;
  isLoading?: boolean;
  loadingHeight?: number;
  className?: string;
  children: ReactNode;
}

export function ChartCard({
  title,
  description,
  toolbar,
  isLoading,
  loadingHeight = 256,
  className,
  children,
}: ChartCardProps) {
  return (
    <Card className={className}>
      <CardHeader className="space-y-4">
        <div className="space-y-1">
          <CardTitle>{title}</CardTitle>
          {description ? (
            <p className="text-sm text-muted-foreground">{description}</p>
          ) : null}
        </div>
        {toolbar}
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading ? (
          <Skeleton
            data-testid="chartcard-skeleton"
            className="w-full"
            style={{ height: loadingHeight }}
          />
        ) : (
          children
        )}
      </CardContent>
    </Card>
  );
}
