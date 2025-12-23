import { useState, type ReactNode } from "react";
import { ChevronDown } from "lucide-react";

import { Skeleton } from "@/components/ui/skeleton";
import { formatUsageDate } from "@/lib/dates";
import { formatUsageUSD } from "@/features/usage/formatters";

export type DailySummary = {
  date: string;
  requests: number;
  tokens: number;
  cost_cents: number;
  cost_usd?: number;
};

export type UsageDailyTableProps<T, D extends DailySummary = DailySummary> = {
  days: D[];
  timezone: string;
  breakdownHeaders: string[];
  getBreakdown: (day: D) => T[];
  renderBreakdown: (item: T) => ReactNode;
  emptyMessage: string;
  currencyOptions?: { minimumFractionDigits?: number; maximumFractionDigits?: number };
};

export function UsageDailyTable<T, D extends DailySummary = DailySummary>({
  days,
  timezone,
  breakdownHeaders,
  getBreakdown,
  renderBreakdown,
  emptyMessage,
  currencyOptions,
}: UsageDailyTableProps<T, D>) {
  const [openState, setOpenState] = useState<Record<string, boolean>>({});

  if (!days.length) {
    return null;
  }

  const toggle = (date: string) => {
    setOpenState((prev) => ({ ...prev, [date]: !prev[date] }));
  };

  return (
    <div className="divide-y rounded-md border">
      {days.map((day) => {
        const isOpen = openState[day.date] ?? false;
        const breakdown = getBreakdown(day);
        return (
          <div key={day.date}>
            <button
              type="button"
              onClick={() => toggle(day.date)}
              className="flex w-full items-center gap-4 p-4 text-left"
            >
              <ChevronDown className={`h-4 w-4 flex-none transition ${isOpen ? "rotate-180" : ""}`} />
              <div className="flex flex-1 flex-col gap-1">
                <p className="font-medium">{formatUsageDate(day.date, timezone)}</p>
                <p className="text-xs text-muted-foreground">
                  {breakdown.length ? `${breakdown.length} entries` : emptyMessage}
                </p>
              </div>
              <div className="flex flex-none gap-6 text-sm text-muted-foreground">
                <span className="w-20 text-right">{day.requests.toLocaleString()} req</span>
                <span className="w-24 text-right">{day.tokens.toLocaleString()} tokens</span>
                <span className="w-20 text-right">
                  {formatUsageUSD(day.cost_usd, day.cost_cents, currencyOptions)}
                </span>
              </div>
            </button>
            {isOpen ? (
              <div className="space-y-3 border-t bg-muted/30 p-4 text-sm">
                {breakdown.length ? (
                  <div className="space-y-2">
                    <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 text-xs uppercase text-muted-foreground">
                      {breakdownHeaders.map((header) => (
                        <span key={header}>{header}</span>
                      ))}
                    </div>
                    {breakdown.map((item, idx) => (
                      <div key={idx}>{renderBreakdown(item)}</div>
                    ))}
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground">{emptyMessage}</p>
                )}
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

export function UsageDailySkeleton() {
  return (
    <div className="space-y-2">
      {[...Array(3)].map((_, idx) => (
        <Skeleton key={idx} className="h-16 w-full" />
      ))}
    </div>
  );
}
