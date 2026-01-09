import { describe, it, expect } from "vitest";
import {
  formatBytes,
  formatDuration,
  formatDurationMs,
  formatNumber,
  formatCompactNumber,
  formatTokensShort,
  formatPercent,
  formatCurrency,
  formatCost,
  formatDate,
  formatShortDate,
  formatDateInput,
  formatBatchFinishedTimestamp,
  getBatchStatusVariant,
  dateFormatter,
  shortDateFormatter,
} from "../formatters";

describe("formatBytes", () => {
  it("formats zero bytes", () => {
    expect(formatBytes(0)).toBe("0 B");
  });

  it("formats bytes", () => {
    expect(formatBytes(500)).toBe("500 B");
  });

  it("formats kilobytes", () => {
    expect(formatBytes(1024)).toBe("1.0 KB");
    expect(formatBytes(1536)).toBe("1.5 KB");
  });

  it("formats megabytes", () => {
    expect(formatBytes(1024 * 1024)).toBe("1.0 MB");
    expect(formatBytes(10 * 1024 * 1024)).toBe("10 MB");
  });

  it("formats gigabytes", () => {
    expect(formatBytes(1024 * 1024 * 1024)).toBe("1.0 GB");
  });

  it("handles non-finite values", () => {
    expect(formatBytes(Infinity)).toBe("0 B");
    expect(formatBytes(NaN)).toBe("0 B");
  });
});

describe("formatDuration", () => {
  it("returns dash for null/undefined", () => {
    expect(formatDuration(null)).toBe("—");
    expect(formatDuration(undefined)).toBe("—");
  });

  it("returns dash for zero or negative", () => {
    expect(formatDuration(0)).toBe("—");
    expect(formatDuration(-10)).toBe("—");
  });

  it("formats seconds", () => {
    expect(formatDuration(45)).toBe("45s");
  });

  it("formats minutes", () => {
    expect(formatDuration(60)).toBe("1m");
    expect(formatDuration(90)).toBe("1m 30s");
  });

  it("formats hours", () => {
    expect(formatDuration(3600)).toBe("1h");
    expect(formatDuration(3660)).toBe("1h 1m");
  });

  it("formats days", () => {
    expect(formatDuration(86400)).toBe("1d");
    expect(formatDuration(90000)).toBe("1d 1h");
  });
});

describe("formatDurationMs", () => {
  it("converts milliseconds to seconds", () => {
    expect(formatDurationMs(60000)).toBe("1m");
    expect(formatDurationMs(3600000)).toBe("1h");
  });
});

describe("formatNumber", () => {
  it("formats numbers with locale separators", () => {
    const result = formatNumber(1234567);
    expect(result).toContain("1");
    expect(result).toContain("234");
    expect(result).toContain("567");
  });

  it("handles non-finite values", () => {
    expect(formatNumber(Infinity)).toBe("0");
    expect(formatNumber(NaN)).toBe("0");
  });
});

describe("formatCompactNumber", () => {
  it("formats millions", () => {
    const result = formatCompactNumber(1500000);
    expect(result.toLowerCase()).toContain("m");
  });

  it("formats thousands", () => {
    const result = formatCompactNumber(1500);
    expect(result.toLowerCase()).toContain("k");
  });
});

describe("formatTokensShort", () => {
  it("formats millions", () => {
    expect(formatTokensShort(1500000)).toBe("1.5M");
  });

  it("formats thousands", () => {
    expect(formatTokensShort(1500)).toBe("1.5K");
  });

  it("formats small numbers", () => {
    const result = formatTokensShort(500);
    expect(result).toContain("500");
  });

  it("handles non-finite values", () => {
    expect(formatTokensShort(Infinity)).toBe("0");
    expect(formatTokensShort(NaN)).toBe("0");
  });
});

