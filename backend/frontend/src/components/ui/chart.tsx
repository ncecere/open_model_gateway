"use client";

import * as React from "react";
import type { TooltipProps } from "recharts";
import { Tooltip } from "recharts";

import { cn } from "@/lib/utils";

export type ChartConfig = Record<string, { label?: string; color?: string }>;

interface ChartContainerProps extends React.HTMLAttributes<HTMLDivElement> {
  config: ChartConfig;
}

const ChartConfigContext = React.createContext<ChartConfig | null>(null);

export function ChartContainer({ config, className, children, ...props }: ChartContainerProps) {
  const style: React.CSSProperties = {};
  Object.entries(config).forEach(([key, value]) => {
    if (value?.color) {
      (style as Record<string, string>)[`--color-${key}`] = value.color;
    }
  });

  return (
    <ChartConfigContext.Provider value={config}>
      <div className={cn("flex w-full flex-col gap-4", className)} style={style} {...props}>
        {children}
      </div>
    </ChartConfigContext.Provider>
  );
}

export function ChartTooltip(props: TooltipProps<number, string>) {
  return <Tooltip {...props} />;
}

interface ChartTooltipContentProps extends TooltipProps<number, string> {
  indicator?: "line" | "dot";
  labelFormatter?: (label: string | number) => React.ReactNode;
  valueFormatter?: (payload: any) => React.ReactNode;
}

export function ChartTooltipContent({
  active,
  payload,
  label,
  indicator = "line",
  labelFormatter,
  valueFormatter,
}: ChartTooltipContentProps) {
  const config = React.useContext(ChartConfigContext);
  if (!active || !payload?.length) return null;

  const formattedLabel = labelFormatter ? labelFormatter(label ?? "") : label;

  return (
    <div className="rounded-lg border bg-popover px-3 py-2 text-xs shadow-sm">
      {formattedLabel ? <div className="mb-1 font-medium text-foreground">{formattedLabel}</div> : null}
      <div className="grid gap-1">
        {payload.map((item) => {
          if (!item || item.value == null) return null;
          const key = item.dataKey?.toString() ?? "";
          const color = item.color ?? (key ? `var(--color-${key})` : undefined);
          const labelText = config?.[key]?.label ?? item.name ?? key;
          const formattedValue = valueFormatter
            ? valueFormatter(item)
            : typeof item.value === "number"
              ? item.value.toLocaleString()
              : item.value;
          return (
            <div key={key} className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-2 text-muted-foreground">
                <span
                  className={cn(
                    indicator === "dot" ? "h-2 w-2 rounded-full" : "h-0.5 w-4 rounded-full",
                  )}
                  style={{ backgroundColor: color }}
                />
                <span>{labelText}</span>
              </div>
              <span className="font-medium text-foreground">{formattedValue}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
