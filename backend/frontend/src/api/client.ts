import { createHttpClient, type RequestConfig } from "./httpClient";

export const ADMIN_SKIP_AUTH_KEY = "__skipAdminRefresh";

const { instance: api, setAuthToken, setUnauthorizedHandler } = createHttpClient({
  baseURL: "/admin",
  skipAuthKey: ADMIN_SKIP_AUTH_KEY,
});

export function setTenantId(tenantId?: string) {
  if (!tenantId) {
    delete api.defaults.headers.common["X-Tenant-ID"];
    return;
  }
  api.defaults.headers.common["X-Tenant-ID"] = tenantId;
}
export { api, setAuthToken, setUnauthorizedHandler };

export type { RequestConfig };
