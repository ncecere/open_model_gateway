import type { ReactNode } from "react";
import { useLocation } from "react-router-dom";
import { ErrorBoundary, type ErrorBoundaryProps } from "./ErrorBoundary";
import { logError } from "./errorLogging";

export type PageErrorBoundaryProps = {
  /** Children to render */
  children: ReactNode;
  /** Page name for error logging */
  pageName?: string;
  /** Custom fallback title */
  title?: string;
  /** Whether to show the error message */
  showError?: boolean;
  /** Additional error handler */
  onError?: ErrorBoundaryProps["onError"];
};

/**
 * PageErrorBoundary wraps page-level components with error handling.
 * Automatically resets when the route changes and logs errors with page context.
 */
export function PageErrorBoundary({
  children,
  pageName,
  title = "This page encountered an error",
  showError = true,
  onError,
}: PageErrorBoundaryProps) {
  const location = useLocation();

  const handleError: ErrorBoundaryProps["onError"] = (error, info) => {
    logError(error, {
      type: "page",
      pageName: pageName ?? location.pathname,
      componentStack: info.componentStack ?? undefined,
    });
    onError?.(error, info);
  };

  return (
    <ErrorBoundary
      variant="page"
      title={title}
      showError={showError}
      resetText="Reload page"
      resetKeys={[location.pathname]}
      onError={handleError}
    >
      {children}
    </ErrorBoundary>
  );
}
