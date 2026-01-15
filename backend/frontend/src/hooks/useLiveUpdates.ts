import { useState, useCallback, useEffect } from "react";

export interface LiveUpdateOptions {
  /** Default interval in milliseconds. Defaults to 30000 (30 seconds). */
  defaultInterval?: number;
  /** Whether live updates are enabled by default. */
  defaultEnabled?: boolean;
  /** Available interval options to show in UI. */
  intervalOptions?: { label: string; value: number }[];
  /** Storage key for persisting preference. */
  storageKey?: string;
}

export interface LiveUpdateState {
  /** Whether live updates are currently enabled. */
  isLive: boolean;
  /** Current refresh interval in milliseconds. */
  interval: number;
  /** Toggle live updates on/off. */
  toggleLive: () => void;
  /** Set the refresh interval. */
  setInterval: (ms: number) => void;
  /** Available interval options for UI. */
  intervalOptions: { label: string; value: number }[];
  /** Timestamp of last update. */
  lastUpdated: Date | null;
  /** Update the last updated timestamp. */
  markUpdated: () => void;
}

const DEFAULT_INTERVALS = [
  { label: "5s", value: 5000 },
  { label: "15s", value: 15000 },
  { label: "30s", value: 30000 },
  { label: "1m", value: 60000 },
  { label: "5m", value: 300000 },
];

/**
 * Hook for managing live/real-time update state.
 * Persists preference to localStorage and provides interval controls.
 */
export function useLiveUpdates(options: LiveUpdateOptions = {}): LiveUpdateState {
  const {
    defaultInterval = 30000,
    defaultEnabled = true,
    intervalOptions = DEFAULT_INTERVALS,
    storageKey = "live-updates-preference",
  } = options;

  // Load persisted state
  const [isLive, setIsLive] = useState<boolean>(() => {
    if (storageKey) {
      const stored = localStorage.getItem(`${storageKey}-enabled`);
      if (stored !== null) {
        return stored === "true";
      }
    }
    return defaultEnabled;
  });

  const [interval, setIntervalState] = useState<number>(() => {
    if (storageKey) {
      const stored = localStorage.getItem(`${storageKey}-interval`);
      if (stored !== null) {
        const parsed = parseInt(stored, 10);
        if (!isNaN(parsed) && parsed > 0) {
          return parsed;
        }
      }
    }
    return defaultInterval;
  });

  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  // Persist changes to localStorage
  useEffect(() => {
    if (storageKey) {
      localStorage.setItem(`${storageKey}-enabled`, String(isLive));
    }
  }, [isLive, storageKey]);

  useEffect(() => {
    if (storageKey) {
      localStorage.setItem(`${storageKey}-interval`, String(interval));
    }
  }, [interval, storageKey]);

  const toggleLive = useCallback(() => {
    setIsLive((prev) => !prev);
  }, []);

  const setInterval = useCallback((ms: number) => {
    if (ms > 0) {
      setIntervalState(ms);
    }
  }, []);

  const markUpdated = useCallback(() => {
    setLastUpdated(new Date());
  }, []);

  return {
    isLive,
    interval,
    toggleLive,
    setInterval,
    intervalOptions,
    lastUpdated,
    markUpdated,
  };
}

/**
 * Formats a duration in milliseconds to a human-readable string.
 */
export function formatInterval(ms: number): string {
  if (ms < 60000) {
    return `${Math.round(ms / 1000)}s`;
  }
  if (ms < 3600000) {
    return `${Math.round(ms / 60000)}m`;
  }
  return `${Math.round(ms / 3600000)}h`;
}

/**
 * Formats a relative time (e.g., "5 seconds ago").
 */
export function formatRelativeTime(date: Date | null): string {
  if (!date) return "Never";

  const seconds = Math.round((Date.now() - date.getTime()) / 1000);

  if (seconds < 5) return "Just now";
  if (seconds < 60) return `${seconds}s ago`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m ago`;
  return `${Math.round(seconds / 3600)}h ago`;
}
