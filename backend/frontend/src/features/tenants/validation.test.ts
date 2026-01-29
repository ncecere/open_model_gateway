import { describe, it, expect } from "vitest";
import {
  parseTenantFormValues,
  validateTenantForm,
  validateRateLimits,
  parseListInput,
  normalizeAliases,
  aliasSelectionsEqual,
  type TenantFormValues,
} from "./validation";

const defaults: TenantFormValues = {
  name: "Test Tenant",
  budgetUsd: "",
  warningThreshold: "",
  alertCooldown: "",
  requestsPerMinute: "",
  tokensPerMinute: "",
  parallelRequests: "",
  selectedModels: ["gpt-4"],
};

describe("parseTenantFormValues", () => {
  it("trims whitespace from all fields", () => {
    const parsed = parseTenantFormValues({ ...defaults, name: "  My Tenant  ", budgetUsd: " 100 " });
    expect(parsed.trimmedName).toBe("My Tenant");
    expect(parsed.trimmedBudget).toBe("100");
    expect(parsed.budgetValue).toBe(100);
  });

  it("detects rate override when any rate field is set", () => {
    const parsed = parseTenantFormValues({ ...defaults, requestsPerMinute: "60" });
    expect(parsed.hasRateOverride).toBe(true);
  });

  it("no rate override when all rate fields are empty", () => {
    const parsed = parseTenantFormValues(defaults);
    expect(parsed.hasRateOverride).toBe(false);
  });
});

describe("validateTenantForm", () => {
  it("returns null for valid form", () => {
    expect(validateTenantForm(defaults)).toBeNull();
  });

  it("requires name by default", () => {
    const err = validateTenantForm({ ...defaults, name: "" });
    expect(err?.field).toBe("name");
  });

  it("skips name when requireName is false", () => {
    const err = validateTenantForm({ ...defaults, name: "" }, { requireName: false });
    expect(err).toBeNull();
  });

  it("rejects negative budget", () => {
    const err = validateTenantForm({ ...defaults, budgetUsd: "-5" });
    expect(err?.field).toBe("budgetUsd");
  });

  it("rejects threshold > 1", () => {
    const err = validateTenantForm({ ...defaults, budgetUsd: "100", warningThreshold: "1.5" });
    expect(err?.field).toBe("warningThreshold");
  });

  it("requires all three rate fields if one is set", () => {
    const err = validateTenantForm({ ...defaults, requestsPerMinute: "60" });
    expect(err?.field).toBe("rateLimits");
  });

  it("requires at least one model by default", () => {
    const err = validateTenantForm({ ...defaults, selectedModels: [] });
    expect(err?.field).toBe("selectedModels");
  });
});

describe("validateRateLimits", () => {
  it("returns null when no rate override", () => {
    const parsed = parseTenantFormValues(defaults);
    expect(validateRateLimits(parsed)).toBeNull();
  });

  it("rejects negative RPM", () => {
    const parsed = parseTenantFormValues({
      ...defaults,
      requestsPerMinute: "-5",
      tokensPerMinute: "1000",
      parallelRequests: "10",
    });
    expect(validateRateLimits(parsed)?.field).toBe("requestsPerMinute");
  });
});

describe("parseListInput", () => {
  it("splits on commas", () => {
    expect(parseListInput("a,b,c")).toEqual(["a", "b", "c"]);
  });

  it("splits on newlines", () => {
    expect(parseListInput("a\nb\nc")).toEqual(["a", "b", "c"]);
  });

  it("trims and filters empty", () => {
    expect(parseListInput(" a , , b ")).toEqual(["a", "b"]);
  });
});

describe("normalizeAliases", () => {
  it("deduplicates and sorts", () => {
    expect(normalizeAliases(["b", "a", "b"])).toEqual(["a", "b"]);
  });
});

describe("aliasSelectionsEqual", () => {
  it("returns true for same elements in different order", () => {
    expect(aliasSelectionsEqual(["b", "a"], ["a", "b"])).toBe(true);
  });

  it("returns false for different lengths", () => {
    expect(aliasSelectionsEqual(["a"], ["a", "b"])).toBe(false);
  });

  it("returns false for different elements", () => {
    expect(aliasSelectionsEqual(["a", "c"], ["a", "b"])).toBe(false);
  });
});
