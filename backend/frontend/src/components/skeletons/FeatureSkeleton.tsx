import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export type FeatureSkeletonVariant = "card" | "list" | "form" | "stats";

export type FeatureSkeletonProps = {
  /** Type of feature skeleton to render */
  variant?: FeatureSkeletonVariant;
  /** Number of items for list/stats variants */
  items?: number;
  /** Additional class name */
  className?: string;
};

/**
 * FeatureSkeleton displays loading skeletons for different feature types.
 * Use within FeatureErrorBoundary or Suspense for feature-level loading states.
 */
export function FeatureSkeleton({
  variant = "card",
  items = 3,
  className,
}: FeatureSkeletonProps) {
  switch (variant) {
    case "card":
      return <CardSkeleton className={className} />;
    case "list":
      return <ListSkeleton items={items} className={className} />;
    case "form":
      return <FormSkeleton className={className} />;
    case "stats":
      return <StatsSkeleton items={items} className={className} />;
    default:
      return <CardSkeleton className={className} />;
  }
}

function CardSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn("rounded-lg border bg-card p-6 space-y-4", className)}>
      <div className="flex items-center justify-between">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-8 w-8 rounded-full" />
      </div>
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-3/4" />
      <div className="flex gap-2 pt-2">
        <Skeleton className="h-9 w-20" />
        <Skeleton className="h-9 w-20" />
      </div>
    </div>
  );
}

function ListSkeleton({ items, className }: { items: number; className?: string }) {
  return (
    <div className={cn("space-y-2", className)}>
      {Array.from({ length: items }).map((_, idx) => (
        <div key={idx} className="flex items-center gap-4 p-3 rounded-lg border bg-card">
          <Skeleton className="h-10 w-10 rounded-full" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-32" />
            <Skeleton className="h-3 w-48" />
          </div>
          <Skeleton className="h-8 w-8" />
        </div>
      ))}
    </div>
  );
}

function FormSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn("space-y-6", className)}>
      <div className="space-y-2">
        <Skeleton className="h-4 w-20" />
        <Skeleton className="h-10 w-full" />
      </div>
      <div className="space-y-2">
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-10 w-full" />
      </div>
      <div className="space-y-2">
        <Skeleton className="h-4 w-28" />
        <Skeleton className="h-24 w-full" />
      </div>
      <div className="flex justify-end gap-2">
        <Skeleton className="h-10 w-20" />
        <Skeleton className="h-10 w-24" />
      </div>
    </div>
  );
}

function StatsSkeleton({ items, className }: { items: number; className?: string }) {
  return (
    <div className={cn("grid gap-4 md:grid-cols-3", className)}>
      {Array.from({ length: items }).map((_, idx) => (
        <div key={idx} className="rounded-lg border bg-card p-4">
          <Skeleton className="h-4 w-20 mb-2" />
          <Skeleton className="h-8 w-16 mb-1" />
          <Skeleton className="h-3 w-24" />
        </div>
      ))}
    </div>
  );
}
