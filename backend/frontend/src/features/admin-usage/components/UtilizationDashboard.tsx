import { useMemo } from "react";
import { Activity, Key, Users, Layers } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { formatUsageUSD } from "@/features/usage";
import { cn } from "@/lib/utils";

export interface UtilizationMetrics {
  // API Key utilization
  totalApiKeys: number;
  activeApiKeys: number;
  keysUsedRecently: number;

  // Tenant utilization
  totalTenants: number;
  activeTenants: number;

  // User utilization
  totalUsers: number;
  activeUsers: number;

  // Cost metrics
  totalSpend: number;
  averageSpendPerTenant?: number;
  averageSpendPerUser?: number;

  // Provider distribution
  providerBreakdown?: Array<{
    provider: string;
    requests: number;
    spend: number;
  }>;
}

export interface UtilizationDashboardProps {
  metrics: UtilizationMetrics | null;
  loading?: boolean;
  className?: string;
}

interface UtilizationCardProps {
  title: string;
  icon: typeof Activity;
  active: number;
  total: number;
  label: string;
  loading?: boolean;
}

function UtilizationCard({
  title,
  icon: Icon,
  active,
  total,
  label,
  loading,
}: UtilizationCardProps) {
  const percentage = total > 0 ? (active / total) * 100 : 0;
  const isHealthy = percentage >= 50;

  if (loading) {
    return (
      <Card>
        <CardHeader className="pb-2">
          <Skeleton className="h-4 w-24" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-8 w-16 mb-2" />
          <Skeleton className="h-2 w-full mb-2" />
          <Skeleton className="h-3 w-20" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="animate-fade-in-up">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold mb-2">
          {active}{" "}
          <span className="text-sm font-normal text-muted-foreground">
            / {total}
          </span>
        </div>
        <Progress
          value={percentage}
          className="h-1.5 mb-2"
          indicatorClassName={cn(isHealthy ? "bg-success" : "bg-warning")}
        />
        <p className="text-xs text-muted-foreground">
          {percentage.toFixed(0)}% {label}
        </p>
      </CardContent>
    </Card>
  );
}

export function UtilizationDashboard({
  metrics,
  loading,
  className,
}: UtilizationDashboardProps) {
  const providerData = useMemo(() => {
    if (!metrics?.providerBreakdown?.length) return [];

    const totalRequests = metrics.providerBreakdown.reduce(
      (sum, p) => sum + p.requests,
      0
    );

    return metrics.providerBreakdown
      .sort((a, b) => b.requests - a.requests)
      .map((p) => ({
        ...p,
        percentage: totalRequests > 0 ? (p.requests / totalRequests) * 100 : 0,
      }));
  }, [metrics?.providerBreakdown]);

  return (
    <div className={cn("space-y-6", className)}>
      {/* Utilization Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <UtilizationCard
          title="API Keys"
          icon={Key}
          active={metrics?.activeApiKeys ?? 0}
          total={metrics?.totalApiKeys ?? 0}
          label="active"
          loading={loading}
        />
        <UtilizationCard
          title="Keys Used Recently"
          icon={Activity}
          active={metrics?.keysUsedRecently ?? 0}
          total={metrics?.totalApiKeys ?? 0}
          label="used in period"
          loading={loading}
        />
        <UtilizationCard
          title="Tenants"
          icon={Layers}
          active={metrics?.activeTenants ?? 0}
          total={metrics?.totalTenants ?? 0}
          label="active"
          loading={loading}
        />
        <UtilizationCard
          title="Users"
          icon={Users}
          active={metrics?.activeUsers ?? 0}
          total={metrics?.totalUsers ?? 0}
          label="active"
          loading={loading}
        />
      </div>

      {/* Cost Efficiency */}
      {!loading && metrics && (
        <Card className="animate-fade-in-up">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Cost Efficiency</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3">
              <div className="space-y-1">
                <p className="text-xs text-muted-foreground uppercase tracking-wider">
                  Total Platform Spend
                </p>
                <p className="text-2xl font-bold">
                  {formatUsageUSD(metrics.totalSpend)}
                </p>
              </div>
              {metrics.averageSpendPerTenant !== undefined && (
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground uppercase tracking-wider">
                    Avg. per Tenant
                  </p>
                  <p className="text-2xl font-bold">
                    {formatUsageUSD(metrics.averageSpendPerTenant)}
                  </p>
                </div>
              )}
              {metrics.averageSpendPerUser !== undefined && (
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground uppercase tracking-wider">
                    Avg. per User
                  </p>
                  <p className="text-2xl font-bold">
                    {formatUsageUSD(metrics.averageSpendPerUser)}
                  </p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Provider Distribution */}
      {!loading && providerData.length > 0 && (
        <Card className="animate-fade-in-up">
          <CardHeader>
            <CardTitle className="text-sm font-medium">
              Provider Distribution
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {providerData.map((provider) => (
                <div key={provider.provider} className="space-y-1">
                  <div className="flex items-center justify-between text-sm">
                    <span className="font-medium capitalize">
                      {provider.provider}
                    </span>
                    <span className="text-muted-foreground">
                      {provider.requests.toLocaleString()} requests ·{" "}
                      {formatUsageUSD(provider.spend)}
                    </span>
                  </div>
                  <Progress
                    value={provider.percentage}
                    className="h-1.5"
                    indicatorClassName="bg-chart-1"
                  />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
