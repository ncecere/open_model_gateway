import {
  createContext,
  useCallback,
  useContext,
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
} from "@/api/auth";

export type AuthContextValue = {
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

export type ClientBinding = {
  setAuthToken: (token?: string) => void;
  setUnauthorizedHandler?: (handler?: () => Promise<void> | void) => void;
};

export type MirrorStorage = {
  accessKey: string;
  refreshKey?: string;
};

export type AuthStoreOptions = {
  name: string;
  accessStorageKey: string;
  refreshStorageKey: string;
  oidcCallbackPath: string;
  logoutRedirectPath?: string;
  clients: ClientBinding[];
  mirrors?: MirrorStorage[];
};

type Session = {
  accessToken: string;
  method: string;
  user: TokenResponse["user"];
  accessExpiresAt: string;
  refreshExpiresAt: string;
  refreshToken?: string;
};

export function createAuthStore(options: AuthStoreOptions) {
  const AuthContext = createContext<AuthContextValue | undefined>(undefined);

  function AuthProvider({ children }: { children: React.ReactNode }) {
    const [session, setSession] = useState<Session | undefined>(undefined);
    const [methods, setMethods] = useState<string[]>([]);
    const refreshPromise = useRef<Promise<void> | null>(null);

    const clearStoredSession = useCallback(() => {
      localStorage.removeItem(options.accessStorageKey);
      localStorage.removeItem(options.refreshStorageKey);
      options.mirrors?.forEach((mirror) => {
        localStorage.removeItem(mirror.accessKey);
        if (mirror.refreshKey) {
          localStorage.removeItem(mirror.refreshKey);
        }
      });
    }, []);

    const clearSession = useCallback(() => {
      setSession(undefined);
      clearStoredSession();
      options.clients.forEach((client) => {
        client.setAuthToken(undefined);
      });
    }, [clearStoredSession]);

    const persistSession = useCallback((payload: Session) => {
      const serialized = JSON.stringify(payload);
      localStorage.setItem(options.accessStorageKey, serialized);
      options.mirrors?.forEach((mirror) => {
        localStorage.setItem(mirror.accessKey, serialized);
      });

      const refreshValue = payload.refreshToken;
      if (refreshValue) {
        localStorage.setItem(options.refreshStorageKey, refreshValue);
        options.mirrors?.forEach((mirror) => {
          if (mirror.refreshKey) {
            localStorage.setItem(mirror.refreshKey, refreshValue);
          }
        });
      } else {
        localStorage.removeItem(options.refreshStorageKey);
        options.mirrors?.forEach((mirror) => {
          if (mirror.refreshKey) {
            localStorage.removeItem(mirror.refreshKey);
          }
        });
      }
    }, []);

    const handleTokenResponse = useCallback(
      (token: TokenResponse) => {
        const payload: Session = {
          accessToken: token.access_token,
          method: token.method,
          user: token.user,
          accessExpiresAt: token.access_expires_at,
          refreshExpiresAt: token.refresh_expires_at,
          refreshToken: token.refresh_token,
        };
        setSession(payload);
        persistSession(payload);
        options.clients.forEach((client) => {
          client.setAuthToken(token.access_token);
        });
      },
      [persistSession],
    );

    useEffect(() => {
      fetchAuthMethods()
        .then(setMethods)
        .catch(() => setMethods([]));

      const storedAccess = localStorage.getItem(options.accessStorageKey);
      if (storedAccess) {
        try {
          const parsed = JSON.parse(storedAccess) as Session;
          setSession(parsed);
          if (parsed.accessToken) {
            options.clients.forEach((client) => client.setAuthToken(parsed.accessToken));
          }
        } catch {
          clearSession();
        }
      }

      const storedRefresh = localStorage.getItem(options.refreshStorageKey);
      if (storedRefresh) {
        refreshSession(storedRefresh)
          .then((result) => handleTokenResponse(result as TokenResponse))
          .catch(() => {
            clearSession();
          });
      }
    }, [clearSession, handleTokenResponse]);

    const performRefresh = useCallback(async () => {
      const refreshToken =
        session?.refreshToken ??
        localStorage.getItem(options.refreshStorageKey) ??
        undefined;
      if (!refreshToken) {
        clearSession();
        throw new Error(`missing refresh token for ${options.name}`);
      }
      const result = (await refreshSession(refreshToken)) as TokenResponse;
      handleTokenResponse(result);
    }, [session?.refreshToken, handleTokenResponse, clearSession]);

    const loginLocalHandler = useCallback(
      async (email: string, password: string) => {
      const result = (await loginLocal({ email, password })) as TokenResponse;
      handleTokenResponse(result);
      },
      [handleTokenResponse],
    );

    const beginOIDC = useCallback(async () => {
      const { auth_url } = await startOIDC(options.oidcCallbackPath);
      window.location.href = auth_url;
    }, []);

    const completeOIDC = useCallback(
      async (params: URLSearchParams) => {
        const error = params.get("error");
        if (error) {
          throw new Error(error);
        }
        const result = (await refreshSession()) as TokenResponse;
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
      if (options.logoutRedirectPath) {
        window.location.href = options.logoutRedirectPath;
      }
    }, [clearSession]);

    useEffect(() => {
      const cleanups = options.clients
        .filter((client) => client.setUnauthorizedHandler)
        .map((client) => {
          client.setUnauthorizedHandler?.(async () => {
            await refresh();
          });
          return () => client.setUnauthorizedHandler?.(undefined);
        });
      return () => {
        cleanups.forEach((cleanup) => cleanup?.());
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

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
  }

  function useAuthStore() {
    const ctx = useContext(AuthContext);
    if (!ctx) {
      throw new Error(`use${options.name}Auth must be used within its provider`);
    }
    return ctx;
  }

  return {
    AuthContext,
    AuthProvider,
    useAuthStore,
  };
}
