import { useState, type PropsWithChildren } from "react";
import {
  QueryClient,
  QueryClientProvider,
  type QueryClientConfig,
} from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { isAxiosError } from "axios";

const DEFAULT_OPTIONS: NonNullable<QueryClientConfig["defaultOptions"]> = {
  queries: {
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retry: (failureCount: number, error: unknown) => {
      if (failureCount >= 2) {
        return false;
      }
      if (isAxiosError(error) && error.response?.status === 401) {
        return false;
      }
      return true;
    },
  },
  mutations: {
    retry: 0,
  },
};

export function QueryProvider({ children }: PropsWithChildren) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: DEFAULT_OPTIONS,
      }),
  );

  return (
    <QueryClientProvider client={client}>
      {children}
      {import.meta.env.DEV ? (
        <ReactQueryDevtools initialIsOpen={false} />
      ) : null}
    </QueryClientProvider>
  );
}
