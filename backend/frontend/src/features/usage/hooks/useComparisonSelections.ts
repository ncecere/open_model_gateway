import { useState } from "react";
import type { Dispatch, SetStateAction } from "react";

type ComparisonSelectionsOpts = {
  limit: number;
  onLimitReached?: () => void;
};

export function useComparisonSelections({ limit, onLimitReached }: ComparisonSelectionsOpts) {
  const [tenantIds, setTenantIds] = useState<string[]>([]);
  const [modelAliases, setModelAliases] = useState<string[]>([]);

  const total = tenantIds.length + modelAliases.length;

  const toggle = (
    value: string,
    setter: Dispatch<SetStateAction<string[]>>,
  ) => {
    setter((prev) => {
      if (prev.includes(value)) {
        return prev.filter((item) => item !== value);
      }
      if (total >= limit) {
        onLimitReached?.();
        return prev;
      }
      return [...prev, value];
    });
  };

  const clear = () => {
    setTenantIds([]);
    setModelAliases([]);
  };

  return {
    tenantIds,
    modelAliases,
    totalSelections: total,
    toggleTenant: (value: string) => toggle(value, setTenantIds),
    toggleModel: (value: string) => toggle(value, setModelAliases),
    setTenantIds,
    setModelAliases,
    clearSelections: clear,
  };
}
