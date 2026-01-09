import type { ReactNode } from "react";
import { ErrorBoundary as ReactErrorBoundary } from "react-error-boundary";
import { ErrorFallback } from "./ErrorFallback";
import { logError } from "./errorLogging";

export type AppErrorBoundaryProps = {
  /** Children to render */
  children: ReactNode;
  /** App name for error logging */
  appName?: string;
};

/**
 * AppErrorBoundary wraps the entire application with error handling.
 * This is the outermost error boundary and doesn't depend on React Router.
 * Provides a full-page error state when critical errors occur.
 */
export function AppErrorBoundary({
  children,
  appName = "app",
}: AppErrorBoundaryProps) {
  return (
    <ReactErrorBoundary
      fallbackRender={(props) => (
        <ErrorFallback
          {...props}
          variant="page"
          title="Application Error"
          showError={true}
          resetText="Reload Application"
        />
      )}
      onError={(error, info) => {
        logError(error, {
          type: "app",
          extra: {
            appName,
            componentStack: info.componentStack,
          },
        });
      }}
      onReset={() => {
        // Reload the page to reset all state
        window.location.reload();
      }}
    >
      {children}
    </ReactErrorBoundary>
  );
}
