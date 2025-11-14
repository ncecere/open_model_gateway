import axios, { type AxiosError, type AxiosRequestConfig } from "axios";
import { toast } from "@/hooks/use-toast";

const BASE_URL = "/user";

export const userApi = axios.create({
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
    userSkipAuthRefresh?: boolean;
  }
}

export function setUserUnauthorizedHandler(handler?: UnauthorizedHandler) {
  unauthorizedHandler = handler;
}

userApi.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<{ error?: string; message?: string }>) => {
    const { response, config } = error;

    if (response?.status === 401 && config && !config.userSkipAuthRefresh) {
      if (unauthorizedHandler) {
        config.userSkipAuthRefresh = true;
        try {
          await unauthorizedHandler(error);
          return userApi.request(config);
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

export function setUserAuthToken(token?: string) {
  if (!token) {
    delete userApi.defaults.headers.common.Authorization;
    return;
  }
  userApi.defaults.headers.common.Authorization = `Bearer ${token}`;
}

export type UserRequestConfig<D = unknown> = AxiosRequestConfig<D>;
