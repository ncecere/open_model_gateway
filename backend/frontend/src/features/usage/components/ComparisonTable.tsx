import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { UsageComparisonSeries } from "@/api/usage";
import type { UsageComparisonMetric } from "@/components/charts/UsageComparisonChart";

function comparisonMetricValue(
  metric: UsageComparisonMetric,
  totals: UsageComparisonSeries["totals"],
) {
  switch (metric) {
    case "requests":
      return totals.requests;
    case "tokens":
      return totals.tokens;
    case "spend":
      if (typeof totals.cost_usd === "number") {
        return totals.cost_usd;
      }
      return totals.cost_cents / 100;
    default:
      return 0;
  }
}

function formatMetricDisplay(metric: UsageComparisonMetric, value: number) {
  if (metric === "spend") {
    return `$${value.toFixed(2)}`;
  }
  return value.toLocaleString();
}

export function ComparisonTable({
  series,
  metric,
  onRemove,
}: {
  series: UsageComparisonSeries[];
  metric: UsageComparisonMetric;
  onRemove: (seriesId: string) => void;
}) {
  if (!series.length) {
    return null;
  }

  const sorted = [...series].sort((a, b) => {
    const aValue = comparisonMetricValue(metric, a.totals);
    const bValue = comparisonMetricValue(metric, b.totals);
    return bValue - aValue;
  });

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium text-muted-foreground">
        Comparison summary
      </h3>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Entity</TableHead>
            <TableHead>
              {metric.charAt(0).toUpperCase() + metric.slice(1)}
            </TableHead>
            <TableHead className="text-right">Requests</TableHead>
            <TableHead className="text-right">Tokens</TableHead>
            <TableHead className="text-right">Spend</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sorted.map((entry) => (
            <TableRow key={entry.id}>
              <TableCell>
                <div>
                  <p className="font-medium">{entry.label}</p>
                  <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    <Badge variant="outline">
                      {entry.kind === "tenant"
                        ? "Tenant"
                        : entry.kind === "model"
                          ? "Model"
                          : "User"}
                    </Badge>
                    {entry.kind === "model" && entry.provider ? (
                      <span>{entry.provider}</span>
                    ) : null}
                    {entry.kind === "tenant" && entry.tenant_status ? (
                      <span>{entry.tenant_status}</span>
                    ) : null}
                    {entry.kind === "user" && (entry.user_email || entry.user_name) ? (
                      <span>{entry.user_email ?? entry.user_name}</span>
                    ) : null}
                  </div>
                </div>
              </TableCell>
              <TableCell>
                {formatMetricDisplay(
                  metric,
                  comparisonMetricValue(metric, entry.totals),
                )}
              </TableCell>
              <TableCell className="text-right">
                {entry.totals.requests.toLocaleString()}
              </TableCell>
              <TableCell className="text-right">
                {entry.totals.tokens.toLocaleString()}
              </TableCell>
              <TableCell className="text-right">
                {formatMetricDisplay("spend", comparisonMetricValue("spend", entry.totals))}
              </TableCell>
              <TableCell className="text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onRemove(entry.id)}
                >
                  Remove
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
