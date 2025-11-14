import { useState, useEffect } from "react";

export const CUSTOM_RANGE_LIMIT_DAYS = 180;

type CustomRange = { start?: string; end?: string };

export function useCustomComparisonRange() {
  const [input, setInput] = useState<{ start: string; end: string }>({ start: "", end: "" });
  const [applied, setApplied] = useState<CustomRange>({});
  const [error, setError] = useState<string | null>(null);
  const [enabled, setEnabled] = useState(false);

  useEffect(() => {
    if (!enabled) {
      setApplied({});
      setError(null);
    }
  }, [enabled]);

  const apply = () => {
    const { start, end } = input;
    if (!start || !end) {
      setError("Select both start and end dates.");
      return false;
    }
    const startDate = new Date(`${start}T00:00:00Z`);
    const endDate = new Date(`${end}T23:59:59Z`);
    if (Number.isNaN(startDate.getTime()) || Number.isNaN(endDate.getTime())) {
      setError("Invalid date format.");
      return false;
    }
    if (endDate <= startDate) {
      setError("End date must be after start date.");
      return false;
    }
    const diffDays = Math.ceil((endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24));
    if (diffDays > CUSTOM_RANGE_LIMIT_DAYS) {
      setError(`Range cannot exceed ${CUSTOM_RANGE_LIMIT_DAYS} days.`);
      return false;
    }
    setApplied({ start: startDate.toISOString(), end: endDate.toISOString() });
    setError(null);
    setEnabled(true);
    return true;
  };

  return {
    input,
    setInput,
    applied,
    error,
    setEnabled,
    apply,
  };
}
