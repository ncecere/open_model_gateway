/**
 * Centralized formatting utilities for the Open Model Gateway UI.
 * This module consolidates all formatting functions to ensure consistency
 * across the application and eliminate duplicate implementations.
 */

// ============================================================================
// Date Formatting
// ============================================================================

/**
 * Default date-time formatter for displaying timestamps.
 * Format: "Jan 1, 2024, 12:00 PM"
 */
export const dateFormatter = new Intl.DateTimeFormat(undefined, {
  year: "numeric",
  month: "short",
  day: "numeric",
  hour: "2-digit",
  minute: "2-digit",
});

/**
 * Short date formatter (no time).
 * Format: "Jan 1, 2024"
 */
export const shortDateFormatter = new Intl.DateTimeFormat(undefined, {
  year: "numeric",
  month: "short",
  day: "numeric",
});

/**
 * Time-only formatter.
 * Format: "12:00 PM"
 */
export const timeFormatter = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
});

/**
 * ISO date formatter for input fields.
 * Format: "2024-01-01"
 */
export function formatDateInput(date: Date): string {
  return date.toISOString().split("T")[0];
}

/**
 * Formats a date string for usage charts with timezone support.
 */
export function formatUsageDate(
  value: string,
  timezone: string = "UTC",
  options?: Intl.DateTimeFormatOptions
): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  const formatter = new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    timeZone: timezone || "UTC",
    ...options,
  });
  return formatter.format(date);
}

/**
 * Formats a date with full timestamp.
 * Returns a formatted string or a fallback if the date is invalid.
 */
export function formatDate(date: Date | string | null | undefined, fallback = "—"): string {
  if (!date) return fallback;
  const d = typeof date === "string" ? new Date(date) : date;
  if (Number.isNaN(d.getTime())) return fallback;
  return dateFormatter.format(d);
}

/**
 * Formats a date as short date (no time).
 */
export function formatShortDate(date: Date | string | null | undefined, fallback = "—"): string {
  if (!date) return fallback;
  const d = typeof date === "string" ? new Date(date) : date;
  if (Number.isNaN(d.getTime())) return fallback;
  return shortDateFormatter.format(d);
}

/**
 * Formats a relative time (e.g., "2 hours ago", "in 3 days").
 */
export function formatRelativeTime(date: Date | string | null | undefined): string {
  if (!date) return "—";
  const d = typeof date === "string" ? new Date(date) : date;
  if (Number.isNaN(d.getTime())) return "—";

  const now = Date.now();
  const diff = d.getTime() - now;
  const absDiff = Math.abs(diff);

  const seconds = Math.floor(absDiff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });

  if (days > 0) {
    return rtf.format(diff > 0 ? days : -days, "day");
  }
  if (hours > 0) {
    return rtf.format(diff > 0 ? hours : -hours, "hour");
  }
  if (minutes > 0) {
    return rtf.format(diff > 0 ? minutes : -minutes, "minute");
  }
  return rtf.format(diff > 0 ? seconds : -seconds, "second");
}

// ============================================================================
// Duration Formatting
// ============================================================================

/**
 * Formats a duration in seconds to a human-readable string.
 * Examples: "1d", "2h", "30m", "45s"
 */
export function formatDuration(seconds: number | null | undefined): string {
  if (seconds == null || seconds <= 0) return "—";

  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  if (days > 0) {
    if (hours > 0) return `${days}d ${hours}h`;
    return `${days}d`;
  }
  if (hours > 0) {
    if (minutes > 0) return `${hours}h ${minutes}m`;
    return `${hours}h`;
  }
  if (minutes > 0) {
    if (secs > 0) return `${minutes}m ${secs}s`;
    return `${minutes}m`;
  }
  return `${secs}s`;
}

/**
 * Formats a duration in milliseconds to a human-readable string.
 */
export function formatDurationMs(ms: number | null | undefined): string {
  if (ms == null || ms <= 0) return "—";
  return formatDuration(Math.floor(ms / 1000));
}

