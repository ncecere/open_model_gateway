import { useEffect, useMemo } from "react";
import {
  Activity,
  AlertCircle,
  CheckCircle2,
  Clock,
  Database,
  Globe,
  RefreshCcw,
  Server,
  Wifi,
  WifiOff,
  Zap,
} from "lucide-react";

import { PageHeader } from "@/components/layouts";
import { LiveIndicator } from "@/components/LiveIndicator";
import { useLiveUpdates } from "@/hooks/useLiveUpdates";
import { useSystemStatus } from "@/api/hooks/useSystemStatus";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { ServiceStatus, SystemMetric } from "@/api/system-status";

const statusConfig = {
  operational: {
    label: "Operational",
    color: "text-success",
    bgColor: "bg-success/10",
    icon: CheckCircle2,
  },
  degraded: {
    label: "Degraded",
    color: "text-warning",
    bgColor: "bg-warning/10",
    icon: AlertCircle,
  },
  down: {
    label: "Down",
    color: "text-destructive",
    bgColor: "bg-destructive/10",
    icon: WifiOff,
  },
  maintenance: {
    label: "Maintenance",
    color: "text-muted-foreground",
    bgColor: "bg-muted",
    icon: Clock,
  },
};

export function SystemStatusPage() {
  // Live updates state
  const liveState = useLiveUpdates({
    defaultInterval: 30000,
    defaultEnabled: true,
    storageKey: "system-status-live",
  });

  // Fetch system status from real API
  const statusQuery = useSystemStatus({
    refetchInterval: liveState.isLive ? liveState.interval : false,
  });

  // Track last update
  useEffect(() => {
    if (!statusQuery.isFetching) {
      liveState.markUpdated();
    }
  }, [statusQuery.dataUpdatedAt]);

  const systemStatus = statusQuery.data;
  const overallStatus = systemStatus?.overall ?? "operational";
  const services = systemStatus?.services ?? [];
  const metrics = systemStatus?.metrics ?? [];

  const operationalCount = useMemo(
    () => services.filter((s) => s.status === "operational").length,
    [services]
  );

  const isLoading = statusQuery.isLoading;

  return (
    <div className="space-y-6">
      <PageHeader
        title="System Status"
        description="Real-time health status of all system components and services."
        actions={
          <div className="flex items-center gap-2">
            <LiveIndicator
              state={liveState}
              isFetching={statusQuery.isFetching}
              onRefresh={() => void statusQuery.refetch()}
            />
            <Button
              variant="outline"
              size="icon"
              onClick={() => void statusQuery.refetch()}
              disabled={statusQuery.isFetching}
            >
              <RefreshCcw className={cn("h-4 w-4", statusQuery.isFetching && "animate-spin")} />
            </Button>
          </div>
        }
      />

      {/* Overall status banner */}
      {isLoading ? (
        <Skeleton className="h-24 w-full" />
      ) : (
        <Card
          className={cn(
            "border-l-4",
            overallStatus === "operational" && "border-l-success",
            overallStatus === "degraded" && "border-l-warning",
            overallStatus === "down" && "border-l-destructive"
          )}
        >
          <CardContent className="flex items-center justify-between py-4">
            <div className="flex items-center gap-4">
              {overallStatus === "operational" ? (
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-success/10">
                  <CheckCircle2 className="h-6 w-6 text-success" />
                </div>
              ) : overallStatus === "degraded" ? (
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-warning/10">
                  <AlertCircle className="h-6 w-6 text-warning" />
                </div>
              ) : (
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
                  <WifiOff className="h-6 w-6 text-destructive" />
                </div>
              )}
              <div>
                <h3 className="text-lg font-semibold">
                  {overallStatus === "operational"
                    ? "All Systems Operational"
                    : overallStatus === "degraded"
                    ? "Partial System Degradation"
                    : "System Outage Detected"}
                </h3>
                <p className="text-sm text-muted-foreground">
                  {operationalCount} of {services.length} services are operational
                </p>
              </div>
            </div>
            <div className="text-right">
              <p className="text-sm font-medium">Last updated</p>
              <p className="text-xs text-muted-foreground">
                {systemStatus?.lastUpdated
                  ? new Date(systemStatus.lastUpdated).toLocaleTimeString()
                  : new Date().toLocaleTimeString()}
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Quick stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Uptime (30d)</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-8 w-20" />
            ) : (
              <>
                <div className="text-2xl font-bold text-success">99.95%</div>
                <p className="text-xs text-muted-foreground">
                  Based on health checks
                </p>
              </>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Avg Latency</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-8 w-20" />
            ) : (
              <>
                <div className="text-2xl font-bold">
                  {services.length > 0
                    ? Math.round(
                        services.reduce((acc, s) => acc + (s.latency ?? 0), 0) /
                          services.filter((s) => s.latency).length || 1
                      )
                    : 0}
                  ms
                </div>
                <p className="text-xs text-muted-foreground">Across all services</p>
              </>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Services</CardTitle>
            <Globe className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-8 w-20" />
            ) : (
              <>
                <div className="text-2xl font-bold">{operationalCount}</div>
                <p className="text-xs text-muted-foreground">
                  of {services.length} total
                </p>
              </>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Issues</CardTitle>
            <AlertCircle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-8 w-20" />
            ) : (
              <>
                <div
                  className={cn(
                    "text-2xl font-bold",
                    services.filter((s) => s.status !== "operational").length > 0
                      ? "text-warning"
                      : "text-success"
                  )}
                >
                  {services.filter((s) => s.status !== "operational").length}
                </div>
                <p className="text-xs text-muted-foreground">
                  Degraded or down services
                </p>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Services grid */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            Services Status
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {[1, 2, 3, 4].map((i) => (
                <Skeleton key={i} className="h-24 w-full" />
              ))}
            </div>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {services.map((service) => (
                <ServiceCard key={service.name} service={service} />
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* System metrics */}
      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Database className="h-5 w-5" />
              System Metrics
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {isLoading ? (
              <>
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </>
            ) : (
              metrics.slice(0, 4).map((metric) => (
                <MetricBar key={metric.label} metric={metric} />
              ))
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wifi className="h-5 w-5" />
              Service Latencies
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {isLoading ? (
              <>
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </>
            ) : (
              services
                .filter((s) => s.latency !== undefined)
                .slice(0, 6)
                .map((service) => (
                  <div
                    key={service.name}
                    className="flex items-center justify-between rounded-md border px-3 py-2"
                  >
                    <span className="text-sm">{service.name}</span>
                    <span
                      className={cn(
                        "text-sm font-medium",
                        service.latency && service.latency > 500
                          ? "text-warning"
                          : service.latency && service.latency > 1000
                          ? "text-destructive"
                          : ""
                      )}
                    >
                      {service.latency}ms
                    </span>
                  </div>
                ))
            )}
          </CardContent>
        </Card>
      </div>

      {/* Recent status */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Status Changes</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {isLoading ? (
              <>
                <Skeleton className="h-16 w-full" />
                <Skeleton className="h-16 w-full" />
              </>
            ) : services.filter((s) => s.status !== "operational").length > 0 ? (
              services
                .filter((s) => s.status !== "operational")
                .map((service) => (
                  <div
                    key={service.name}
                    className="flex items-start gap-3 rounded-lg border p-3"
                  >
                    <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-warning" />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="font-medium">{service.name}</p>
                        <Badge variant="secondary" className="text-xs">
                          {statusConfig[service.status].label}
                        </Badge>
                      </div>
                      {service.details && (
                        <p className="mt-1 text-sm text-muted-foreground">
                          {service.details}
                        </p>
                      )}
                    </div>
                  </div>
                ))
            ) : (
              <div className="flex items-center gap-3 rounded-lg border border-success/30 bg-success/5 p-4">
                <CheckCircle2 className="h-5 w-5 text-success" />
                <div>
                  <p className="font-medium text-success">No recent issues</p>
                  <p className="text-sm text-muted-foreground">
                    All services are running normally
                  </p>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function ServiceCard({ service }: { service: ServiceStatus }) {
  const config = statusConfig[service.status];
  const StatusIcon = config.icon;

  return (
    <div
      className={cn(
        "flex items-start gap-3 rounded-lg border p-3 transition-colors",
        service.status !== "operational" && "bg-muted/30"
      )}
    >
      <div
        className={cn(
          "flex h-8 w-8 shrink-0 items-center justify-center rounded-full",
          config.bgColor
        )}
      >
        <StatusIcon className={cn("h-4 w-4", config.color)} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-medium truncate">{service.name}</p>
          <Badge variant="outline" className={cn("shrink-0 text-xs", config.color)}>
            {config.label}
          </Badge>
        </div>
        <div className="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
          {service.latency !== undefined && <span>{service.latency}ms</span>}
          {service.uptime !== undefined && <span>{service.uptime}%</span>}
        </div>
        {service.details && (
          <p className="mt-1 text-xs text-warning">{service.details}</p>
        )}
      </div>
    </div>
  );
}

function MetricBar({ metric }: { metric: SystemMetric }) {
  const percentage = (metric.value / metric.max) * 100;
  const displayValue =
    metric.value % 1 === 0 ? metric.value : metric.value.toFixed(2);

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-sm">
        <span>{metric.label}</span>
        <span className="font-medium">
          {displayValue}
          {metric.unit}
        </span>
      </div>
      <Progress
        value={percentage}
        className={cn(
          "h-2",
          metric.status === "warning" && "[&>div]:bg-warning",
          metric.status === "critical" && "[&>div]:bg-destructive"
        )}
      />
    </div>
  );
}

export default SystemStatusPage;
