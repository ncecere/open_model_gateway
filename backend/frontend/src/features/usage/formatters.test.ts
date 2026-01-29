import { describe, it, expect } from "vitest";
import { formatUsageUSD, parseSpendValue } from "./formatters";

describe("formatUsageUSD", () => {
  it("formats usd value", () => {
    const result = formatUsageUSD(12.5);
    expect(result).toContain("12.50");
  });

  it("converts cents to usd", () => {
    const result = formatUsageUSD(undefined, 1250);
    expect(result).toContain("12.50");
  });

  it("defaults to zero when no args", () => {
    const result = formatUsageUSD();
    expect(result).toContain("0.00");
  });

  it("prefers usd over cents", () => {
    const result = formatUsageUSD(5, 9999);
    expect(result).toContain("5.00");
  });
});

describe("parseSpendValue", () => {
  it("returns usd when provided", () => {
    expect(parseSpendValue(10)).toBe(10);
  });

  it("converts cents to usd", () => {
    expect(parseSpendValue(undefined, 500)).toBe(5);
  });

  it("returns 0 for no inputs", () => {
    expect(parseSpendValue()).toBe(0);
  });

  it("skips NaN usd and uses cents", () => {
    expect(parseSpendValue(NaN, 200)).toBe(2);
  });

  it("skips zero usd and uses cents", () => {
    expect(parseSpendValue(0, 300)).toBe(3);
  });
});
