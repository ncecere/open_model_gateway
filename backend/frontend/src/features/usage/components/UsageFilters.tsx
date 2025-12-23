import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export type UsageFiltersProps = {
  startInput: string;
  endInput: string;
  onStartChange: (value: string) => void;
  onEndChange: (value: string) => void;
  rangeError?: string;
  rangeDisplay?: string | null;
  timezone?: string;
  startLabel?: string;
  endLabel?: string;
  idPrefix?: string;
};

export function UsageFilters({
  startInput,
  endInput,
  onStartChange,
  onEndChange,
  rangeError,
  rangeDisplay,
  timezone,
  startLabel = "Start date",
  endLabel = "End date",
  idPrefix = "usage",
}: UsageFiltersProps) {
  return (
    <div className="w-full space-y-2">
      <div className="grid gap-2 sm:grid-cols-2">
        <div className="space-y-1">
          <Label htmlFor={`${idPrefix}-start`}>{startLabel}</Label>
          <Input
            id={`${idPrefix}-start`}
            type="date"
            value={startInput}
            onChange={(event) => onStartChange(event.target.value)}
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor={`${idPrefix}-end`}>{endLabel}</Label>
          <Input
            id={`${idPrefix}-end`}
            type="date"
            value={endInput}
            onChange={(event) => onEndChange(event.target.value)}
          />
        </div>
      </div>
      {rangeError ? (
        <p className="text-xs text-destructive">{rangeError}</p>
      ) : (
        <p className="text-xs text-muted-foreground">
          Times shown in {timezone ?? "UTC"}. Range: {rangeDisplay ?? "Select dates"}
        </p>
      )}
    </div>
  );
}
