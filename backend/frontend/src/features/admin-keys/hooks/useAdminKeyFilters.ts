import { useState, useMemo } from "react";
import type {
  AdminKeyRecord,
  AdminKeyScopeFilter,
  AdminKeyStatusFilter,
} from "../types";
import { getKeyStatus } from "../types";

export function useAdminKeyFilters(keys: AdminKeyRecord[]) {
  const [searchTerm, setSearchTerm] = useState("");
  const [scopeFilter, setScopeFilter] = useState<AdminKeyScopeFilter>("all");
  const [statusFilter, setStatusFilter] = useState<AdminKeyStatusFilter>("all");

  const filteredKeys = useMemo(() => {
    return keys.filter((key) => {
      // Search filter
      if (searchTerm) {
        const search = searchTerm.toLowerCase();
        const matchesName = key.name.toLowerCase().includes(search);
        const matchesPrefix = key.prefix.toLowerCase().includes(search);
        if (!matchesName && !matchesPrefix) return false;
      }

      // Scope filter
      if (scopeFilter !== "all" && key.scope !== scopeFilter) {
        return false;
      }

      // Status filter
      if (statusFilter !== "all") {
        const status = getKeyStatus(key);
        if (status !== statusFilter) return false;
      }

      return true;
    });
  }, [keys, searchTerm, scopeFilter, statusFilter]);

  const stats = useMemo(() => {
    let active = 0;
    let expired = 0;
    let revoked = 0;

    keys.forEach((key) => {
      const status = getKeyStatus(key);
      if (status === "active") active++;
      else if (status === "expired") expired++;
      else if (status === "revoked") revoked++;
    });

    return { total: keys.length, active, expired, revoked };
  }, [keys]);

  return {
    searchTerm,
    setSearchTerm,
    scopeFilter,
    setScopeFilter,
    statusFilter,
    setStatusFilter,
    filteredKeys,
    stats,
  };
}
