import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import {
  listPersonalTenants,
  type PersonalTenantRecord,
  type TenantStatus,
} from "@/api/tenants";

export type PersonalTenantStatusFilter = "all" | TenantStatus;

export function usePersonalTenantsQuery(limit = 500) {
  return useQuery({
    queryKey: ["personal-tenants", limit],
    queryFn: () => listPersonalTenants({ limit }),
  });
}

export function usePersonalTenantFilters(records: PersonalTenantRecord[]) {
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<PersonalTenantStatusFilter>("all");

  const filteredRecords = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return records.filter((record) => {
      const matchesStatus = statusFilter === "all" || record.status === statusFilter;
      if (!matchesStatus) {
        return false;
      }
      if (!term) {
        return true;
      }
      return (
        record.user_email.toLowerCase().includes(term) ||
        record.user_name.toLowerCase().includes(term)
      );
    });
  }, [records, searchTerm, statusFilter]);

  return {
    searchTerm,
    setSearchTerm,
    statusFilter,
    setStatusFilter,
    filteredRecords,
  };
}
