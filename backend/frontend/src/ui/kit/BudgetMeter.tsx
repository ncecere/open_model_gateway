import { Progress } from "@/components/ui/progress";

const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

type BudgetMeterProps = {
  used: number;
  limit?: number | null;
  warningThreshold?: number | null;
};

export function BudgetMeter({ used, limit, warningThreshold }: BudgetMeterProps) {
  const safeLimit =
    typeof limit === "number" && Number.isFinite(limit) && limit > 0
      ? limit
      : 0;
  const pct = safeLimit > 0 ? Math.min((used / safeLimit) * 100, 100) : 0;
  const ratio = safeLimit > 0 ? used / safeLimit : 0;
  const warn = normalizeThreshold(warningThreshold);
  const indicatorClassName = ratio >= 1
    ? "bg-destructive text-destructive-foreground"
    : ratio >= warn
      ? "bg-amber-500 text-black"
      : "bg-emerald-500 text-white";

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>{currencyFormatter.format(used)}</span>
        <span>{safeLimit > 0 ? currencyFormatter.format(safeLimit) : "—"}</span>
      </div>
      <Progress value={pct} className="h-2" indicatorClassName={indicatorClassName} />
    </div>
  );
}

function normalizeThreshold(value?: number | null) {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0.8;
  }
  if (value <= 0) {
    return 0.5;
  }
  if (value >= 1) {
    return 0.99;
  }
  return value;
}
