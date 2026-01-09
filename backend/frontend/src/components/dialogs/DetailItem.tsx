import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export type DetailItemProps = {
  label: string;
  className?: string;
  children: ReactNode;
};

/**
 * DetailItem displays a label-value pair in a stacked layout.
 * Used within DetailsDialog for consistent formatting.
 */
export function DetailItem({ label, className, children }: DetailItemProps) {
  return (
    <div className={cn("", className)}>
      <p className="text-xs uppercase text-muted-foreground">{label}</p>
      <div className="text-sm font-medium text-foreground">{children}</div>
    </div>
  );
}
