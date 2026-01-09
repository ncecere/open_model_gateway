import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { AlertTriangle } from "lucide-react";

export type EmptyStateProps = {
  /** Icon to display (defaults to AlertTriangle) */
  icon?: ReactNode;
  /** Main message */
  message: string;
  /** Additional description */
  description?: string;
  /** Action button or link */
  action?: ReactNode;
  /** Additional class name */
  className?: string;
  /** Whether to use dashed border style */
  dashed?: boolean;
};

/**
 * EmptyState displays a message when there's no data to show.
 * Commonly used in tables and lists.
 */
export function EmptyState({
  icon,
  message,
  description,
  action,
  className,
  dashed = true,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center gap-3 rounded-md border p-8 text-center text-sm text-muted-foreground",
        dashed && "border-dashed",
        className
      )}
    >
      <div className="text-muted-foreground">
        {icon ?? <AlertTriangle className="h-8 w-8" />}
      </div>
      <div className="space-y-1">
        <p className="font-medium text-foreground">{message}</p>
        {description && <p>{description}</p>}
      </div>
      {action && <div className="pt-2">{action}</div>}
    </div>
  );
}
