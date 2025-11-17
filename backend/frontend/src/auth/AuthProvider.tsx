import { setAuthToken, setUnauthorizedHandler } from "@/api/client";
import { setUserAuthToken } from "@/api/userClient";
import {
  ADMIN_ACCESS_STORAGE_KEY,
  ADMIN_REFRESH_STORAGE_KEY,
  USER_ACCESS_STORAGE_KEY,
  USER_REFRESH_STORAGE_KEY,
} from "./storage";

import { createAuthStore } from "./createAuthStore";

const { AuthProvider, AuthContext } = createAuthStore({
  name: "Admin",
  accessStorageKey: ADMIN_ACCESS_STORAGE_KEY,
  refreshStorageKey: ADMIN_REFRESH_STORAGE_KEY,
  oidcCallbackPath: "/admin/ui/auth/oidc/callback",
  logoutRedirectPath: "/",
  clients: [
    { setAuthToken, setUnauthorizedHandler },
    { setAuthToken: setUserAuthToken },
  ],
  mirrors: [
    {
      accessKey: USER_ACCESS_STORAGE_KEY,
      refreshKey: USER_REFRESH_STORAGE_KEY,
    },
  ],
});

export { AuthProvider, AuthContext };
export {
  ADMIN_ACCESS_STORAGE_KEY,
  ADMIN_REFRESH_STORAGE_KEY,
} from "./storage";
