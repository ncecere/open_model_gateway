import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export type TableSkeletonProps = {
  /** Number of rows to display */
  rows?: number;
  /** Height of each row (defaults to h-12) */
  rowHeight?: string;
  /** Additional class name */
  className?: string;
};

/**
 * TableSkeleton displays a loading skeleton for table content.
 */
export function TableSkeleton({
  rows = 4,
  rowHeight = "h-12",
  className,
}: TableSkeletonProps) {
  return (
    <div className={cn("space-y-2", className)}>
      {Array.from({ length: rows }).map((_, idx) => (
        <Skeleton key={idx} className={cn("w-full", rowHeight)} />
      ))}
    </div>
  );
}
