export const dateFormatter = new Intl.DateTimeFormat(undefined, {
  year: "numeric",
  month: "short",
  day: "numeric",
  hour: "2-digit",
  minute: "2-digit",
});

export const statusVariants: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  queued: "outline",
  running: "secondary",
  in_progress: "secondary",
  finalizing: "secondary",
  completed: "default",
  failed: "destructive",
  cancelled: "destructive",
};

export const BATCH_PAGE_SIZE = 20;

export const formatFinishedTimestamp = (batch: { completed_at?: string | null; failed_at?: string | null; cancelled_at?: string | null }) => {
  const ts = batch.completed_at || batch.failed_at || batch.cancelled_at;
  return ts ? dateFormatter.format(new Date(ts)) : "—";
};
