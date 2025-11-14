import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import {
  listTenants,
  type TenantRecord,
  type TenantStatus,
} from "@/api/tenants";

export const TENANTS_QUERY_KEY = ["tenants", "list"] as const;
export const TENANTS_DASHBOARD_KEY = ["tenants", "dashboard"] as const;

type UseTenantDirectoryOptions = {
  limit?: number;
};

export function useTenantDirectoryQuery(
  options: UseTenantDirectoryOptions = {},
) {
  const { limit = 100 } = options;
  return useQuery({
    queryKey: TENANTS_QUERY_KEY,
    queryFn: () => listTenants({ limit }),
  });
}

export type TenantStatusFilter = "all" | TenantStatus;

export function useTenantDirectoryFilters(tenants: TenantRecord[]) {
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] =
    useState<TenantStatusFilter>("all");

  const activeCount = useMemo(
    () => tenants.filter((tenant) => tenant.status === "active").length,
    [tenants],
  );

  const filteredTenants = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return tenants.filter((tenant) => {
      const matchesStatus =
        statusFilter === "all" || tenant.status === statusFilter;
      if (!matchesStatus) {
        return false;
      }
      if (!term) {
        return true;
      }
      return (
        tenant.name.toLowerCase().includes(term) ||
        tenant.id.toLowerCase().includes(term)
      );
    });
  }, [tenants, searchTerm, statusFilter]);

  const sortedTenants = useMemo(
    () =>
      [...filteredTenants].sort(
        (a, b) =>
          new Date(b.created_at).getTime() -
          new Date(a.created_at).getTime(),
      ),
    [filteredTenants],
  );

  return {
    searchTerm,
    setSearchTerm,
    statusFilter,
    setStatusFilter,
    activeCount,
    filteredTenants,
    sortedTenants,
  };
}
