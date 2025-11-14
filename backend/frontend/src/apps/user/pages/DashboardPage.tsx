import { useEffect, useMemo, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { MetricCard } from "@/ui/kit/Cards";
import { formatTokensShort } from "@/lib/numbers";
import { useUserDashboardQuery } from "../hooks/useUserData";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 4,
  maximumFractionDigits: 4,
});

export function UserDashboardPage() {
  const [scopeSelection, setScopeSelection] = useState("personal");
  const { data, isLoading } = useUserDashboardQuery(
    "7d",
    scopeSelection === "personal" ? undefined : scopeSelection,
  );

  const scopes = useMemo(() => {
    if (data?.scopes?.length) {
      return data.scopes;
    }
    if (data) {
      return [
        {
          id: "personal",
          kind: "personal" as const,
          name: "Personal",
          totals: data.totals,
        },
      ];
    }
    return [];
  }, [data]);

  useEffect(() => {
    if (!scopes.length) {
      return;
    }
    if (!scopes.find((scope) => scope.id === scopeSelection)) {
      setScopeSelection(scopes[0].id);
    }
  }, [scopes, scopeSelection]);

  const scopeMap = useMemo(() => {
    const map = new Map<string, (typeof scopes)[number]>();
    scopes.forEach((scope) => map.set(scope.id, scope));
    return map;
  }, [scopes]);

  const selectedScope =
    data?.selected_scope?.scope ?? scopeMap.get(scopeSelection) ?? scopes[0];

  const selectedTotals = selectedScope?.totals ?? data?.totals;
  const spendValue =
    selectedTotals?.cost_usd ??
    (selectedTotals ? selectedTotals.cost_cents / 100 : 0);
  const series = data?.selected_scope?.series ?? data?.personal_series ?? [];
  const apiKeys =
    data?.selected_scope?.api_keys ?? data?.personal_api_keys ?? [];
  const tenantScopes = scopes.filter((scope) => scope.kind === "tenant");

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <p className="text-sm font-medium text-muted-foreground">Scope</p>
          <p className="text-xs text-muted-foreground">
            View metrics for your personal account or tenant keys you issued.
          </p>
        </div>
        <Select
          value={scopeSelection}
          onValueChange={(value) => setScopeSelection(value)}
          disabled={!scopes.length}
        >
          <SelectTrigger className="w-full md:w-72">
            <SelectValue placeholder="Select scope" />
          </SelectTrigger>
          <SelectContent>
            {scopes.map((scope) => (
              <SelectItem key={scope.id} value={scope.id}>
                {scope.kind === "personal" ? "Personal" : scope.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </section>

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          title="Requests"
          value={selectedTotals?.requests.toLocaleString() ?? 0}
          secondary={
            data
              ? `${Math.round(
                  (selectedTotals?.requests ?? 0) / 7,
                ).toLocaleString()} avg / day`
              : undefined
          }
          loading={isLoading}
        />
        <MetricCard
          title="Tokens"
          value={formatTokensShort(selectedTotals?.tokens ?? 0)}
          secondary="total processed"
          loading={isLoading}
        />
        <MetricCard
          title="Spend"
          value={
            selectedTotals
              ? currencyFormatter.format(spendValue ?? 0)
              : "—"
          }
          secondary={data ? `Window: ${data.period}` : undefined}
          loading={isLoading}
        />
        <MetricCard
          title="Tenants"
          value={tenantScopes.length}
          secondary={`${tenantScopes.filter((t) => t.status === "active").length} active`}
          loading={isLoading}
        />
      </section>

      <section className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>
              Recent usage — {selectedScope?.name ?? "Personal"}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-36 w-full" />
            ) : series.length ? (
              <div className="space-y-3 text-sm">
                {series.slice(-7).map((point) => (
                  <div
                    key={point.date}
                    className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-2"
                  >
                    <div>
                      <p className="font-medium">
                        {new Date(point.date).toLocaleDateString()}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {formatTokensShort(point.tokens)} tokens
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-semibold">
                        {point.requests.toLocaleString()} reqs
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {currencyFormatter.format(
                          point.cost_usd ?? point.cost_cents / 100,
                        )}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                No usage recorded yet.
              </p>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Recent API keys</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-36 w-full" />
            ) : apiKeys.length ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Prefix</TableHead>
                    <TableHead className="text-right">Last used</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {apiKeys.slice(0, 5).map((key) => (
                    <TableRow key={key.api_key_id}>
                      <TableCell className="font-medium">{key.name}</TableCell>
                      <TableCell>{key.prefix}</TableCell>
                      <TableCell className="text-right text-sm text-muted-foreground">
                        {key.last_used_at
                          ? new Date(key.last_used_at).toLocaleDateString()
                          : "—"}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">
                Keys for this scope will appear once created.
              </p>
            )}
          </CardContent>
        </Card>
      </section>

      <section>
        <Card>
          <CardHeader>
            <CardTitle>Tenant activity</CardTitle>
          </CardHeader>
          <CardContent className="overflow-x-auto">
            {isLoading ? (
              <Skeleton className="h-32 w-full" />
            ) : tenantScopes.length ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead className="text-right">Requests</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tenantScopes.map((scope) => (
                    <TableRow key={scope.id}>
                      <TableCell>{scope.name}</TableCell>
                      <TableCell className="capitalize">
                        {scope.status ?? "—"}
                      </TableCell>
                      <TableCell className="capitalize">
                        {scope.role ?? "member"}
                      </TableCell>
                      <TableCell className="text-right text-sm text-muted-foreground">
                        {scope.totals.requests.toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">
                You have not created tenant keys yet.
              </p>
            )}
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
