import { useState, useMemo, useEffect } from "react";
import {
  Activity,
  AlertCircle,
  Calendar,
  ChevronDown,
  ChevronUp,
  Download,
  Filter,
  Key,
  Shield,
  User,
  type LucideIcon,
} from "lucide-react";

import { PageHeader } from "@/components/layouts";
import { LiveIndicator } from "@/components/LiveIndicator";
import { useLiveUpdates } from "@/hooks/useLiveUpdates";
import { useAuditLogs, useAuditLogStats } from "@/api/hooks/useAudit";
import type { AuditLogEntry as ApiAuditLogEntry } from "@/api/audit";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/EmptyState";
import { cn } from "@/lib/utils";

// UI types for audit log entries (enriched from API)
interface AuditLogEntry {
  id: string;
  timestamp: Date;
  action: string;
  category: "auth" | "api_key" | "tenant" | "user" | "model" | "system";
  actor: {
    id: string;
    type: "user" | "system" | "api_key";
    email?: string;
    name?: string;
  };
  target?: {
    type: string;
    id: string;
    name?: string;
  };
  metadata?: Record<string, unknown>;
  success: boolean;
  error_message?: string;
}

const categoryIcons: Record<AuditLogEntry["category"], LucideIcon> = {
  auth: Shield,
  api_key: Key,
  tenant: Activity,
  user: User,
  model: Activity,
  system: AlertCircle,
};

const categoryColors: Record<AuditLogEntry["category"], string> = {
  auth: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  api_key: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
  tenant: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  user: "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400",
  model: "bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-400",
  system: "bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-400",
};

// Human-readable action descriptions
const actionDescriptions: Record<string, string> = {
  // Tenant actions
  "tenant.create": "Created tenant",
  "tenant.update": "Updated tenant",
  "tenant.delete": "Deleted tenant",
  "tenant.models.set": "Updated tenant models",
  "tenant.models.add": "Added model to tenant",
  "tenant.models.remove": "Removed model from tenant",
  "tenant.models.clear": "Cleared tenant models",

  // Membership actions
  "membership.upsert": "Updated membership",
  "membership.remove": "Removed member",
  "tenant.memberships.view": "Viewed memberships",

  // API key actions
  "tenant.api_key.create": "Created API key",
  "tenant.api_key.revoke": "Revoked API key",
  "admin_api_key.create": "Created admin API key",
  "admin_api_key.revoke": "Revoked admin API key",

  // User actions
  "user.create": "Created user",
  "user.update": "Updated user",
  "user.delete": "Deleted user",

  // Model actions
  "model.create": "Added model to catalog",
  "model.update": "Updated model",
  "model.delete": "Removed model from catalog",
  "default_model.add": "Added default model",
  "default_model.remove": "Removed default model",

  // Budget actions
  "budget.create": "Created budget",
  "budget.update": "Updated budget",
  "budget.delete": "Deleted budget",
  "tenant.budget.set": "Set tenant budget",
  "tenant.budget.update": "Updated tenant budget",
  "tenant.budget.delete": "Removed tenant budget",

  // Rate limit actions
  "rate_limit.create": "Created rate limit",
  "rate_limit.update": "Updated rate limit",
  "rate_limit.delete": "Deleted rate limit",
  "tenant.rate_limit.set": "Set tenant rate limit",
  "tenant.rate_limit.delete": "Removed tenant rate limit",

  // Settings actions
  "settings.update": "Updated settings",

  // File actions
  "file.delete": "Deleted file",

  // Batch actions
  "batch.cancel": "Cancelled batch job",

  // Auth actions
  "auth.login": "Logged in",
  "auth.logout": "Logged out",
  "auth.login.failed": "Failed login attempt",
};

