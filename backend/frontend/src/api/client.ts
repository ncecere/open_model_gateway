import axios, { type AxiosError, type AxiosRequestConfig } from "axios";
import { toast } from "@/hooks/use-toast";

const BASE_URL = "/admin";

export const api = axios.create({
  baseURL: BASE_URL,
  withCredentials: true,
  headers: {
    "Content-Type": "application/json",
    Accept: "application/json",
  },
});

type UnauthorizedHandler = (error: AxiosError) => Promise<void> | void;
let unauthorizedHandler: UnauthorizedHandler | undefined;

declare module "axios" {
  interface AxiosRequestConfig {
    skipAuthRefresh?: boolean;
  }
}

export function setUnauthorizedHandler(handler?: UnauthorizedHandler) {
  unauthorizedHandler = handler;
}

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<{ error?: string; message?: string }>) => {
    const { response, config } = error;

    if (response?.status === 401 && config && !config.skipAuthRefresh) {
      if (unauthorizedHandler) {
        config.skipAuthRefresh = true;
        try {
          await unauthorizedHandler(error);
          return api.request(config);
        } catch (refreshError) {
          return Promise.reject(refreshError);
        }
      }
    }

    const description =
      response?.data?.error || response?.data?.message || error.message;

    toast({
      variant: "destructive",
      title: "Request failed",
      description,
    });

    return Promise.reject(error);
  },
);

export function setAuthToken(token?: string) {
  if (!token) {
    delete api.defaults.headers.common.Authorization;
    return;
  }
  api.defaults.headers.common.Authorization = `Bearer ${token}`;
}

export function setTenantId(tenantId?: string) {
  if (!tenantId) {
    delete api.defaults.headers.common["X-Tenant-ID"];
    return;
  }
  api.defaults.headers.common["X-Tenant-ID"] = tenantId;
}

export type RequestConfig<D = unknown> = AxiosRequestConfig<D>;
