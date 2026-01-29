import { describe, it, expect } from "vitest";
import {
  ADMIN_ACCESS_STORAGE_KEY,
  ADMIN_REFRESH_STORAGE_KEY,
  USER_ACCESS_STORAGE_KEY,
  USER_REFRESH_STORAGE_KEY,
} from "./storage";

describe("auth storage keys", () => {
  it("admin access key has expected prefix", () => {
    expect(ADMIN_ACCESS_STORAGE_KEY).toBe("og:admin:access");
  });

  it("admin refresh key has expected prefix", () => {
    expect(ADMIN_REFRESH_STORAGE_KEY).toBe("og:admin:refresh");
  });

  it("user access key has expected prefix", () => {
    expect(USER_ACCESS_STORAGE_KEY).toBe("og:user:access");
  });

  it("user refresh key has expected prefix", () => {
    expect(USER_REFRESH_STORAGE_KEY).toBe("og:user:refresh");
  });

  it("admin and user keys are distinct", () => {
    expect(ADMIN_ACCESS_STORAGE_KEY).not.toBe(USER_ACCESS_STORAGE_KEY);
    expect(ADMIN_REFRESH_STORAGE_KEY).not.toBe(USER_REFRESH_STORAGE_KEY);
  });
});
