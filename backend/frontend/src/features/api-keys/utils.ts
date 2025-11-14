const scheduleLabelMap: Record<string, string> = {
  calendar_month: "Calendar month",
  weekly: "Weekly",
  rolling_7d: "Rolling 7 days",
  rolling_30d: "Rolling 30 days",
};

export function formatScheduleLabel(value?: string | null) {
  if (!value) {
    return "Inherits tenant defaults";
  }
  return scheduleLabelMap[value] ?? value.replace(/_/g, " ");
}

export function formatRateValue(value?: number | null) {
  if (!value || value <= 0) {
    return "—";
  }
  return value.toLocaleString();
}

export function computeNextResetDate(schedule?: string | null) {
  if (!schedule) {
    return null;
  }
  const now = new Date();
  switch (schedule) {
    case "calendar_month":
      return new Date(now.getFullYear(), now.getMonth() + 1, 1);
    case "weekly":
    case "rolling_7d": {
      const next = new Date(now);
      next.setDate(now.getDate() + 7);
      return next;
    }
    case "rolling_30d": {
      const next = new Date(now);
      next.setDate(now.getDate() + 30);
      return next;
    }
    default:
      return null;
  }
}
