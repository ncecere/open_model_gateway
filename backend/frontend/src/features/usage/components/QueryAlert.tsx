import { AlertCircle } from "lucide-react";

import { Button } from "@/components/ui/button";

export function QueryAlert({
  error,
  onRetry,
}: {
  error?: Error | null;
  onRetry?: () => void;
}) {
  if (!error) {
    return null;
  }
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
      <div className="flex items-center gap-2">
        <AlertCircle className="h-4 w-4" />
        <span>{error.message}</span>
      </div>
      {onRetry ? (
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="text-destructive hover:text-destructive"
          onClick={onRetry}
        >
          Retry
        </Button>
      ) : null}
    </div>
  );
}
