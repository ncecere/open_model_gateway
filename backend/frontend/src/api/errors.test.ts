import { describe, it, expect } from "vitest";
import { AxiosError } from "axios";
import { ApiError, toApiError, getErrorMessage, isAxiosError } from "./errors";

function makeAxiosError(status: number, data?: Record<string, unknown>): AxiosError {
  const err = new AxiosError("Request failed");
  err.response = {
    status,
    statusText: "Error",
    data: data ?? {},
    headers: {},
    config: {} as any,
  };
  return err;
}

describe("ApiError", () => {
  it("extracts status from response", () => {
    const e = new ApiError(makeAxiosError(404, { error: "Not found" }));
    expect(e.status).toBe(404);
    expect(e.message).toBe("Not found");
  });

  it("isUnauthorized for 401", () => {
    expect(new ApiError(makeAxiosError(401)).isUnauthorized()).toBe(true);
    expect(new ApiError(makeAxiosError(403)).isUnauthorized()).toBe(false);
  });

  it("isForbidden for 403", () => {
    expect(new ApiError(makeAxiosError(403)).isForbidden()).toBe(true);
  });

  it("isNotFound for 404", () => {
    expect(new ApiError(makeAxiosError(404)).isNotFound()).toBe(true);
  });

  it("isRateLimited for 429", () => {
    expect(new ApiError(makeAxiosError(429)).isRateLimited()).toBe(true);
  });

  it("isValidationError for 400", () => {
    expect(new ApiError(makeAxiosError(400)).isValidationError()).toBe(true);
  });

  it("isServerError for 5xx", () => {
    expect(new ApiError(makeAxiosError(500)).isServerError()).toBe(true);
    expect(new ApiError(makeAxiosError(503)).isServerError()).toBe(true);
    expect(new ApiError(makeAxiosError(499)).isServerError()).toBe(false);
  });

  it("isStatus matches specific code", () => {
    expect(new ApiError(makeAxiosError(418)).isStatus(418)).toBe(true);
  });
});

describe("toApiError", () => {
  it("returns existing ApiError as-is", () => {
    const e = new ApiError(makeAxiosError(400));
    expect(toApiError(e)).toBe(e);
  });

  it("wraps AxiosError", () => {
    const axErr = makeAxiosError(500, { error: "Internal" });
    const e = toApiError(axErr);
    expect(e).toBeInstanceOf(ApiError);
    expect(e.status).toBe(500);
  });

  it("wraps plain Error", () => {
    const e = toApiError(new Error("boom"));
    expect(e).toBeInstanceOf(ApiError);
    expect(e.message).toBe("boom");
  });

  it("wraps string", () => {
    const e = toApiError("something broke");
    expect(e).toBeInstanceOf(ApiError);
    expect(e.message).toBe("something broke");
  });
});

describe("getErrorMessage", () => {
  it("extracts from ApiError", () => {
    expect(getErrorMessage(new ApiError(makeAxiosError(400, { error: "bad" })))).toBe("bad");
  });

  it("extracts from plain Error", () => {
    expect(getErrorMessage(new Error("oops"))).toBe("oops");
  });

  it("stringifies non-errors", () => {
    expect(getErrorMessage(42)).toBe("42");
  });
});

describe("isAxiosError", () => {
  it("detects AxiosError", () => {
    expect(isAxiosError(makeAxiosError(400))).toBe(true);
  });

  it("rejects plain Error", () => {
    expect(isAxiosError(new Error("nope"))).toBe(false);
  });

  it("rejects non-Error", () => {
    expect(isAxiosError("string")).toBe(false);
  });
});
