export type ErrorContext = {
  /** Type of error boundary that caught the error */
  type: "page" | "feature" | "app";
  /** Page name if caught by PageErrorBoundary */
  pageName?: string;
  /** Feature name if caught by FeatureErrorBoundary */
  featureName?: string;
  /** React component stack trace */
  componentStack?: string;
  /** Additional context */
  extra?: Record<string, unknown>;
};

/**
 * Logs an error with context information.
 * In production, this could be extended to send errors to a monitoring service.
 */
export function logError(error: Error, context: ErrorContext): void {
  const timestamp = new Date().toISOString();

  // Log to console with structured data
  console.error("[ErrorBoundary]", {
    timestamp,
    error: {
      name: error.name,
      message: error.message,
      stack: error.stack,
    },
    context,
  });

  // In production, you might send this to an error monitoring service:
  // - Sentry: Sentry.captureException(error, { extra: context })
  // - LogRocket: LogRocket.captureException(error, { extra: context })
  // - Custom endpoint: fetch('/api/errors', { method: 'POST', body: JSON.stringify({ error, context }) })
}

/**
 * Creates an error handler that logs errors with the given context.
 */
export function createErrorHandler(
  baseContext: Omit<ErrorContext, "componentStack">
): (error: Error, info: { componentStack?: string | null }) => void {
  return (error, info) => {
    logError(error, {
      ...baseContext,
      componentStack: info.componentStack ?? undefined,
    });
  };
}
