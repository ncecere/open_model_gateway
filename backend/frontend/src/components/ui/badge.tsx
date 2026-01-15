import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        // Solid variants
        default:
          "border-transparent bg-primary text-primary-foreground",
        secondary:
          "border-transparent bg-secondary text-secondary-foreground",
        destructive:
          "border-transparent bg-destructive text-destructive-foreground",
        success:
          "border-transparent bg-success text-success-foreground",
        warning:
          "border-transparent bg-warning text-warning-foreground",
        info:
          "border-transparent bg-info text-info-foreground",

        // Soft/muted variants (lighter backgrounds)
        "destructive-soft":
          "border-transparent bg-destructive-muted text-destructive",
        "success-soft":
          "border-transparent bg-success-muted text-success",
        "warning-soft":
          "border-transparent bg-warning-muted text-warning",
        "info-soft":
          "border-transparent bg-info-muted text-info",
        "muted":
          "border-transparent bg-muted text-muted-foreground",

        // Outline variants
        outline:
          "border-current text-foreground",
        "outline-destructive":
          "border-destructive/50 text-destructive",
        "outline-success":
          "border-success/50 text-success",
        "outline-warning":
          "border-warning/50 text-warning",
        "outline-info":
          "border-info/50 text-info",
      },
      size: {
        default: "px-2 py-0.5 text-xs",
        sm: "px-1.5 py-0 text-2xs",
        lg: "px-2.5 py-1 text-sm",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, size, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant, size }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
