import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { formatScheduleLabel, formatRateValue, computeNextResetDate } from "./utils";

describe("formatScheduleLabel", () => {
  it("returns default for null/undefined", () => {
    expect(formatScheduleLabel(null)).toBe("Inherits tenant defaults");
    expect(formatScheduleLabel(undefined)).toBe("Inherits tenant defaults");
  });

  it("maps known schedules", () => {
    expect(formatScheduleLabel("calendar_month")).toBe("Calendar month");
    expect(formatScheduleLabel("weekly")).toBe("Weekly");
    expect(formatScheduleLabel("rolling_7d")).toBe("Rolling 7 days");
    expect(formatScheduleLabel("rolling_30d")).toBe("Rolling 30 days");
  });

  it("falls back to replacing underscores", () => {
    expect(formatScheduleLabel("custom_schedule")).toBe("custom schedule");
  });
});

describe("formatRateValue", () => {
  it("returns dash for null/undefined/zero/negative", () => {
    expect(formatRateValue(null)).toBe("—");
    expect(formatRateValue(undefined)).toBe("—");
    expect(formatRateValue(0)).toBe("—");
    expect(formatRateValue(-1)).toBe("—");
  });

  it("formats positive numbers", () => {
    expect(formatRateValue(1000)).toBe("1,000");
  });
});

describe("computeNextResetDate", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 0, 15)); // Jan 15, 2026
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns null for null/undefined", () => {
    expect(computeNextResetDate(null)).toBeNull();
    expect(computeNextResetDate(undefined)).toBeNull();
  });

  it("returns first of next month for calendar_month", () => {
    const d = computeNextResetDate("calendar_month")!;
    expect(d.getFullYear()).toBe(2026);
    expect(d.getMonth()).toBe(1); // February
    expect(d.getDate()).toBe(1);
  });

  it("returns 7 days ahead for weekly", () => {
    const d = computeNextResetDate("weekly")!;
    expect(d.getDate()).toBe(22);
  });

  it("returns 30 days ahead for rolling_30d", () => {
    const d = computeNextResetDate("rolling_30d")!;
    expect(d.getMonth()).toBe(1); // February
    expect(d.getDate()).toBe(14);
  });

  it("returns null for unknown schedule", () => {
    expect(computeNextResetDate("unknown")).toBeNull();
  });
});
