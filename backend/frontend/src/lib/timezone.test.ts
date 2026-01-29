import { describe, it, expect } from "vitest";
import { getBrowserTimezone } from "./timezone";

describe("getBrowserTimezone", () => {
  it("returns a non-empty string", () => {
    const tz = getBrowserTimezone();
    expect(tz).toBeTruthy();
    expect(tz.length).toBeGreaterThan(0);
  });

  it("returns a valid IANA timezone string", () => {
    const tz = getBrowserTimezone();
    // Should contain a slash (e.g. "America/New_York") or be "UTC"
    expect(tz === "UTC" || tz.includes("/")).toBe(true);
  });
});
