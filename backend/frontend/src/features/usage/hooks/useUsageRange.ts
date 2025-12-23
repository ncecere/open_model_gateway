import { useMemo, useState } from "react";

export type UsageRange = { start: string; end: string };

type RangeResult = { range: UsageRange } | { error: string };

type UsageRangeOptions = {
  defaultDays?: number;
  startLabel?: string;
  endLabel?: string;
};

function startOfToday() {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), now.getDate());
}

function addDays(date: Date, amount: number) {
  return new Date(date.getTime() + amount * 24 * 60 * 60 * 1000);
}

function formatDateInput(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function parseDateInput(value: string): Date | null {
  if (!value) return null;
  const [yearStr, monthStr, dayStr] = value.split("-");
  const year = Number(yearStr);
  const month = Number(monthStr);
  const day = Number(dayStr);
  if (!year || !month || !day) {
    return null;
  }
  return new Date(year, month - 1, day);
}

function formatInputDisplay(value: string) {
  const parsed = parseDateInput(value);
  if (!parsed) {
    return "Select dates";
  }
  return parsed.toLocaleDateString();
}

function deriveRangeISO(startInput: string, endInput: string): RangeResult {
  const startDate = parseDateInput(startInput);
  const endDate = parseDateInput(endInput);
  if (!startDate || !endDate) {
    return { error: "Select a valid start and end date." };
  }
  if (endDate.getTime() < startDate.getTime()) {
    return { error: "End date must be after start date." };
  }
  const startISO = startDate.toISOString();
  const endISO = addDays(endDate, 1).toISOString();
  return { range: { start: startISO, end: endISO } };
}

export function useUsageRange(options?: UsageRangeOptions) {
  const defaultDays = options?.defaultDays ?? 6;
  const [startInput, setStartInput] = useState(() =>
    formatDateInput(addDays(startOfToday(), -defaultDays)),
  );
  const [endInput, setEndInput] = useState(() =>
    formatDateInput(startOfToday()),
  );

  const rangeResult = useMemo(
    () => deriveRangeISO(startInput, endInput),
    [startInput, endInput],
  );
  const range = "range" in rangeResult ? rangeResult.range : undefined;
  const rangeError = "error" in rangeResult ? rangeResult.error : undefined;
  const rangeDisplay = range && !rangeError
    ? `${formatInputDisplay(startInput)} – ${formatInputDisplay(endInput)}`
    : null;

  return {
    startInput,
    endInput,
    setStartInput,
    setEndInput,
    range,
    rangeError,
    rangeDisplay,
    startLabel: options?.startLabel ?? "Start date",
    endLabel: options?.endLabel ?? "End date",
  };
}
