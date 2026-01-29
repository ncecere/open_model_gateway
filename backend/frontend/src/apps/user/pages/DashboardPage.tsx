import { useEffect, useMemo, useState } from "react";
import { Activity, Building2, CircleDollarSign, Key } from "lucide-react";
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
import { PeriodSelector, usePeriodSelector } from "@/components/ui/period-selector";
import { PageHeader } from "@/components/layouts";
import { UsageBreakdownCard } from "../features/dashboard";

import { formatPricingCurrency } from "@/lib/formatters";

export function UserDashboardPage() {
  const [scopeSelection, setScopeSelection] = useState("personal");
  const periodSelector = usePeriodSelector("7d");
  const currentPeriod = periodSelector.value.period === "custom" ? "7d" : periodSelector.value.period;
  const periodLabel = periodSelector.getLabel();

  const { data, isLoading } = useUserDashboardQuery(
    currentPeriod,
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
      <PageHeader
        title="Dashboard"
        description="View metrics for your personal account or tenant keys you issued."
        actions={
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <PeriodSelector
              value={periodSelector.value}
              onChange={periodSelector.onChange}
              showCustom={false}
            />
            <Select
              value={scopeSelection}
              onValueChange={(value) => setScopeSelection(value)}
              disabled={!scopes.length}
            >
              <SelectTrigger className="w-full sm:w-48">
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
          </div>
        }
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          title="Requests"
          value={selectedTotals?.requests.toLocaleString() ?? 0}
          secondary={data ? periodLabel : undefined}
          loading={isLoading}
          icon={Activity}
        />
        <MetricCard
          title="Tokens"
          value={formatTokensShort(selectedTotals?.tokens ?? 0)}
          secondary="total processed"
          loading={isLoading}
          icon={Key}
        />
        <MetricCard
          title="Spend"
          value={
            selectedTotals
              ? formatPricingCurrency(spendValue ?? 0)
              : "—"
          }
          secondary={data ? periodLabel : undefined}
          loading={isLoading}
          icon={CircleDollarSign}
        />
        <MetricCard
          title="Tenants"
          value={tenantScopes.length}
          secondary={`${tenantScopes.filter((t) => t.status === "active").length} active`}
          loading={isLoading}
          icon={Building2}
        />
      </section>

      <section className="grid gap-4 md:grid-cols-2">
        <UsageBreakdownCard
          series={series}
          scopeName={selectedScope?.name ?? "Personal"}
          loading={isLoading}
        />
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
