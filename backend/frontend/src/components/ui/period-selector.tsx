import { useState } from "react";
import { Calendar } from "lucide-react";
import { Button } from "./button";
import { Input } from "./input";
import { Popover, PopoverContent, PopoverTrigger } from "./popover";
import { cn } from "@/lib/utils";

export type Period = "7d" | "30d" | "90d" | "custom";

export interface PeriodValue {
  period: Period;
  startDate?: string;
  endDate?: string;
}

interface PeriodSelectorProps {
  value: PeriodValue;
  onChange: (value: PeriodValue) => void;
  className?: string;
  showCustom?: boolean;
}

const PRESET_OPTIONS: { value: Period; label: string }[] = [
  { value: "7d", label: "7 days" },
  { value: "30d", label: "30 days" },
  { value: "90d", label: "90 days" },
];

function startOfToday() {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), now.getDate());
}

function formatDateInput(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function PeriodSelector({
  value,
  onChange,
  className,
  showCustom = true,
}: PeriodSelectorProps) {
  const [customOpen, setCustomOpen] = useState(false);
  const [tempStart, setTempStart] = useState(value.startDate ?? "");
  const [tempEnd, setTempEnd] = useState(value.endDate ?? "");

  const handlePresetClick = (period: Period) => {
    onChange({ period });
  };

  const handleApplyCustom = () => {
    if (tempStart && tempEnd) {
      onChange({
        period: "custom",
        startDate: tempStart,
        endDate: tempEnd,
      });
      setCustomOpen(false);
    }
  };

  const handleOpenCustom = () => {
    if (!tempStart || !tempEnd) {
      const today = startOfToday();
      const weekAgo = new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000);
      setTempStart(formatDateInput(weekAgo));
      setTempEnd(formatDateInput(today));
    }
    setCustomOpen(true);
  };

  const isValidRange =
    tempStart && tempEnd && new Date(tempEnd) >= new Date(tempStart);

  return (
    <div className={cn("flex items-center gap-1", className)}>
      {PRESET_OPTIONS.map((option) => (
        <Button
          key={option.value}
          variant={value.period === option.value ? "secondary" : "ghost"}
          size="sm"
          onClick={() => handlePresetClick(option.value)}
          className="h-8 px-3 text-xs"
        >
          {option.label}
        </Button>
      ))}
      {showCustom && (
        <Popover open={customOpen} onOpenChange={setCustomOpen}>
          <PopoverTrigger asChild>
            <Button
              variant={value.period === "custom" ? "secondary" : "ghost"}
              size="sm"
              onClick={handleOpenCustom}
              className="h-8 gap-1.5 px-3 text-xs"
            >
              <Calendar className="h-3.5 w-3.5" />
              {value.period === "custom" && value.startDate && value.endDate
                ? `${new Date(value.startDate).toLocaleDateString()} - ${new Date(value.endDate).toLocaleDateString()}`
                : "Custom"}
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-auto p-4" align="end">
            <div className="space-y-4">
              <div className="space-y-2">
                <label className="text-xs font-medium text-muted-foreground">
                  Start date
                </label>
                <Input
                  type="date"
                  value={tempStart}
                  onChange={(e) => setTempStart(e.target.value)}
                  className="h-9"
                />
              </div>
              <div className="space-y-2">
                <label className="text-xs font-medium text-muted-foreground">
                  End date
                </label>
                <Input
                  type="date"
                  value={tempEnd}
                  onChange={(e) => setTempEnd(e.target.value)}
                  className="h-9"
                />
              </div>
              <Button
                onClick={handleApplyCustom}
                disabled={!isValidRange}
                className="w-full"
                size="sm"
              >
                Apply
              </Button>
            </div>
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
}

export type UsagePeriod = "7d" | "30d" | "90d";

export interface QueryParams {
  period?: UsagePeriod;
  start?: string;
  end?: string;
  tenantId?: string;
}

export function usePeriodSelector(defaultPeriod: UsagePeriod = "7d") {
  const [value, setValue] = useState<PeriodValue>({ period: defaultPeriod });

  const getQueryParams = (): QueryParams => {
    if (value.period === "custom" && value.startDate && value.endDate) {
      const startDate = new Date(value.startDate);
      const endDate = new Date(value.endDate);
      endDate.setDate(endDate.getDate() + 1);
      return {
        start: startDate.toISOString(),
        end: endDate.toISOString(),
      };
    }
    return { period: value.period as UsagePeriod };
  };

  const getLabel = () => {
    if (value.period === "custom" && value.startDate && value.endDate) {
      return `${new Date(value.startDate).toLocaleDateString()} - ${new Date(value.endDate).toLocaleDateString()}`;
    }
    const days = parseInt(value.period);
    return `Last ${days} days`;
  };

  return {
    value,
    onChange: setValue,
    getQueryParams,
    getLabel,
    period: value.period,
  };
}