// ============================================================================
// Number Formatting
// ============================================================================

/**
 * Formats a number with locale-specific separators.
 */
export function formatNumber(value: number, options?: Intl.NumberFormatOptions): string {
  if (!Number.isFinite(value)) return "0";
  return new Intl.NumberFormat(undefined, options).format(value);
}

/**
 * Formats a large number in compact form (e.g., "1.2M", "3.4K").
 */
export function formatCompactNumber(value: number): string {
  if (!Number.isFinite(value)) return "0";
  return new Intl.NumberFormat(undefined, {
    notation: "compact",
    compactDisplay: "short",
    maximumFractionDigits: 1,
  }).format(value);
}

/**
 * Formats token counts in short form (e.g., "1.2M", "3.4K").
 */
export function formatTokensShort(value: number): string {
  if (!Number.isFinite(value)) return "0";
  if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(1)}M`;
  }
  if (value >= 1_000) {
    return `${(value / 1_000).toFixed(1)}K`;
  }
  return value.toLocaleString();
}

/**
 * Formats a percentage value.
 */
export function formatPercent(value: number, decimals = 1): string {
  if (!Number.isFinite(value)) return "0%";
  return `${(value * 100).toFixed(decimals)}%`;
}

// ============================================================================
// Currency Formatting
// ============================================================================

/**
 * Formats a monetary value in USD.
 */
export function formatCurrency(
  value: number,
  currency = "USD",
  options?: Partial<Intl.NumberFormatOptions>
): string {
  if (!Number.isFinite(value)) return "$0.00";
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
    ...options,
  }).format(value);
}

/**
 * Formats a cost value with appropriate precision.
 * For small values (< $0.01), shows more decimal places.
 */
export function formatCost(value: number): string {
  if (!Number.isFinite(value)) return "$0.00";
  if (value === 0) return "$0.00";
  if (Math.abs(value) < 0.01) {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency: "USD",
      minimumFractionDigits: 4,
      maximumFractionDigits: 6,
    }).format(value);
  }
  return formatCurrency(value);
}

// ============================================================================
// File Size Formatting
// ============================================================================

/**
 * Formats a byte count to a human-readable string.
 * Examples: "1.5 KB", "2.3 MB", "1 GB"
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  if (!Number.isFinite(bytes)) return "0 B";

  const units = ["B", "KB", "MB", "GB", "TB", "PB"];
  const idx = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, idx);
  return `${value.toFixed(value < 10 ? 1 : 0)} ${units[idx]}`;
}

// ============================================================================
// Status Formatting
// ============================================================================

/**
 * Batch status to badge variant mapping.
 */
export const batchStatusVariants: Record<
  string,
  "default" | "secondary" | "outline" | "destructive"
> = {
  queued: "outline",
  validating: "outline",
  running: "secondary",
  in_progress: "secondary",
  finalizing: "secondary",
  cancelling: "secondary",
  completed: "default",
  expired: "outline",
  failed: "destructive",
  cancelled: "destructive",
};

/**
 * Gets the appropriate badge variant for a batch status.
 */
export function getBatchStatusVariant(
  status: string
): "default" | "secondary" | "outline" | "destructive" {
  return batchStatusVariants[status] ?? "outline";
}

// ============================================================================
// Batch-specific Formatting
// ============================================================================

export interface BatchTimestamps {
  completed_at?: string | null;
  failed_at?: string | null;
  cancelled_at?: string | null;
  expired_at?: string | null;
}

/**
 * Formats the finished timestamp for a batch.
 * Returns the first available terminal timestamp.
 */
export function formatBatchFinishedTimestamp(batch: BatchTimestamps): string {
  const ts =
    batch.completed_at || batch.failed_at || batch.cancelled_at || batch.expired_at;
  return ts ? dateFormatter.format(new Date(ts)) : "—";
}

// ============================================================================
// Utility Exports
// ============================================================================

/**
 * Re-export for backwards compatibility with existing code.
 */
export { formatTokensShort as formatTokens };
