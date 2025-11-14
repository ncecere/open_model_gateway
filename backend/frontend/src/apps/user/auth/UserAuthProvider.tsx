import {
  createContext,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  fetchAuthMethods,
  loginLocal,
  logout,
  refreshSession,
  startOIDC,
  type TokenResponse,
} from "../../../api/auth";
import { setAuthToken } from "../../../api/client";
import {
  setUserAuthToken,
  setUserUnauthorizedHandler,
} from "../../../api/userClient";
import {
  ADMIN_ACCESS_STORAGE_KEY,
  ADMIN_REFRESH_STORAGE_KEY,
} from "../../../auth/AuthProvider";

const ACCESS_STORAGE_KEY = "og:user:access";
const REFRESH_STORAGE_KEY = "og:user:refresh";

type Session = {
  accessToken: string;
  method: string;
  user: TokenResponse["user"];
  accessExpiresAt: string;
  refreshExpiresAt: string;
  refreshToken?: string;
};

type AuthContextValue = {
  isAuthenticated: boolean;
  user?: TokenResponse["user"];
  methods: string[];
  accessToken?: string;
  loginLocal: (email: string, password: string) => Promise<void>;
  beginOIDC: () => Promise<void>;
  completeOIDC: (params: URLSearchParams) => Promise<void>;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
};

export const UserAuthContext = createContext<AuthContextValue | undefined>(
  undefined,
);

export function UserAuthProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [session, setSession] = useState<Session | undefined>(undefined);
  const [methods, setMethods] = useState<string[]>([]);
  const refreshPromise = useRef<Promise<void> | null>(null);

  const clearSession = useCallback(() => {
    setSession(undefined);
    localStorage.removeItem(ACCESS_STORAGE_KEY);
    localStorage.removeItem(REFRESH_STORAGE_KEY);
    setAuthToken(undefined);
    setUserAuthToken(undefined);
    localStorage.removeItem(ADMIN_ACCESS_STORAGE_KEY);
    localStorage.removeItem(ADMIN_REFRESH_STORAGE_KEY);
  }, []);

  const handleTokenResponse = useCallback((token: TokenResponse) => {
    const payload: Session = {
      accessToken: token.access_token,
      method: token.method,
      user: token.user,
      accessExpiresAt: token.access_expires_at,
      refreshExpiresAt: token.refresh_expires_at,
      refreshToken: token.refresh_token,
    };
    setSession(payload);
    localStorage.setItem(ACCESS_STORAGE_KEY, JSON.stringify(payload));
    if (token.refresh_token) {
      localStorage.setItem(REFRESH_STORAGE_KEY, token.refresh_token);
    } else {
      localStorage.removeItem(REFRESH_STORAGE_KEY);
    }
    localStorage.setItem(ADMIN_ACCESS_STORAGE_KEY, JSON.stringify(payload));
    if (token.refresh_token) {
      localStorage.setItem(ADMIN_REFRESH_STORAGE_KEY, token.refresh_token);
    } else {
      localStorage.removeItem(ADMIN_REFRESH_STORAGE_KEY);
    }
    setAuthToken(token.access_token);
    setUserAuthToken(token.access_token);
  }, []);

  useEffect(() => {
    fetchAuthMethods()
      .then(setMethods)
      .catch(() => setMethods([]));

    const storedAccess = localStorage.getItem(ACCESS_STORAGE_KEY);
    if (storedAccess) {
      try {
        const parsed = JSON.parse(storedAccess) as Session;
        setSession(parsed);
        if (parsed.accessToken) {
          setAuthToken(parsed.accessToken);
          setUserAuthToken(parsed.accessToken);
        }
      } catch {
        clearSession();
      }
    }
    const storedRefresh = localStorage.getItem(REFRESH_STORAGE_KEY);
    if (!storedRefresh) {
      return;
    }
    refreshSession(storedRefresh)
      .then(handleTokenResponse)
      .catch(clearSession);
  }, [clearSession, handleTokenResponse]);

  const performRefresh = useCallback(async () => {
    const refreshToken =
      session?.refreshToken ??
      localStorage.getItem(REFRESH_STORAGE_KEY) ??
      undefined;
    if (!refreshToken) {
      clearSession();
      throw new Error("missing refresh token");
    }
    const result = await refreshSession(refreshToken);
    handleTokenResponse(result);
  }, [session?.refreshToken, handleTokenResponse, clearSession]);

  const loginLocalHandler = useCallback(
    async (email: string, password: string) => {
      const result = await loginLocal({ email, password });
      handleTokenResponse(result);
    },
    [handleTokenResponse],
  );

  const beginOIDC = useCallback(async () => {
    const { auth_url } = await startOIDC("/auth/oidc/callback");
    window.location.href = auth_url;
  }, []);

  const completeOIDC = useCallback(
    async (params: URLSearchParams) => {
      const error = params.get("error");
      if (error) {
        throw new Error(error);
      }
      const result = await refreshSession();
      handleTokenResponse(result);
    },
    [handleTokenResponse],
  );

  const refresh = useCallback(async () => {
    if (!refreshPromise.current) {
      refreshPromise.current = performRefresh().finally(() => {
        refreshPromise.current = null;
      });
    }
    return refreshPromise.current;
  }, [performRefresh]);

  const logoutHandler = useCallback(async () => {
    await logout();
    clearSession();
  }, [clearSession]);

  useEffect(() => {
    setUserUnauthorizedHandler(async () => {
      await refresh();
    });
    return () => {
      setUserUnauthorizedHandler(undefined);
    };
  }, [refresh]);

  const value = useMemo<AuthContextValue>(
    () => ({
      isAuthenticated: Boolean(session?.accessToken),
      user: session?.user,
      methods,
      accessToken: session?.accessToken,
      loginLocal: loginLocalHandler,
      beginOIDC,
      completeOIDC,
      refresh,
      logout: logoutHandler,
    }),
    [
      session,
      methods,
      loginLocalHandler,
      beginOIDC,
      completeOIDC,
      refresh,
      logoutHandler,
    ],
  );

  return (
    <UserAuthContext.Provider value={value}>
      {children}
    </UserAuthContext.Provider>
  );
}
