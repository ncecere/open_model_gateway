import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import type { ApiKeyRecord } from "@/api/tenants";
import {
  getTenantRateLimits,
  listAdminApiKeys,
  listPersonalTenants,
  listTenants,
} from "@/api/tenants";
import { getBudgetDefaults } from "@/api/budgets";
import { getRateLimitDefaults } from "@/api/rate-limits";
import { useDefaultSelection } from "@/hooks/useDefaultSelection";
import { formatCurrency } from "@/lib/formatters";
import type { TenantBudgetInfo, RateLimitInfo } from "../components/AdminKeyCreateDialog";

const TENANTS_QUERY_KEY = ["tenants", "list"] as const;

export function useKeysPageData() {
  const tenantsQuery = useQuery({
    queryKey: TENANTS_QUERY_KEY,
    queryFn: () => listTenants({ limit: 100 }),
  });

  const personalTenantsQuery = useQuery({
    queryKey: ["tenants", "personal"],
    queryFn: () => listPersonalTenants({ limit: 200 }),
  });

  const budgetDefaultsQuery = useQuery({
    queryKey: ["budget-defaults"],
    queryFn: getBudgetDefaults,
  });

  const rateLimitDefaultsQuery = useQuery({
    queryKey: ["rate-limit-defaults"],
    queryFn: getRateLimitDefaults,
  });

  const keysQuery = useQuery({
    queryKey: ["admin-api-keys"],
    queryFn: () => listAdminApiKeys(),
  });

  const tenants = tenantsQuery.data?.tenants ?? [];
  const personalTenants = personalTenantsQuery.data?.personal_tenants ?? [];
  const budgetDefaults = budgetDefaultsQuery.data;
  const rateLimitDefaults = rateLimitDefaultsQuery.data;

  const [selectedTenantId, setSelectedTenantId] = useState<string | undefined>(
    undefined,
  );
  useDefaultSelection({
    items: tenants,
    selected: selectedTenantId,
    onChange: setSelectedTenantId,
    getValue: (tenant) => tenant.id,
  });

  const tenantRateLimitQuery = useQuery({
    queryKey: ["tenant-rate-limits", selectedTenantId],
    queryFn: () =>
      selectedTenantId ? getTenantRateLimits(selectedTenantId) : Promise.resolve(null),
    enabled: Boolean(selectedTenantId),
  });

  const tenantBudgetMap = useMemo(() => {
    const fallbackLimit = budgetDefaults?.default_usd ?? null;
    const fallbackWarn = budgetDefaults?.warning_threshold_perc ?? 0.8;
    const map = new Map<string, TenantBudgetInfo>();
    tenants.forEach((tenant) => {
      map.set(tenant.id, {
        limit: tenant.budget_limit_usd ?? fallbackLimit,
        warning: tenant.warning_threshold ?? fallbackWarn,
      });
    });
    personalTenants.forEach((tenant) => {
      map.set(tenant.tenant_id, {
        limit: tenant.budget_limit_usd ?? fallbackLimit,
        warning: tenant.warning_threshold ?? fallbackWarn,
      });
    });
    return map;
  }, [tenants, personalTenants, budgetDefaults]);

  const defaultKeyRateLimit: RateLimitInfo | null = useMemo(
    () =>
      rateLimitDefaults
        ? {
            requests_per_minute: rateLimitDefaults.requests_per_minute,
            tokens_per_minute: rateLimitDefaults.tokens_per_minute,
            parallel_requests: rateLimitDefaults.parallel_requests_key,
          }
        : null,
    [rateLimitDefaults],
  );

  const defaultTenantRateLimit: RateLimitInfo | null = useMemo(
    () =>
      rateLimitDefaults
        ? {
            requests_per_minute: rateLimitDefaults.requests_per_minute,
            tokens_per_minute: rateLimitDefaults.tokens_per_minute,
            parallel_requests: rateLimitDefaults.parallel_requests_tenant,
          }
        : null,
    [rateLimitDefaults],
  );

  const effectiveTenantRateLimit = tenantRateLimitQuery.data ?? defaultTenantRateLimit;

  const resolveBudgetMeta = (key: ApiKeyRecord) => {
    const tenantBudget = tenantBudgetMap.get(key.tenant_id);
    const limit =
      key.quota?.budget_usd ??
      tenantBudget?.limit ??
      budgetDefaults?.default_usd ??
      0;
    const warning =
      key.quota?.warning_threshold ??
      tenantBudget?.warning ??
      budgetDefaults?.warning_threshold_perc ??
      0.8;
    return { limit, warning };
  };

  const formatBudgetValue = (key: ApiKeyRecord) => {
    const { limit } = resolveBudgetMeta(key);
    return formatCurrency(limit);
  };

  const formatWarningThresholdValue = (key: ApiKeyRecord) => {
    const { warning } = resolveBudgetMeta(key);
    return `${Math.round(warning * 100)}%`;
  };

  return {
    tenants,
    budgetDefaults,
    selectedTenantId,
    setSelectedTenantId,
    tenantBudgetMap,
    defaultKeyRateLimit,
    effectiveTenantRateLimit,
    keysQuery,
    formatBudgetValue,
    formatWarningThresholdValue,
  };
}

export function useKeysFilter(keys: ApiKeyRecord[]) {
  const [searchTerm, setSearchTerm] = useState("");
  const [issuerFilter, setIssuerFilter] = useState<"all" | "tenant" | "personal">("all");
  const [statusFilter, setStatusFilter] = useState<"all" | "active" | "revoked">("all");

  const filteredKeys = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return keys.filter((key) => {
      const matchesTerm =
        !term ||
        key.name.toLowerCase().includes(term) ||
        key.prefix.toLowerCase().includes(term) ||
        key.issuer?.label?.toLowerCase().includes(term) ||
        key.tenant_name?.toLowerCase().includes(term);
      const matchesIssuer =
        issuerFilter === "all" || key.issuer?.type === issuerFilter;
      const matchesStatus =
        statusFilter === "all" ||
        (statusFilter === "active" ? !key.revoked : key.revoked);
      return matchesTerm && matchesIssuer && matchesStatus;
    });
  }, [keys, searchTerm, issuerFilter, statusFilter]);

  const activeKeys = keys.filter((key) => !key.revoked);
  const revokedKeys = keys.filter((key) => key.revoked);

  return {
    searchTerm,
    setSearchTerm,
    issuerFilter,
    setIssuerFilter,
    statusFilter,
    setStatusFilter,
    filteredKeys,
    activeKeys,
    revokedKeys,
  };
}
