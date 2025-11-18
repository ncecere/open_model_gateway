import * as React from "react";

import { cn } from "@/lib/utils";

const Progress = React.forwardRef<
  HTMLDivElement,
  React.ComponentPropsWithoutRef<"div"> & {
    value?: number;
    indicatorClassName?: string;
  }
>(({ className, value = 0, indicatorClassName, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "relative h-2 w-full overflow-hidden rounded-full bg-muted",
      className,
    )}
    {...props}
  >
    <div
      className={cn("h-full flex-1 bg-primary transition-all", indicatorClassName)}
      style={{ width: `${Math.min(Math.max(value, 0), 100)}%` }}
    />
  </div>
));
Progress.displayName = "Progress";

export { Progress };
