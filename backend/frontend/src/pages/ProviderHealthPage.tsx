import { useMemo, useState, type ReactNode } from "react";
import { AlertTriangle, Gauge, RefreshCcw, ShieldCheck, Siren } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { PageHeader } from "@/components/layouts";

import { useProviderAlerts, useProviderIncidents, useProviderSLIs } from "@/api/hooks/useTelemetry";
import {
  clearTelemetrySeed,
  listProviderIncidents,
  listProviderSLIs,
  type ProviderIncident,
  type ProviderSLI,
} from "@/api/telemetry";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
// Charts temporarily removed per request; keep imports minimal.
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useDirectoryData } from "@/providers/DirectoryProvider";

function formatPercent(value: number) {
  return `${(value * 100).toFixed(1)}%`;
}

function formatMs(ms: number) {
  return `${ms.toLocaleString()} ms`;
}

function statusForSLI(sli: ProviderSLI) {
  if (sli.request_count < sli.thresholds.min_samples) {
    return { label: "Cold", tone: "muted" as const };
  }
  if (
    sli.error_rate > sli.thresholds.error_rate_threshold ||
    sli.timeout_rate > sli.thresholds.timeout_rate_threshold ||
    sli.latency_p95_ms > sli.thresholds.latency_p95_ms
  ) {
    return { label: "Degraded", tone: "warn" as const };
  }
  return { label: "Healthy", tone: "ok" as const };
}

function StatusBadge({ label, tone }: { label: string; tone: "ok" | "warn" | "muted" }) {
  const variant =
    tone === "ok" ? "default" : tone === "warn" ? "destructive" : "secondary";
  return <Badge variant={variant}>{label}</Badge>;
}

