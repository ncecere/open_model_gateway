import type { ReactNode } from "react";
import {
  ErrorBoundary as ReactErrorBoundary,
  type ErrorBoundaryProps as ReactErrorBoundaryProps,
} from "react-error-boundary";
import { ErrorFallback, type ErrorFallbackProps } from "./ErrorFallback";

export type ErrorBoundaryProps = {
  /** Children to render */
  children: ReactNode;
  /** Error fallback variant */
  variant?: ErrorFallbackProps["variant"];
  /** Custom fallback title */
  title?: string;
  /** Whether to show the error message */
  showError?: boolean;
  /** Whether to show the reset button */
  showReset?: boolean;
  /** Custom reset button text */
  resetText?: string;
  /** Called when an error is caught */
  onError?: ReactErrorBoundaryProps["onError"];
  /** Called when the error boundary is reset */
  onReset?: ReactErrorBoundaryProps["onReset"];
  /** Keys that trigger a reset when changed */
  resetKeys?: ReactErrorBoundaryProps["resetKeys"];
  /** Additional class name for the fallback */
  className?: string;
};

/**
 * ErrorBoundary wraps react-error-boundary with sensible defaults.
 * Catches JavaScript errors in child components and displays a fallback UI.
 */
export function ErrorBoundary({
  children,
  variant = "card",
  title,
  showError = true,
  showReset = true,
  resetText,
  onError,
  onReset,
  resetKeys,
  className,
}: ErrorBoundaryProps) {
  return (
    <ReactErrorBoundary
      fallbackRender={(props) => (
        <ErrorFallback
          {...props}
          variant={variant}
          title={title}
          showError={showError}
          showReset={showReset}
          resetText={resetText}
          className={className}
        />
      )}
      onError={onError}
      onReset={onReset}
      resetKeys={resetKeys}
    >
      {children}
    </ReactErrorBoundary>
  );
}