describe("formatPercent", () => {
  it("formats percentages", () => {
    expect(formatPercent(0.5)).toBe("50.0%");
    expect(formatPercent(1)).toBe("100.0%");
    expect(formatPercent(0.123, 2)).toBe("12.30%");
  });

  it("handles non-finite values", () => {
    expect(formatPercent(Infinity)).toBe("0%");
  });
});

describe("formatCurrency", () => {
  it("formats USD by default", () => {
    const result = formatCurrency(1234.56);
    expect(result).toContain("1");
    expect(result).toContain("234");
    expect(result).toContain("56");
  });

  it("handles non-finite values", () => {
    expect(formatCurrency(NaN)).toBe("$0.00");
  });
});

describe("formatCost", () => {
  it("shows more precision for small values", () => {
    const result = formatCost(0.001234);
    expect(result).toContain("0.0012");
  });

  it("shows standard precision for normal values", () => {
    const result = formatCost(12.34);
    expect(result).toContain("12");
    expect(result).toContain("34");
  });

  it("handles zero", () => {
    expect(formatCost(0)).toBe("$0.00");
  });
});

describe("formatDate", () => {
  it("formats valid dates", () => {
    const result = formatDate(new Date("2024-01-15T12:00:00Z"));
    expect(result).toContain("Jan");
    expect(result).toContain("15");
    expect(result).toContain("2024");
  });

  it("formats date strings", () => {
    const result = formatDate("2024-06-20T10:30:00Z");
    expect(result).toContain("Jun");
    expect(result).toContain("20");
  });

  it("returns fallback for null/undefined", () => {
    expect(formatDate(null)).toBe("—");
    expect(formatDate(undefined)).toBe("—");
  });

  it("returns fallback for invalid dates", () => {
    expect(formatDate("invalid")).toBe("—");
  });
});

describe("formatShortDate", () => {
  it("formats dates without time", () => {
    const result = formatShortDate(new Date("2024-01-15T12:00:00Z"));
    expect(result).toContain("Jan");
    expect(result).toContain("15");
    expect(result).toContain("2024");
  });
});

describe("formatDateInput", () => {
  it("formats date for input fields", () => {
    const result = formatDateInput(new Date("2024-06-15T12:00:00Z"));
    expect(result).toBe("2024-06-15");
  });
});

describe("formatBatchFinishedTimestamp", () => {
  it("uses completed_at first", () => {
    const result = formatBatchFinishedTimestamp({
      completed_at: "2024-01-15T12:00:00Z",
      failed_at: null,
    });
    expect(result).toContain("Jan");
    expect(result).toContain("15");
  });

  it("falls back to failed_at", () => {
    const result = formatBatchFinishedTimestamp({
      completed_at: null,
      failed_at: "2024-02-20T10:00:00Z",
    });
    expect(result).toContain("Feb");
    expect(result).toContain("20");
  });

  it("returns dash when no timestamps", () => {
    expect(formatBatchFinishedTimestamp({})).toBe("—");
  });
});

describe("getBatchStatusVariant", () => {
  it("returns correct variants for known statuses", () => {
    expect(getBatchStatusVariant("completed")).toBe("default");
    expect(getBatchStatusVariant("failed")).toBe("destructive");
    expect(getBatchStatusVariant("in_progress")).toBe("secondary");
    expect(getBatchStatusVariant("queued")).toBe("outline");
  });

  it("returns outline for unknown statuses", () => {
    expect(getBatchStatusVariant("unknown_status")).toBe("outline");
  });
});

describe("dateFormatter and shortDateFormatter", () => {
  it("dateFormatter includes time", () => {
    const date = new Date("2024-01-15T14:30:00Z");
    const result = dateFormatter.format(date);
    expect(result).toContain("Jan");
    expect(result).toContain("15");
    expect(result).toContain("2024");
  });

  it("shortDateFormatter excludes time", () => {
    const date = new Date("2024-01-15T14:30:00Z");
    const result = shortDateFormatter.format(date);
    expect(result).toContain("Jan");
    expect(result).toContain("15");
    expect(result).toContain("2024");
  });
});
