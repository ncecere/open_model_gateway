import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export type PageSkeletonProps = {
  /** Show header section with title and actions placeholder */
  showHeader?: boolean;
  /** Number of stat cards to show (0 to hide) */
  statCards?: number;
  /** Show main content area skeleton */
  showContent?: boolean;
  /** Height of main content area */
  contentHeight?: string;
  /** Additional class name */
  className?: string;
};

/**
 * PageSkeleton displays a loading skeleton for full page content.
 * Matches the typical page layout: header, stat cards, and main content.
 */
export function PageSkeleton({
  showHeader = true,
  statCards = 3,
  showContent = true,
  contentHeight = "h-96",
  className,
}: PageSkeletonProps) {
  return (
    <div className={cn("space-y-6", className)}>
      {showHeader && (
        <div className="flex items-center justify-between">
          <div className="space-y-2">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-4 w-64" />
          </div>
          <div className="flex gap-2">
            <Skeleton className="h-9 w-24" />
            <Skeleton className="h-9 w-32" />
          </div>
        </div>
      )}

      {statCards > 0 && (
        <div className="grid gap-4 md:grid-cols-3">
          {Array.from({ length: statCards }).map((_, idx) => (
            <div key={idx} className="rounded-lg border bg-card p-6">
              <Skeleton className="h-4 w-24 mb-2" />
              <Skeleton className="h-8 w-16 mb-1" />
              <Skeleton className="h-3 w-32" />
            </div>
          ))}
        </div>
      )}

      {showContent && (
        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b">
            <div className="flex items-center justify-between">
              <Skeleton className="h-6 w-32" />
              <div className="flex gap-2">
                <Skeleton className="h-9 w-48" />
                <Skeleton className="h-9 w-24" />
              </div>
            </div>
          </div>
          <div className="p-4">
            <Skeleton className={cn("w-full", contentHeight)} />
          </div>
        </div>
      )}
    </div>
  );
}
