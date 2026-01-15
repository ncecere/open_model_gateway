import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all duration-fast ease-out-expo focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground shadow-soft-sm hover:bg-primary/90 hover:shadow-soft-md hover:-translate-y-px active:translate-y-0 active:shadow-soft-sm",
        destructive:
          "bg-destructive text-destructive-foreground shadow-soft-sm hover:bg-destructive/90 hover:shadow-soft-md hover:-translate-y-px active:translate-y-0",
        success:
          "bg-success text-success-foreground shadow-soft-sm hover:bg-success/90 hover:shadow-soft-md hover:-translate-y-px active:translate-y-0",
        warning:
          "bg-warning text-warning-foreground shadow-soft-sm hover:bg-warning/90 hover:shadow-soft-md hover:-translate-y-px active:translate-y-0",
        outline:
          "border border-input bg-background hover:bg-muted hover:border-border-strong active:bg-muted/80",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80 active:bg-secondary/70",
        ghost:
          "hover:bg-muted active:bg-muted/80",
        link:
          "text-primary underline-offset-4 hover:underline",
        accent:
          "bg-accent text-accent-foreground shadow-soft-sm hover:bg-accent/90 hover:shadow-soft-md hover:-translate-y-px active:translate-y-0",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 rounded-md px-3 text-xs",
        lg: "h-10 rounded-md px-6",
        xl: "h-11 rounded-md px-8 text-base",
        icon: "h-9 w-9",
        "icon-sm": "h-8 w-8",
        "icon-lg": "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  loading?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    { className, variant, size, asChild = false, loading = false, children, disabled, ...props },
    ref
  ) => {
    const Comp = asChild ? Slot : "button";

    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        disabled={disabled || loading}
        {...props}
      >
        {loading ? (
          <>
            <Loader2 className="animate-spin" />
            <span className="sr-only">Loading</span>
            {children}
          </>
        ) : (
          children
        )}
      </Comp>
    );
  }
);
Button.displayName = "Button";

export { Button, buttonVariants };
