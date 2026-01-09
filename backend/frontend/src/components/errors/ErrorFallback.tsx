import type { FallbackProps } from "react-error-boundary";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export type ErrorFallbackProps = FallbackProps & {
  /** Title to display (defaults to "Something went wrong") */
  title?: string;
  /** Whether to show the error message */
  showError?: boolean;
  /** Whether to show the reset button */
  showReset?: boolean;
  /** Custom reset button text */
  resetText?: string;
  /** Layout variant */
  variant?: "page" | "card" | "inline";
  /** Additional class name */
  className?: string;
};

/**
 * ErrorFallback displays an error state with optional reset functionality.
 * Used as the fallback component for react-error-boundary.
 */
export function ErrorFallback({
  error,
  resetErrorBoundary,
  title = "Something went wrong",
  showError = true,
  showReset = true,
  resetText = "Try again",
  variant = "card",
  className,
}: ErrorFallbackProps) {
  const errorMessage = error?.message || "An unexpected error occurred";

  if (variant === "inline") {
    return (
      <div
        role="alert"
        className={cn(
          "flex items-center gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive",
          className
        )}
      >
        <AlertTriangle className="h-4 w-4 shrink-0" />
        <span className="flex-1">{showError ? errorMessage : title}</span>
        {showReset && (
          <Button
            variant="ghost"
            size="sm"
            onClick={resetErrorBoundary}
            className="h-auto p-1 text-destructive hover:text-destructive"
          >
            <RefreshCw className="h-3 w-3" />
          </Button>
        )}
      </div>
    );
  }

  if (variant === "page") {
    return (
      <div
        role="alert"
        className={cn(
          "flex min-h-[50vh] flex-col items-center justify-center gap-6 p-8 text-center",
          className
        )}
      >
        <div className="rounded-full bg-destructive/10 p-4">
          <AlertTriangle className="h-12 w-12 text-destructive" />
        </div>
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold">{title}</h1>
          {showError && (
            <p className="max-w-md text-sm text-muted-foreground">
              {errorMessage}
            </p>
          )}
        </div>
        {showReset && (
          <Button onClick={resetErrorBoundary} variant="outline">
            <RefreshCw className="mr-2 h-4 w-4" />
            {resetText}
          </Button>
        )}
      </div>
    );
  }

  // Default "card" variant
  return (
    <div
      role="alert"
      className={cn(
        "flex flex-col items-center gap-4 rounded-lg border border-destructive/50 bg-destructive/5 p-6 text-center",
        className
      )}
    >
      <AlertTriangle className="h-8 w-8 text-destructive" />
      <div className="space-y-1">
        <h3 className="font-medium">{title}</h3>
        {showError && (
          <p className="text-sm text-muted-foreground">{errorMessage}</p>
        )}
      </div>
      {showReset && (
        <Button onClick={resetErrorBoundary} variant="outline" size="sm">
          <RefreshCw className="mr-2 h-4 w-4" />
          {resetText}
        </Button>
      )}
    </div>
  );
}
