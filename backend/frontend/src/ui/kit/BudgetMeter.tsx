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
};

export function BudgetMeter({ used, limit }: BudgetMeterProps) {
  const safeLimit =
    typeof limit === "number" && Number.isFinite(limit) && limit > 0
      ? limit
      : 0;
  const pct = safeLimit > 0 ? Math.min((used / safeLimit) * 100, 100) : 0;

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>{currencyFormatter.format(used)}</span>
        <span>{safeLimit > 0 ? currencyFormatter.format(safeLimit) : "—"}</span>
      </div>
      <Progress value={pct} className="h-2" />
    </div>
  );
}
