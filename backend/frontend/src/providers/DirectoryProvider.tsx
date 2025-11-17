import { createContext, useContext, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";

import { listTenants, type TenantRecord } from "@/api/tenants";
import { listUsers, type AdminUser } from "@/api/users";
import { listModelCatalog, type ModelCatalogEntry } from "@/api/model-catalog";
import { useAuth } from "@/hooks/useAuth";

const DIRECTORY_TENANTS_KEY = ["tenants", "directory"] as const;
const DIRECTORY_USERS_KEY = ["users", "directory"] as const;
const DIRECTORY_MODELS_KEY = ["models", "directory"] as const;

export type DirectoryContextValue = {
  tenants: TenantRecord[];
  users: AdminUser[];
  models: ModelCatalogEntry[];
  tenantsLoading: boolean;
  usersLoading: boolean;
  modelsLoading: boolean;
  isLoading: boolean;
  refetchTenants: () => void;
  refetchUsers: () => void;
  refetchModels: () => void;
};

const DirectoryContext = createContext<DirectoryContextValue | undefined>(undefined);

export function DirectoryProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();

  const tenantsQuery = useQuery({
    queryKey: DIRECTORY_TENANTS_KEY,
    queryFn: () => listTenants({ limit: 500 }),
    enabled: isAuthenticated,
    staleTime: 60_000,
  });

  const usersQuery = useQuery({
    queryKey: DIRECTORY_USERS_KEY,
    queryFn: () => listUsers({ limit: 500 }),
    enabled: isAuthenticated,
    staleTime: 60_000,
  });

  const modelsQuery = useQuery({
    queryKey: DIRECTORY_MODELS_KEY,
    queryFn: listModelCatalog,
    enabled: isAuthenticated,
    staleTime: 60_000,
  });

  const tenants = tenantsQuery.data?.tenants ?? [];
  const users = usersQuery.data?.users ?? [];
  const models = modelsQuery.data ?? [];

  const value: DirectoryContextValue = {
    tenants,
    users,
    models,
    tenantsLoading: tenantsQuery.isFetching && isAuthenticated,
    usersLoading: usersQuery.isFetching && isAuthenticated,
    modelsLoading: modelsQuery.isFetching && isAuthenticated,
    isLoading:
      (tenantsQuery.isFetching || usersQuery.isFetching || modelsQuery.isFetching) &&
      isAuthenticated,
    refetchTenants: () => {
      void tenantsQuery.refetch();
    },
    refetchUsers: () => {
      void usersQuery.refetch();
    },
    refetchModels: () => {
      void modelsQuery.refetch();
    },
  };

  return <DirectoryContext.Provider value={value}>{children}</DirectoryContext.Provider>;
}

export function useDirectoryData() {
  const ctx = useContext(DirectoryContext);
  if (!ctx) {
    throw new Error("useDirectoryData must be used within a DirectoryProvider");
  }
  return ctx;
}
