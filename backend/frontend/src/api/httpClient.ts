import axios, { type AxiosError, type AxiosInstance, type AxiosRequestConfig } from "axios";

import { toast } from "@/hooks/use-toast";

export type UnauthorizedHandler = (error: AxiosError) => Promise<void> | void;

export type ClientFactoryOptions = {
  baseURL: string;
  skipAuthKey: string;
};

export type HttpClient = {
  instance: AxiosInstance;
  setAuthToken: (token?: string) => void;
  setUnauthorizedHandler: (handler?: UnauthorizedHandler) => void;
};

export function createHttpClient(options: ClientFactoryOptions): HttpClient {
  const instance = axios.create({
    baseURL: options.baseURL,
    withCredentials: true,
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
    },
  });

  let unauthorizedHandler: UnauthorizedHandler | undefined;

  const setUnauthorizedHandler = (handler?: UnauthorizedHandler) => {
    unauthorizedHandler = handler;
  };

  instance.interceptors.response.use(
    (response) => response,
    async (error: AxiosError<{ error?: string; message?: string }>) => {
      const { response, config } = error;
      const flagKey = options.skipAuthKey;

      if (response?.status === 401 && config && unauthorizedHandler) {
        const flaggedConfig = config as unknown as Record<string, unknown>;
        if (!flaggedConfig[flagKey]) {
          flaggedConfig[flagKey] = true;
          try {
            await unauthorizedHandler(error);
            return instance.request(config);
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

  const setAuthToken = (token?: string) => {
    if (!token) {
      delete instance.defaults.headers.common.Authorization;
      return;
    }
    instance.defaults.headers.common.Authorization = `Bearer ${token}`;
  };

  return {
    instance,
    setAuthToken,
    setUnauthorizedHandler,
  };
}

export type RequestConfig<D = unknown> = AxiosRequestConfig<D>;