export function ProviderHealthPage() {
  const queryClient = useQueryClient();
  const { models } = useDirectoryData();
  const [providerFilter, setProviderFilter] = useState<string>();
  const [aliasFilter, setAliasFilter] = useState<string>();
  const [searchRoute, setSearchRoute] = useState("");
  const [incidentStatus, setIncidentStatus] = useState<"all" | "open" | "resolved">("all");
  const [selectedIncident, setSelectedIncident] = useState<ProviderIncident | null>(null);
  const [clearingSeed, setClearingSeed] = useState(false);

  const slisQuery = useProviderSLIs({
    provider: providerFilter || undefined,
    alias: aliasFilter || undefined,
  });
  const incidentsQuery = useProviderIncidents({
    provider: providerFilter || undefined,
    alias: aliasFilter || undefined,
    status: incidentStatus === "all" ? undefined : incidentStatus,
  });
  const alertsQuery = useProviderAlerts({
    provider: providerFilter || undefined,
    alias: aliasFilter || undefined,
  });

  const providerOptions = useMemo(() => {
    const set = new Set<string>();
    (slisQuery.data ?? []).forEach((sli) => set.add(sli.provider));
    return Array.from(set);
  }, [slisQuery.data]);

  const aliasOptions = useMemo(() => {
    const fromModels = models.map((m) => m.alias);
    const fromData = (slisQuery.data ?? []).map((sli) => sli.alias);
    return Array.from(new Set([...fromModels, ...fromData])).sort();
  }, [models, slisQuery.data]);

  const filteredSLIs = useMemo(() => {
    const search = searchRoute.trim().toLowerCase();
    return (slisQuery.data ?? []).filter((sli) =>
      search ? (sli.route ?? "").toLowerCase().includes(search) : true,
    );
  }, [slisQuery.data, searchRoute]);

  const totalDegraded = useMemo(
    () => filteredSLIs.filter((sli) => statusForSLI(sli).tone === "warn").length,
    [filteredSLIs],
  );

  const seedSampleData = async () => {
    try {
      await Promise.all([
        listProviderSLIs({ seed: true }),
        listProviderIncidents({ seed: true }),
      ]);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["provider-slis"] }),
        queryClient.invalidateQueries({ queryKey: ["provider-incidents"] }),
        queryClient.invalidateQueries({ queryKey: ["provider-alerts"] }),
      ]);
    } catch (err) {
      console.error("seed telemetry data failed", err);
    }
  };

  const clearSeedData = async () => {
    setClearingSeed(true);
    try {
      await clearTelemetrySeed();
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["provider-slis"] }),
        queryClient.invalidateQueries({ queryKey: ["provider-incidents"] }),
        queryClient.invalidateQueries({ queryKey: ["provider-alerts"] }),
      ]);
    } catch (err) {
      console.error("clear telemetry seed failed", err);
    } finally {
      setClearingSeed(false);
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Provider Health"
        description="Live SLIs, degraded routes, and recent incidents across providers."
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={() => {
                void slisQuery.refetch();
                void incidentsQuery.refetch();
                void alertsQuery.refetch();
              }}
              loading={slisQuery.isFetching || incidentsQuery.isFetching || alertsQuery.isFetching}
            >
              <RefreshCcw className="h-4 w-4" />
            </Button>
            <Button variant="outline" onClick={() => void clearSeedData()} disabled={clearingSeed}>
              {clearingSeed ? "Clearing…" : "Clear seed"}
            </Button>
            <Button variant="secondary" onClick={() => void seedSampleData()}>
              Seed data
            </Button>
          </div>
        }
      />

      <Card>
        <CardContent className="flex flex-wrap gap-3 pt-6">
          <Select
            value={providerFilter ?? "all"}
            onValueChange={(val) => setProviderFilter(val === "all" ? undefined : val)}
          >
            <SelectTrigger className="w-[200px]">
              <SelectValue placeholder="Provider" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All providers</SelectItem>
              {providerOptions.map((p) => (
                <SelectItem key={p} value={p}>
                  {p}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={aliasFilter ?? "all"}
            onValueChange={(val) => setAliasFilter(val === "all" ? undefined : val)}
          >
            <SelectTrigger className="w-[220px]">
              <SelectValue placeholder="Model alias" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All aliases</SelectItem>
              {aliasOptions.map((alias) => (
                <SelectItem key={alias} value={alias}>
                  {alias}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Input
            placeholder="Filter by route"
            value={searchRoute}
            onChange={(e) => setSearchRoute(e.target.value)}
            className="w-[220px]"
          />
          <Select
            value={incidentStatus}
            onValueChange={(val) => setIncidentStatus((val as any) ?? "all")}
          >
            <SelectTrigger className="w-[200px]">
              <SelectValue placeholder="Incident status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All incidents</SelectItem>
              <SelectItem value="open">Open only</SelectItem>
              <SelectItem value="resolved">Resolved</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard
          title="Tracked routes"
          value={filteredSLIs.length}
          icon={<Gauge className="size-4" />}
        />
        <StatCard
          title="Degraded"
          value={totalDegraded}
          tone={totalDegraded > 0 ? "warn" : "ok"}
          icon={<AlertTriangle className="size-4" />}
        />
        <StatCard
          title="Active alerts"
          value={alertsQuery.data?.length ?? 0}
          tone={(alertsQuery.data?.length ?? 0) > 0 ? "warn" : "ok"}
          icon={<Siren className="size-4" />}
        />
      </div>

      {/* Charts removed per request; reinstating requires SLI data with meaningful points. */}

      <Card>
        <CardHeader>
          <CardTitle>Current SLIs</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {slisQuery.isLoading ? (
            <div className="space-y-2 p-4">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Provider</TableHead>
                  <TableHead>Alias</TableHead>
                  <TableHead>Route</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>P95</TableHead>
                  <TableHead>Error rate</TableHead>
                  <TableHead>Timeout rate</TableHead>
                  <TableHead>Requests</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredSLIs.map((sli) => {
                  const status = statusForSLI(sli);
                  return (
                    <TableRow key={`${sli.provider}-${sli.alias}-${sli.route ?? "default"}`}>
                      <TableCell className="font-medium">{sli.provider}</TableCell>
                      <TableCell>{sli.alias}</TableCell>
                      <TableCell>{sli.route || "default"}</TableCell>
                      <TableCell>
                        <StatusBadge label={status.label} tone={status.tone} />
                      </TableCell>
                      <TableCell>
                        {formatMs(sli.latency_p95_ms)}{" "}
                        <span className="text-muted-foreground text-xs">
                          / {formatMs(sli.thresholds.latency_p95_ms)}
                        </span>
                      </TableCell>
                      <TableCell>
                        {formatPercent(sli.error_rate)}{" "}
                        <span className="text-muted-foreground text-xs">
                          / {formatPercent(sli.thresholds.error_rate_threshold)}
                        </span>
                      </TableCell>
                      <TableCell>
                        {formatPercent(sli.timeout_rate)}{" "}
                        <span className="text-muted-foreground text-xs">
                          / {formatPercent(sli.thresholds.timeout_rate_threshold)}
                        </span>
                      </TableCell>
                      <TableCell>{sli.request_count.toLocaleString()}</TableCell>
                    </TableRow>
                  );
                })}
                {filteredSLIs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} className="text-center text-sm text-muted-foreground">
                      No telemetry yet for this filter.
                    </TableCell>
                  </TableRow>
                ) : null}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Active Alerts</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {alertsQuery.isLoading ? (
              <div className="p-4">
                <Skeleton className="h-10 w-full" />
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Provider</TableHead>
                    <TableHead>Alias</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Opened</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(alertsQuery.data ?? []).map((inc) => (
                    <TableRow key={inc.id}>
                      <TableCell className="font-medium">{inc.provider}</TableCell>
                      <TableCell>{inc.alias}</TableCell>
                      <TableCell className="uppercase text-xs font-semibold">
                        {inc.type.replaceAll("_", " ")}
                      </TableCell>
                      <TableCell>
                        {new Date(inc.opened_at).toLocaleString(undefined, {
                          hour12: false,
                        })}
                      </TableCell>
                    </TableRow>
                  ))}
                  {(alertsQuery.data?.length ?? 0) === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center text-sm text-muted-foreground">
                        No active alerts.
                      </TableCell>
                    </TableRow>
                  ) : null}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Recent Incidents</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {incidentsQuery.isLoading ? (
              <div className="p-4">
                <Skeleton className="h-10 w-full" />
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Provider</TableHead>
                    <TableHead>Alias</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Samples</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(incidentsQuery.data ?? []).slice(0, 10).map((inc) => (
                    <TableRow
                      key={inc.id}
                      className="cursor-pointer hover:bg-muted/60"
                      onClick={() => setSelectedIncident(inc)}
                    >
                      <TableCell className="font-medium">{inc.provider}</TableCell>
                      <TableCell>{inc.alias}</TableCell>
                      <TableCell className="uppercase text-xs font-semibold">
                        {inc.type.replaceAll("_", " ")}
                      </TableCell>
                      <TableCell>
                        <StatusBadge
                          label={inc.status === "open" ? "Open" : "Resolved"}
                          tone={inc.status === "open" ? "warn" : "ok"}
                        />
                      </TableCell>
                      <TableCell>{inc.request_count.toLocaleString()}</TableCell>
                    </TableRow>
                  ))}
                  {(incidentsQuery.data?.length ?? 0) === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
                        No incidents yet for this filter.
                      </TableCell>
                    </TableRow>
                  ) : null}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
      <Dialog open={Boolean(selectedIncident)} onOpenChange={() => setSelectedIncident(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Incident details</DialogTitle>
            <DialogDescription>
              Provider telemetry window summary with thresholds and sample error (if present).
            </DialogDescription>
          </DialogHeader>
          {selectedIncident ? (
            <div className="space-y-3 text-sm">
              <div className="flex flex-wrap items-center gap-3">
                <Badge variant="outline">{selectedIncident.provider}</Badge>
                <Badge variant="secondary">{selectedIncident.alias}</Badge>
                <Badge variant={selectedIncident.status === "open" ? "destructive" : "outline"}>
                  {selectedIncident.status === "open" ? "Open" : "Resolved"}
                </Badge>
              </div>
              <div className="grid gap-2 md:grid-cols-2">
                <DetailRow label="Type" value={selectedIncident.type.replaceAll("_", " ")} />
                <DetailRow
                  label="Opened"
                  value={new Date(selectedIncident.opened_at).toLocaleString()}
                />
                {selectedIncident.resolved_at ? (
                  <DetailRow
                    label="Resolved"
                    value={new Date(selectedIncident.resolved_at).toLocaleString()}
                  />
                ) : null}
                <DetailRow label="Window (s)" value={selectedIncident.window} />
                <DetailRow
                  label="Requests"
                  value={selectedIncident.request_count.toLocaleString()}
                />
                <DetailRow
                  label="Errors"
                  value={`${selectedIncident.error_count.toLocaleString()} (${formatPercent(selectedIncident.error_rate)})`}
                />
                <DetailRow
                  label="Timeouts"
                  value={`${selectedIncident.timeout_count.toLocaleString()} (${formatPercent(selectedIncident.timeout_rate)})`}
                />
                <DetailRow
                  label="Latency p95"
                  value={`${selectedIncident.latency_p95.toLocaleString()} ms (limit ${selectedIncident.thresholds.latency_p95_ms} ms)`}
                />
              </div>
              {selectedIncident.sample_error ? (
                <div className="rounded-md border bg-muted/40 p-3">
                  <div className="text-xs uppercase text-muted-foreground">Sample error</div>
                  <pre className="mt-1 whitespace-pre-wrap text-sm">
                    {selectedIncident.sample_error}
                  </pre>
                </div>
              ) : null}
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex justify-between rounded-md border px-3 py-2">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium text-right">{value}</span>
    </div>
  );
}


function StatCard({
  title,
  value,
  tone = "ok",
  icon,
}: {
  title: string;
  value: number | string;
  tone?: "ok" | "warn";
  icon?: ReactNode;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <span className="text-muted-foreground">{icon ?? <ShieldCheck className="size-4" />}</span>
      </CardHeader>
      <CardContent>
        <div
          className={`text-2xl font-semibold ${
            tone === "warn" ? "text-destructive" : "text-foreground"
          }`}
        >
          {value}
        </div>
      </CardContent>
    </Card>
  );
}

export default ProviderHealthPage;
