import type { ReactNode } from "react";
import { ErrorBoundary, type ErrorBoundaryProps } from "./ErrorBoundary";
import { logError } from "./errorLogging";

export type FeatureErrorBoundaryProps = {
  /** Children to render */
  children: ReactNode;
  /** Feature name for error logging */
  featureName: string;
  /** Custom fallback title */
  title?: string;
  /** Whether to show the error message */
  showError?: boolean;
  /** Fallback variant (defaults to "card") */
  variant?: "card" | "inline";
  /** Additional class name */
  className?: string;
  /** Additional error handler */
  onError?: ErrorBoundaryProps["onError"];
  /** Keys that trigger a reset when changed */
  resetKeys?: ErrorBoundaryProps["resetKeys"];
};

/**
 * FeatureErrorBoundary wraps feature-level components with error handling.
 * Provides a contained error state that doesn't crash the entire page.
 */
export function FeatureErrorBoundary({
  children,
  featureName,
  title,
  showError = true,
  variant = "card",
  className,
  onError,
  resetKeys,
}: FeatureErrorBoundaryProps) {
  const handleError: ErrorBoundaryProps["onError"] = (error, info) => {
    logError(error, {
      type: "feature",
      featureName,
      componentStack: info.componentStack ?? undefined,
    });
    onError?.(error, info);
  };

  return (
    <ErrorBoundary
      variant={variant}
      title={title ?? `${featureName} encountered an error`}
      showError={showError}
      resetText="Retry"
      className={className}
      onError={handleError}
      resetKeys={resetKeys}
    >
      {children}
    </ErrorBoundary>
  );
}
