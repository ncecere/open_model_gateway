import { useMemo, useState, useCallback } from "react";
import { CalendarDays, ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";

export type DatePreset = "today" | "7d" | "30d" | "90d" | "ytd" | "custom";

export interface DateRangePresetsProps {
  startInput: string;
  endInput: string;
  onStartChange: (value: string) => void;
  onEndChange: (value: string) => void;
  rangeError?: string;
  rangeDisplay?: string | null;
  timezone?: string;
  idPrefix?: string;
  className?: string;
}

const PRESETS: { value: DatePreset; label: string }[] = [
  { value: "today", label: "Today" },
  { value: "7d", label: "7d" },
  { value: "30d", label: "30d" },
  { value: "90d", label: "90d" },
  { value: "ytd", label: "YTD" },
  { value: "custom", label: "Custom" },
];

function formatDateInput(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function startOfToday(): Date {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), now.getDate());
}

function addDays(date: Date, amount: number): Date {
  return new Date(date.getTime() + amount * 24 * 60 * 60 * 1000);
}

function startOfYear(): Date {
  const now = new Date();
  return new Date(now.getFullYear(), 0, 1);
}

function getPresetRange(preset: DatePreset): { start: string; end: string } {
  const today = startOfToday();
  const todayStr = formatDateInput(today);

  switch (preset) {
    case "today":
      return { start: todayStr, end: todayStr };
    case "7d":
      return { start: formatDateInput(addDays(today, -6)), end: todayStr };
    case "30d":
      return { start: formatDateInput(addDays(today, -29)), end: todayStr };
    case "90d":
      return { start: formatDateInput(addDays(today, -89)), end: todayStr };
    case "ytd":
      return { start: formatDateInput(startOfYear()), end: todayStr };
    default:
      return { start: "", end: "" };
  }
}

function detectPreset(startInput: string, endInput: string): DatePreset | null {
  for (const preset of PRESETS) {
    if (preset.value === "custom") continue;
    const range = getPresetRange(preset.value);
    if (startInput === range.start && endInput === range.end) {
      return preset.value;
    }
  }
  return null;
}

export function DateRangePresets({
  startInput,
  endInput,
  onStartChange,
  onEndChange,
  rangeError,
  rangeDisplay,
  timezone,
  idPrefix = "usage",
  className,
}: DateRangePresetsProps) {
  const [customOpen, setCustomOpen] = useState(false);

  const activePreset = useMemo(
    () => detectPreset(startInput, endInput),
    [startInput, endInput]
  );

  const handlePresetClick = useCallback(
    (preset: DatePreset) => {
      if (preset === "custom") {
        setCustomOpen(true);
        return;
      }
      const range = getPresetRange(preset);
      onStartChange(range.start);
      onEndChange(range.end);
      setCustomOpen(false);
    },
    [onStartChange, onEndChange]
  );

  const isCustomActive = activePreset === null && startInput && endInput;

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <div className="flex flex-wrap items-center gap-1.5">
        {PRESETS.filter((p) => p.value !== "custom").map((preset) => (
          <Button
            key={preset.value}
            variant={activePreset === preset.value ? "default" : "outline"}
            size="sm"
            onClick={() => handlePresetClick(preset.value)}
            className="h-8 px-3"
          >
            {preset.label}
          </Button>
        ))}

        <Popover open={customOpen} onOpenChange={setCustomOpen}>
          <PopoverTrigger asChild>
            <Button
              variant={isCustomActive ? "default" : "outline"}
              size="sm"
              className="h-8 gap-1.5"
            >
              <CalendarDays className="h-3.5 w-3.5" />
              Custom
              <ChevronDown className="h-3 w-3" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-80" align="end">
            <div className="space-y-4">
              <div className="space-y-2">
                <h4 className="font-medium text-sm">Custom date range</h4>
                <p className="text-xs text-muted-foreground">
                  Select a specific start and end date.
                </p>
              </div>
              <div className="grid gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor={`${idPrefix}-custom-start`}>Start date</Label>
                  <Input
                    id={`${idPrefix}-custom-start`}
                    type="date"
                    value={startInput}
                    onChange={(e) => onStartChange(e.target.value)}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor={`${idPrefix}-custom-end`}>End date</Label>
                  <Input
                    id={`${idPrefix}-custom-end`}
                    type="date"
                    value={endInput}
                    onChange={(e) => onEndChange(e.target.value)}
                  />
                </div>
              </div>
              {rangeError && (
                <p className="text-xs text-destructive">{rangeError}</p>
              )}
            </div>
          </PopoverContent>
        </Popover>
      </div>

      <p className="text-xs text-muted-foreground">
        {rangeError ? (
          <span className="text-destructive">{rangeError}</span>
        ) : (
          <>
            {rangeDisplay ?? "Select dates"}
            {timezone && ` · ${timezone}`}
          </>
        )}
      </p>
    </div>
  );
}
