import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export type TablePaginationProps = {
  /** Current page (1-based) */
  page: number;
  /** Total number of pages */
  totalPages: number;
  /** Whether there's a previous page */
  hasPrev: boolean;
  /** Whether there's a next page */
  hasNext: boolean;
  /** Called when navigating to previous page */
  onPrev: () => void;
  /** Called when navigating to next page */
  onNext: () => void;
  /** Whether pagination is loading/disabled */
  isLoading?: boolean;
  /** Additional class name */
  className?: string;
};

/**
 * TablePagination provides prev/next navigation for paginated tables.
 */
export function TablePagination({
  page,
  totalPages,
  hasPrev,
  hasNext,
  onPrev,
  onNext,
  isLoading = false,
  className,
}: TablePaginationProps) {
  return (
    <div className={cn("flex items-center justify-between", className)}>
      <Button
        variant="outline"
        size="sm"
        disabled={!hasPrev || isLoading}
        onClick={onPrev}
      >
        Previous
      </Button>
      <p className="text-sm text-muted-foreground">
        Page {page} of {totalPages || 1}
      </p>
      <Button
        variant="outline"
        size="sm"
        disabled={!hasNext || isLoading}
        onClick={onNext}
      >
        Next
      </Button>
    </div>
  );
}

/**
 * Helper to calculate pagination state from offset-based pagination.
 */
export function usePaginationFromOffset(
  offset: number,
  pageSize: number,
  total: number
) {
  const page = Math.floor(offset / pageSize) + 1;
  const totalPages = Math.ceil(total / pageSize);
  const hasPrev = offset > 0;
  const hasNext = offset + pageSize < total;

  return { page, totalPages, hasPrev, hasNext };
}