// Get human-readable action description
function getActionDescription(action: string, _metadata?: Record<string, unknown>): string {
  // Check for exact match first
  if (actionDescriptions[action]) {
    return actionDescriptions[action];
  }

  // Try to find partial match (for variations)
  for (const [key, desc] of Object.entries(actionDescriptions)) {
    if (action.startsWith(key) || action.endsWith(key)) {
      return desc;
    }
  }

  // Fallback: convert action to readable format
  return action
    .split(".")
    .map((part) => part.replace(/_/g, " "))
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

// Get target display name from metadata or resource ID
function getTargetDisplay(
  resourceType: string,
  resourceId: string,
  metadata?: Record<string, unknown>
): string {
  // Try to get a friendly name from metadata
  if (metadata) {
    // Common metadata fields that might contain names
    const nameFields = ["name", "alias", "email", "tenant_name", "user_email", "model_alias", "label"];
    for (const field of nameFields) {
      if (metadata[field] && typeof metadata[field] === "string") {
        return metadata[field] as string;
      }
    }

    // For model lists, show count or first few
    if (metadata["models"] && Array.isArray(metadata["models"])) {
      const models = metadata["models"] as string[];
      if (models.length <= 2) {
        return models.join(", ");
      }
      return `${models.slice(0, 2).join(", ")} +${models.length - 2} more`;
    }

    // For role updates
    if (metadata["role"] && typeof metadata["role"] === "string") {
      return `Role: ${metadata["role"]}`;
    }
  }

  // For short IDs (like model aliases), use as-is
  if (resourceId && !resourceId.includes("-") && resourceId.length < 40) {
    return resourceId;
  }

  // For UUIDs, show truncated version with resource type
  if (resourceId && resourceId.length >= 32) {
    const shortId = resourceId.substring(0, 8);
    const typeLabel = formatResourceType(resourceType);
    return `${typeLabel} ${shortId}...`;
  }

  return resourceId || "—";
}

// Format resource type for display
function formatResourceType(resourceType: string): string {
  const typeNames: Record<string, string> = {
    tenant: "Tenant",
    user: "User",
    model: "Model",
    apikey: "API Key",
    admin_api_key: "Admin Key",
    file: "File",
    batch: "Batch",
    budget: "Budget",
    rate_limit: "Rate Limit",
  };
  return typeNames[resourceType.toLowerCase()] ?? resourceType.replace(/_/g, " ");
}

// Format actor ID - shorten UUIDs for display
function formatActorId(id: string): string {
  // If it looks like a UUID, shorten it
  if (id && id.length >= 32 && id.includes("-")) {
    return `User ${id.substring(0, 8)}...`;
  }
  return id;
}

// Format category for display
function formatCategory(category: string): string {
  const names: Record<string, string> = {
    auth: "Authentication",
    api_key: "API Keys",
    tenant: "Tenants",
    user: "Users",
    model: "Models",
    system: "System",
  };
  return names[category] ?? category;
}

// Map resource type from API to UI category
function getCategory(resource: string, action: string): AuditLogEntry["category"] {
  const lowerResource = resource.toLowerCase();
  const lowerAction = action.toLowerCase();

  if (lowerAction.includes("login") || lowerAction.includes("auth") || lowerResource === "auth") {
    return "auth";
  }
  if (lowerResource.includes("key") || lowerResource === "apikey") {
    return "api_key";
  }
  if (lowerResource === "tenant") {
    return "tenant";
  }
  if (lowerResource === "user") {
    return "user";
  }
  if (lowerResource === "model") {
    return "model";
  }
  return "system";
}

// Transform API response to UI format
function transformAuditLog(apiLog: ApiAuditLogEntry): AuditLogEntry {
  const category = getCategory(apiLog.resource, apiLog.action);
  const isSystemAction = !apiLog.user_id || apiLog.action.toLowerCase().includes("system");
  const isFailed = apiLog.action.toLowerCase().includes("failed") || apiLog.action.toLowerCase().includes("error");
  const metadata = apiLog.metadata as Record<string, unknown> | undefined;

  // Get actor email from metadata if available
  const actorEmail = metadata?.actor_email as string | undefined;

  return {
    id: apiLog.id,
    timestamp: new Date(apiLog.created_at),
    action: apiLog.action,
    category,
    actor: {
      id: apiLog.user_id ?? "system",
      type: isSystemAction ? "system" : "user",
      email: actorEmail,
    },
    target: apiLog.resource_id ? {
      type: apiLog.resource,
      id: apiLog.resource_id,
      name: getTargetDisplay(apiLog.resource, apiLog.resource_id, metadata),
    } : undefined,
    metadata,
    success: !isFailed,
    error_message: isFailed ? metadata?.error as string : undefined,
  };
}

export function AuditLogPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [categoryFilter, setCategoryFilter] = useState<string>("all");
  const [successFilter, setSuccessFilter] = useState<string>("all");
  const [selectedLog, setSelectedLog] = useState<AuditLogEntry | null>(null);
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");

  // Live updates state
  const liveState = useLiveUpdates({
    defaultInterval: 30000,
    defaultEnabled: true,
    storageKey: "audit-log-live",
  });

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchQuery), 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Map UI category filter to API resource filter
  const resourceFilter = useMemo(() => {
    if (categoryFilter === "all") return undefined;
    const mapping: Record<string, string> = {
      auth: "auth",
      api_key: "apikey",
      tenant: "tenant",
      user: "user",
      model: "model",
      system: "system",
    };
    return mapping[categoryFilter];
  }, [categoryFilter]);

  // Fetch audit logs from API
  const logsQuery = useAuditLogs(
    {
      limit: 100,
      resource: resourceFilter,
    },
    {
      refetchInterval: liveState.isLive ? liveState.interval : false,
    }
  );

  // Fetch stats
  const statsQuery = useAuditLogStats();

  // Track last update
  useEffect(() => {
    if (!logsQuery.isFetching) {
      liveState.markUpdated();
    }
  }, [logsQuery.dataUpdatedAt]);

  // Transform and filter logs
  const logs = useMemo(() => {
    const apiLogs = logsQuery.data?.logs ?? [];
    return apiLogs.map(transformAuditLog);
  }, [logsQuery.data]);

  const filteredLogs = useMemo(() => {
    let result = [...logs];

    // Search filter (client-side for now)
    if (debouncedSearch.trim()) {
      const query = debouncedSearch.toLowerCase();
      result = result.filter(
        (log) =>
          log.action.toLowerCase().includes(query) ||
          log.actor.email?.toLowerCase().includes(query) ||
          log.target?.name?.toLowerCase().includes(query) ||
          log.target?.id?.toLowerCase().includes(query)
      );
    }

    // Success filter (client-side)
    if (successFilter !== "all") {
      result = result.filter(
        (log) => log.success === (successFilter === "success")
      );
    }

    // Sort
    result.sort((a, b) => {
      const diff = a.timestamp.getTime() - b.timestamp.getTime();
      return sortOrder === "asc" ? diff : -diff;
    });

    return result;
  }, [logs, debouncedSearch, successFilter, sortOrder]);

  const formatTimestamp = (date: Date) => {
    return new Intl.DateTimeFormat("en-US", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    }).format(date);
  };

  const formatActionDisplay = (action: string, metadata?: Record<string, unknown>) => {
    return getActionDescription(action, metadata);
  };

  const isLoading = logsQuery.isLoading;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Audit Log"
        description="Track and review all administrative actions and security events."
        actions={
          <div className="flex items-center gap-2">
            <LiveIndicator
              state={liveState}
              isFetching={logsQuery.isFetching}
              onRefresh={() => void logsQuery.refetch()}
            />
            <Button variant="outline" size="sm">
              <Download className="mr-2 h-4 w-4" />
              Export
            </Button>
          </div>
        }
      />

      {/* Filters */}
      <Card>
        <CardContent className="flex flex-wrap items-center gap-3 pt-6">
          <div className="flex items-center gap-2">
            <Filter className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm font-medium">Filters:</span>
          </div>
          <Input
            placeholder="Search actions, users, IPs..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-[250px]"
          />
          <Select value={categoryFilter} onValueChange={setCategoryFilter}>
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder="Category" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All categories</SelectItem>
              <SelectItem value="auth">Authentication</SelectItem>
              <SelectItem value="api_key">API Keys</SelectItem>
              <SelectItem value="tenant">Tenants</SelectItem>
              <SelectItem value="user">Users</SelectItem>
              <SelectItem value="model">Models</SelectItem>
              <SelectItem value="system">System</SelectItem>
            </SelectContent>
          </Select>
          <Select value={successFilter} onValueChange={setSuccessFilter}>
            <SelectTrigger className="w-[130px]">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              <SelectItem value="success">Success</SelectItem>
              <SelectItem value="failed">Failed</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Events</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {statsQuery.isLoading ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <div className="text-2xl font-bold">{statsQuery.data?.total ?? logs.length}</div>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Auth Events</CardTitle>
            <Shield className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {statsQuery.isLoading ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <div className="text-2xl font-bold">
                {statsQuery.data?.auth_events ?? logs.filter((l) => l.category === "auth").length}
              </div>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Failed Actions</CardTitle>
            <AlertCircle className="h-4 w-4 text-destructive" />
          </CardHeader>
          <CardContent>
            {statsQuery.isLoading ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <div className="text-2xl font-bold text-destructive">
                {statsQuery.data?.failed_events ?? logs.filter((l) => !l.success).length}
              </div>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Unique Actors</CardTitle>
            <User className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {statsQuery.isLoading ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <div className="text-2xl font-bold">
                {statsQuery.data?.unique_actors ?? new Set(logs.map((l) => l.actor.id)).size}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Log table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>Recent Activity</span>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setSortOrder((o) => (o === "asc" ? "desc" : "asc"))}
              className="gap-1 text-muted-foreground"
            >
              <Calendar className="h-4 w-4" />
              {sortOrder === "desc" ? (
                <ChevronDown className="h-3 w-3" />
              ) : (
                <ChevronUp className="h-3 w-3" />
              )}
            </Button>
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="space-y-2 p-4">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : filteredLogs.length === 0 ? (
            <EmptyState
              illustration="no-results"
              title="No audit logs found"
              description="Try adjusting your filters or check back later for activity."
              size="sm"
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[180px]">Timestamp</TableHead>
                  <TableHead>Action</TableHead>
                  <TableHead>Actor</TableHead>
                  <TableHead>Target</TableHead>
                  <TableHead className="w-[100px]">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredLogs.map((log) => {
                  const CategoryIcon = categoryIcons[log.category];
                  return (
                    <TableRow
                      key={log.id}
                      className="cursor-pointer hover:bg-muted/50"
                      onClick={() => setSelectedLog(log)}
                    >
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {formatTimestamp(log.timestamp)}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <span
                            className={cn(
                              "flex h-6 w-6 items-center justify-center rounded",
                              categoryColors[log.category]
                            )}
                          >
                            <CategoryIcon className="h-3.5 w-3.5" />
                          </span>
                          <span className="font-medium">
                            {formatActionDisplay(log.action, log.metadata)}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1.5">
                          {log.actor.type === "system" ? (
                            <Badge variant="outline" className="text-xs">
                              System
                            </Badge>
                          ) : (
                            <span className="text-sm font-mono">
                              {log.actor.email ?? formatActorId(log.actor.id)}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {log.target?.name ?? "—"}
                      </TableCell>
                      <TableCell>
                        {log.success ? (
                          <Badge variant="default" className="bg-success/20 text-success hover:bg-success/30">
                            Success
                          </Badge>
                        ) : (
                          <Badge variant="destructive">Failed</Badge>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Detail dialog */}
      <Dialog open={!!selectedLog} onOpenChange={() => setSelectedLog(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Audit Log Details</DialogTitle>
            <DialogDescription>
              Full details of this audit event including metadata.
            </DialogDescription>
          </DialogHeader>
          {selectedLog && (
            <div className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <DetailRow label="Timestamp" value={selectedLog.timestamp.toISOString()} />
                <DetailRow label="Action" value={formatActionDisplay(selectedLog.action, selectedLog.metadata)} />
                <DetailRow label="Category" value={formatCategory(selectedLog.category)} />
                <DetailRow label="Status" value={selectedLog.success ? "Success" : "Failed"} />
                <DetailRow label="Actor" value={selectedLog.actor.email ?? formatActorId(selectedLog.actor.id)} />
                <DetailRow label="Actor Type" value={selectedLog.actor.type === "user" ? "User" : selectedLog.actor.type === "system" ? "System" : "API Key"} />
                {selectedLog.target && (
                  <>
                    <DetailRow label="Target Type" value={selectedLog.target.type} />
                    <DetailRow label="Target" value={selectedLog.target.name ?? selectedLog.target.id} />
                  </>
                )}
              </div>
              {selectedLog.error_message && (
                <div className="rounded-md border border-destructive/20 bg-destructive/5 p-3">
                  <p className="text-sm font-medium text-destructive">Error</p>
                  <p className="text-sm text-destructive/80">{selectedLog.error_message}</p>
                </div>
              )}
              {selectedLog.metadata && Object.keys(selectedLog.metadata).length > 0 && (
                <div>
                  <p className="mb-2 text-sm font-medium text-muted-foreground">Metadata</p>
                  <pre className="rounded-md bg-muted p-3 text-xs">
                    {JSON.stringify(selectedLog.metadata, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between rounded-md border px-3 py-2">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="text-sm font-medium">{value}</span>
    </div>
  );
}

export default AuditLogPage;
