import { setAuthToken } from "@/api/client";
import {
  setUserAuthToken,
  setUserUnauthorizedHandler,
} from "@/api/userClient";
import { createAuthStore } from "@/auth/createAuthStore";
import {
  ADMIN_ACCESS_STORAGE_KEY,
  ADMIN_REFRESH_STORAGE_KEY,
  USER_ACCESS_STORAGE_KEY,
  USER_REFRESH_STORAGE_KEY,
} from "@/auth/storage";

const { AuthProvider, AuthContext } = createAuthStore({
  name: "User",
  accessStorageKey: USER_ACCESS_STORAGE_KEY,
  refreshStorageKey: USER_REFRESH_STORAGE_KEY,
  oidcCallbackPath: "/auth/oidc/callback",
  clients: [
    { setAuthToken },
    { setAuthToken: setUserAuthToken, setUnauthorizedHandler: setUserUnauthorizedHandler },
  ],
  mirrors: [
    {
      accessKey: ADMIN_ACCESS_STORAGE_KEY,
      refreshKey: ADMIN_REFRESH_STORAGE_KEY,
    },
  ],
});

export { AuthProvider as UserAuthProvider, AuthContext as UserAuthContext };
export {
  USER_ACCESS_STORAGE_KEY as ACCESS_STORAGE_KEY,
  USER_REFRESH_STORAGE_KEY as REFRESH_STORAGE_KEY,
} from "@/auth/storage";
