import { api } from "./client";

export type AuthMethodsResponse = {
  methods: string[];
};

export type LoginRequest = {
  email: string;
  password: string;
};

export type TokenResponse = {
  access_token: string;
  access_expires_at: string;
  refresh_expires_at: string;
  refresh_token?: string;
  method: string;
  user: {
    id: string;
    email: string;
    name: string;
    theme_preference: ThemePreference;
    created_at: string;
    updated_at: string;
    last_login_at?: string | null;
    is_super_admin?: boolean;
  };
};

export type OIDCStartResponse = {
  auth_url: string;
  state: string;
};

export async function fetchAuthMethods() {
  const { data } = await api.get<AuthMethodsResponse>("/auth/methods");
  return data.methods;
}

export async function loginLocal(payload: LoginRequest) {
  const { data } = await api.post<TokenResponse>("/auth/login", payload);
  return data;
}

export async function refreshSession(refreshToken?: string) {
  const payload = refreshToken ? { refresh_token: refreshToken } : undefined;
  const { data } = await api.post<TokenResponse>("/auth/refresh", payload, {
    skipAuthRefresh: true,
  });
  return data;
}

export async function startOIDC(returnTo?: string) {
  const params = returnTo ? { return_to: returnTo } : undefined;
  const { data } = await api.get<OIDCStartResponse>("/auth/oidc/start", {
    params,
  });
  return data;
}

export async function logout() {
  await api.post("/auth/logout", undefined, { skipAuthRefresh: true });
}
import type { ThemePreference } from "@/types/theme";
