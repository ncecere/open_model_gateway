import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Key, RefreshCcw, Search } from "lucide-react";

import type {
  ApiKeyRecord,
  CreateApiKeyRequest,
  CreateApiKeyResponse,
} from "@/api/tenants";
import {
  createTenantApiKey,
  getTenantRateLimits,
  listAdminApiKeys,
  listPersonalTenants,
  listTenants,
  revokeTenantApiKey,
} from "@/api/tenants";
import { getBudgetDefaults } from "@/api/budgets";
import { getRateLimitDefaults } from "@/api/rate-limits";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useToast } from "@/hooks/use-toast";
import {
  AdminKeyTable,
  IssuedKeyDialog,
  RateLimitCard,
  formatScheduleLabel,
} from "@/features/api-keys";

const TENANTS_QUERY_KEY = ["tenants", "list"] as const;

const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

const dateFormatter = new Intl.DateTimeFormat(undefined, {
  year: "numeric",
  month: "short",
  day: "numeric",
});

export function KeysPage() {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const tenantsQuery = useQuery({
    queryKey: TENANTS_QUERY_KEY,
    queryFn: () => listTenants({ limit: 100 }),
  });

  const personalTenantsQuery = useQuery({
    queryKey: ["tenants", "personal"],
    queryFn: () => listPersonalTenants({ limit: 200 }),
  });

  const budgetDefaultsQuery = useQuery({
    queryKey: ["budget-defaults"],
    queryFn: getBudgetDefaults,
  });

  const rateLimitDefaultsQuery = useQuery({
    queryKey: ["rate-limit-defaults"],
    queryFn: getRateLimitDefaults,
  });

  const tenants = tenantsQuery.data?.tenants ?? [];
  const personalTenants = personalTenantsQuery.data?.personal_tenants ?? [];
  const [selectedTenantId, setSelectedTenantId] = useState<string | undefined>(
    undefined,
  );

  useEffect(() => {
    if (!selectedTenantId && tenants.length > 0) {
      setSelectedTenantId(tenants[0].id);
    }
  }, [selectedTenantId, tenants]);

  const keysQuery = useQuery({
    queryKey: ["admin-api-keys"],
    queryFn: () => listAdminApiKeys(),
  });

  const tenantRateLimitQuery = useQuery({
    queryKey: ["tenant-rate-limits", selectedTenantId],
    queryFn: () =>
      selectedTenantId ? getTenantRateLimits(selectedTenantId) : Promise.resolve(null),
    enabled: Boolean(selectedTenantId),
  });

  const createKeyMutation = useMutation({
    mutationFn: ({
      tenantId,
      payload,
    }: {
      tenantId: string;
      payload: CreateApiKeyRequest;
    }) => createTenantApiKey(tenantId, payload),
    onSuccess: (data) => {
      queryClient.invalidateQueries({
        queryKey: ["admin-api-keys"],
      });
      toast({
        title: "API key created",
        description: `${data.api_key.name} issued successfully`,
      });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to create key",
        description: "Please retry.",
      });
    },
  });

  const revokeKeyMutation = useMutation({
    mutationFn: ({
      tenantId,
      apiKeyId,
    }: {
      tenantId: string;
      apiKeyId: string;
    }) => revokeTenantApiKey(tenantId, apiKeyId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["admin-api-keys"],
      });
      toast({ title: "API key revoked" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to revoke key",
        description: "Try again in a moment.",
      });
    },
  });

  const [createOpen, setCreateOpen] = useState(false);
  const [issuedKey, setIssuedKey] = useState<CreateApiKeyResponse | null>(null);
  const [keyName, setKeyName] = useState("");
  const [budgetUsd, setBudgetUsd] = useState<string>("");
  const [warningThreshold, setWarningThreshold] = useState<string>("");
  const [requestsPerMinute, setRequestsPerMinute] = useState("");
  const [tokensPerMinute, setTokensPerMinute] = useState("");
  const [parallelRequests, setParallelRequests] = useState("");
  const [selectedKey, setSelectedKey] = useState<ApiKeyRecord | null>(null);
  const [pendingRevokeKey, setPendingRevokeKey] = useState<ApiKeyRecord | null>(
    null,
  );
  const [searchTerm, setSearchTerm] = useState("");
  const [issuerFilter, setIssuerFilter] = useState<"all" | "tenant" | "personal">(
    "all",
  );
  const [statusFilter, setStatusFilter] = useState<"all" | "active" | "revoked">(
    "all",
  );

  useEffect(() => {
    if (!createOpen) {
      setKeyName("");
      setBudgetUsd("");
      setWarningThreshold("");
      setRequestsPerMinute("");
      setTokensPerMinute("");
      setParallelRequests("");
    }
  }, [createOpen]);

  const handleCreateKey = async () => {
    if (!selectedTenantId) return;
    if (!keyName.trim()) {
      toast({ variant: "destructive", title: "Name is required" });
      return;
    }
    const payload: CreateApiKeyRequest = {
      name: keyName.trim(),
    };
    const parsedBudget = Number(budgetUsd);
    const parsedThreshold = Number(warningThreshold);
    const tenantBudgetLimit =
      tenantBudgetMap.get(selectedTenantId)?.limit ??
      budgetDefaults?.default_usd ??
      0;

    if (budgetUsd && Number.isFinite(parsedBudget)) {
      if (
        tenantBudgetLimit > 0 &&
        parsedBudget > tenantBudgetLimit
      ) {
        toast({
          variant: "destructive",
          title: `Budget exceeds tenant cap ($${tenantBudgetLimit.toFixed(2)})`,
        });
        return;
      }
      payload.quota = {
        budget_usd: parsedBudget,
        warning_threshold: Number.isFinite(parsedThreshold)
          ? parsedThreshold
          : undefined,
      };
    } else if (warningThreshold) {
      payload.quota = {
        warning_threshold: Number.isFinite(parsedThreshold)
          ? parsedThreshold
          : undefined,
      };
    }
    const trimmedRPM = requestsPerMinute.trim();
    const trimmedTPM = tokensPerMinute.trim();
    const trimmedParallel = parallelRequests.trim();
    const hasRateOverride =
      trimmedRPM.length > 0 || trimmedTPM.length > 0 || trimmedParallel.length > 0;
    let rpmValue: number | undefined;
    let tpmValue: number | undefined;
    let parallelValue: number | undefined;
    if (hasRateOverride) {
      rpmValue = Number.parseInt(trimmedRPM, 10);
      tpmValue = Number.parseInt(trimmedTPM, 10);
      parallelValue = Number.parseInt(trimmedParallel, 10);
      if (
        !Number.isFinite(rpmValue) ||
        !Number.isFinite(tpmValue) ||
        !Number.isFinite(parallelValue) ||
        (rpmValue as number) <= 0 ||
        (tpmValue as number) <= 0 ||
        (parallelValue as number) <= 0
      ) {
        toast({
          variant: "destructive",
          title: "Rate limits must be positive integers",
        });
        return;
      }
      const keyMaxRPM = defaultKeyRateLimit?.requests_per_minute ?? 0;
      const keyMaxTPM = defaultKeyRateLimit?.tokens_per_minute ?? 0;
      const keyMaxParallel = defaultKeyRateLimit?.parallel_requests ?? 0;
      const tenantMaxRPM = effectiveTenantRateLimit?.requests_per_minute ?? 0;
      const tenantMaxTPM = effectiveTenantRateLimit?.tokens_per_minute ?? 0;
      const tenantMaxParallel = effectiveTenantRateLimit?.parallel_requests ?? 0;
      if (keyMaxRPM > 0 && (rpmValue as number) > keyMaxRPM) {
        toast({
          variant: "destructive",
          title: `RPM exceeds key default (${keyMaxRPM})`,
        });
        return;
      }
      if (tenantMaxRPM > 0 && (rpmValue as number) > tenantMaxRPM) {
        toast({
          variant: "destructive",
          title: `RPM exceeds tenant cap (${tenantMaxRPM})`,
        });
        return;
      }
      if (keyMaxTPM > 0 && (tpmValue as number) > keyMaxTPM) {
        toast({
          variant: "destructive",
          title: `TPM exceeds key default (${keyMaxTPM})`,
        });
        return;
      }
      if (tenantMaxTPM > 0 && (tpmValue as number) > tenantMaxTPM) {
        toast({
          variant: "destructive",
          title: `TPM exceeds tenant cap (${tenantMaxTPM})`,
        });
        return;
      }
      if (keyMaxParallel > 0 && (parallelValue as number) > keyMaxParallel) {
        toast({
          variant: "destructive",
          title: `Parallel requests exceed key default (${keyMaxParallel})`,
        });
        return;
      }
      if (
        tenantMaxParallel > 0 &&
        (parallelValue as number) > tenantMaxParallel
      ) {
        toast({
          variant: "destructive",
          title: `Parallel requests exceed tenant cap (${tenantMaxParallel})`,
        });
        return;
      }
      payload.rate_limits = {
        requests_per_minute: rpmValue,
        tokens_per_minute: tpmValue,
        parallel_requests: parallelValue,
      };
    }

    try {
      const result = await createKeyMutation.mutateAsync({
        tenantId: selectedTenantId,
        payload,
      });
      setIssuedKey(result);
      setKeyName("");
      setBudgetUsd("");
      setWarningThreshold("");
      setRequestsPerMinute("");
      setTokensPerMinute("");
      setParallelRequests("");
      setCreateOpen(false);
    } catch (error) {
      console.error(error);
    }
  };

  const budgetDefaults = budgetDefaultsQuery.data;

  const tenantBudgetMap = useMemo(() => {
    const fallbackLimit = budgetDefaults?.default_usd ?? null;
    const fallbackWarn = budgetDefaults?.warning_threshold_perc ?? 0.8;
    const map = new Map<
      string,
      { limit: number | null; warning: number | null }
    >();
    tenants.forEach((tenant) => {
      map.set(tenant.id, {
        limit: tenant.budget_limit_usd ?? fallbackLimit,
        warning: tenant.warning_threshold ?? fallbackWarn,
      });
    });
    personalTenants.forEach((tenant) => {
      map.set(tenant.tenant_id, {
        limit: tenant.budget_limit_usd ?? fallbackLimit,
        warning: tenant.warning_threshold ?? fallbackWarn,
      });
    });
    return map;
  }, [tenants, personalTenants, budgetDefaults]);

  const resolveBudgetMeta = (key: ApiKeyRecord) => {
    const tenantBudget = tenantBudgetMap.get(key.tenant_id);
    const limit =
      key.quota?.budget_usd ??
      tenantBudget?.limit ??
      budgetDefaults?.default_usd ??
      0;
    const warning =
      key.quota?.warning_threshold ??
      tenantBudget?.warning ??
      budgetDefaults?.warning_threshold_perc ??
      0.8;
    return { limit, warning };
  };

  const selectedTenantBudgetLimit =
    (selectedTenantId && tenantBudgetMap.get(selectedTenantId)?.limit) ?? null;

  const keys: ApiKeyRecord[] = keysQuery.data?.api_keys ?? [];
  const filteredKeys = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return keys.filter((key) => {
      const matchesTerm =
        !term ||
        key.name.toLowerCase().includes(term) ||
        key.prefix.toLowerCase().includes(term) ||
        key.issuer?.label?.toLowerCase().includes(term) ||
        key.tenant_name?.toLowerCase().includes(term);
      const matchesIssuer =
        issuerFilter === "all" || key.issuer?.type === issuerFilter;
      const matchesStatus =
        statusFilter === "all" ||
        (statusFilter === "active" ? !key.revoked : key.revoked);
      return matchesTerm && matchesIssuer && matchesStatus;
    });
  }, [keys, searchTerm, issuerFilter, statusFilter]);

  const activeKeys = keys.filter((key) => !key.revoked);
  const revokedKeys = keys.filter((key) => key.revoked);

  const rateLimitDefaults = rateLimitDefaultsQuery.data;
  const defaultKeyRateLimit = useMemo(
    () =>
      rateLimitDefaults
        ? {
            requests_per_minute: rateLimitDefaults.requests_per_minute,
            tokens_per_minute: rateLimitDefaults.tokens_per_minute,
            parallel_requests: rateLimitDefaults.parallel_requests_key,
          }
        : null,
    [rateLimitDefaults],
  );
  const defaultTenantRateLimit = useMemo(
    () =>
      rateLimitDefaults
        ? {
            requests_per_minute: rateLimitDefaults.requests_per_minute,
            tokens_per_minute: rateLimitDefaults.tokens_per_minute,
            parallel_requests: rateLimitDefaults.parallel_requests_tenant,
          }
        : null,
    [rateLimitDefaults],
  );
  const tenantRateLimitOverride = tenantRateLimitQuery.data;
  const effectiveTenantRateLimit =
    tenantRateLimitOverride ?? defaultTenantRateLimit;

  const formatBudgetValue = (key: ApiKeyRecord) => {
    const { limit } = resolveBudgetMeta(key);
    return currencyFormatter.format(limit);
  };

  const formatWarningThresholdValue = (key: ApiKeyRecord) => {
    const { warning } = resolveBudgetMeta(key);
    return `${Math.round(warning * 100)}%`;
  };

  const handleCopy = async (value: string, label: string) => {
    try {
      await navigator.clipboard.writeText(value);
      toast({ title: `${label} copied to clipboard` });
    } catch (error) {
      console.error(error);
      toast({
        variant: "destructive",
        title: "Copy failed",
        description: "Copy manually instead.",
      });
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">API keys</h1>
          <p className="text-sm text-muted-foreground">
            Issue, rotate, and revoke tenant-scoped virtual keys with quota
            controls.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            onClick={() => keysQuery.refetch()}
            disabled={keysQuery.isFetching}
          >
            <RefreshCcw className="h-4 w-4" />
          </Button>
          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogTrigger asChild>
              <Button disabled={!selectedTenantId}>
                <Key className="mr-2 h-4 w-4" /> Generate key
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Generate API key</DialogTitle>
                <DialogDescription>
                  Issue a new key for the selected tenant. Secrets are shown
                  once—store them securely.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-2">
                <div className="space-y-2">
                  <Label htmlFor="tenant-select">Tenant</Label>
                  <Select
                    value={selectedTenantId}
                    onValueChange={(value) => setSelectedTenantId(value)}
                    disabled={createKeyMutation.isPending}
                  >
                    <SelectTrigger id="tenant-select">
                      <SelectValue placeholder="Select tenant" />
                    </SelectTrigger>
                    <SelectContent>
                      {tenants.map((tenant) => (
                        <SelectItem key={tenant.id} value={tenant.id}>
                          {tenant.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="key-name">Key name</Label>
                  <Input
                    id="key-name"
                    value={keyName}
                    onChange={(event) => setKeyName(event.target.value)}
                    placeholder="Production gateway"
                  />
                </div>
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="budget">Monthly budget (USD)</Label>
                    <Input
                      id="budget"
                      value={budgetUsd}
                      onChange={(event) => setBudgetUsd(event.target.value)}
                      placeholder={
                        selectedTenantBudgetLimit
                          ? `${selectedTenantBudgetLimit.toFixed(2)}`
                          : budgetDefaults?.default_usd
                            ? `${budgetDefaults.default_usd}`
                            : "Budget"
                      }
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="threshold">Warning threshold (0-1)</Label>
                    <Input
                      id="threshold"
                      value={warningThreshold}
                      onChange={(event) =>
                        setWarningThreshold(event.target.value)
                      }
                      placeholder="0.8"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label>Rate limit override (optional)</Label>
                    <p className="text-xs text-muted-foreground">
                      Max per key:{" "}
                      {defaultKeyRateLimit?.requests_per_minute ?? "—"} RPM /{" "}
                      {defaultKeyRateLimit?.tokens_per_minute ?? "—"} TPM /{" "}
                      {defaultKeyRateLimit?.parallel_requests ?? "—"} parallel
                    </p>
                  </div>
                  {effectiveTenantRateLimit ? (
                    <p className="text-xs text-muted-foreground">
                      Tenant cap: {effectiveTenantRateLimit.requests_per_minute} RPM ·{" "}
                      {effectiveTenantRateLimit.tokens_per_minute} TPM ·{" "}
                      {effectiveTenantRateLimit.parallel_requests} parallel
                    </p>
                  ) : null}
                  <div className="grid gap-4 md:grid-cols-3">
                    <Input
                      id="rpm"
                      value={requestsPerMinute}
                      onChange={(event) => setRequestsPerMinute(event.target.value)}
                      placeholder={
                        defaultKeyRateLimit?.requests_per_minute
                          ? `${defaultKeyRateLimit.requests_per_minute}`
                          : "RPM"
                      }
                      aria-label="Requests per minute"
                    />
                    <Input
                      id="tpm"
                      value={tokensPerMinute}
                      onChange={(event) => setTokensPerMinute(event.target.value)}
                      placeholder={
                        defaultKeyRateLimit?.tokens_per_minute
                          ? `${defaultKeyRateLimit.tokens_per_minute}`
                          : "Tokens per minute"
                      }
                      aria-label="Tokens per minute"
                    />
                    <Input
                      id="parallel"
                      value={parallelRequests}
                      onChange={(event) => setParallelRequests(event.target.value)}
                      placeholder={
                        defaultKeyRateLimit?.parallel_requests
                          ? `${defaultKeyRateLimit.parallel_requests}`
                          : "Parallel requests"
                      }
                      aria-label="Parallel requests"
                    />
                  </div>
                </div>
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setCreateOpen(false)}
                  disabled={createKeyMutation.isPending}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleCreateKey}
                  disabled={createKeyMutation.isPending || !selectedTenantId}
                >
                  {createKeyMutation.isPending ? "Issuing…" : "Generate"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>
      <Separator />

      <Card>
        <CardHeader className="flex flex-col gap-4 lg:flex-row lg:items-start">
          <div className="flex-1">
            <CardTitle>Key registry</CardTitle>
            <p className="text-sm text-muted-foreground">
              {activeKeys.length} active · {revokedKeys.length} revoked
            </p>
          </div>
          <div className="flex w-full flex-col gap-2 sm:flex-row sm:flex-1 sm:items-center">
            <div className="relative sm:w-64">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={searchTerm}
                onChange={(event) => setSearchTerm(event.target.value)}
                placeholder="Search name or issuer"
                className="pl-9"
              />
            </div>
            <Select
              value={issuerFilter}
              onValueChange={(value: "all" | "tenant" | "personal") =>
                setIssuerFilter(value)
              }
            >
              <SelectTrigger className="sm:w-44">
                <SelectValue placeholder="Issuer filter" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All issuers</SelectItem>
                <SelectItem value="tenant">Tenant</SelectItem>
                <SelectItem value="personal">Personal</SelectItem>
              </SelectContent>
            </Select>
            <Select
              value={statusFilter}
              onValueChange={(value: "all" | "active" | "revoked") =>
                setStatusFilter(value)
              }
            >
              <SelectTrigger className="sm:w-40">
                <SelectValue placeholder="Status filter" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="revoked">Revoked</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          <AdminKeyTable
            allKeys={keys}
            filteredKeys={filteredKeys}
            isLoading={keysQuery.isLoading}
            onViewDetails={setSelectedKey}
            onRequestRevoke={setPendingRevokeKey}
            revokeDisabled={revokeKeyMutation.isPending}
            formatBudgetValue={formatBudgetValue}
            formatWarningThresholdValue={formatWarningThresholdValue}
          />
        </CardContent>
      </Card>

      <AlertDialog
        open={Boolean(pendingRevokeKey)}
        onOpenChange={(open) => !open && setPendingRevokeKey(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke API key</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. Requests made with this key will
              immediately begin failing.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={revokeKeyMutation.isPending}
              onClick={() => {
                if (!pendingRevokeKey) return;
                revokeKeyMutation.mutate({
                  tenantId: pendingRevokeKey.tenant_id,
                  apiKeyId: pendingRevokeKey.id,
                });
                setPendingRevokeKey(null);
              }}
            >
              Revoke
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog
        open={Boolean(selectedKey)}
        onOpenChange={(open) => !open && setSelectedKey(null)}
      >
        <DialogContent className="sm:max-w-[640px]">
          {selectedKey ? (
            <>
              <DialogHeader>
                <DialogTitle>{selectedKey.name}</DialogTitle>
                <DialogDescription className="text-sm text-muted-foreground">
                  Issuer:{" "}
                  <span className="font-medium text-foreground">
                    {selectedKey.issuer?.label ??
                      selectedKey.tenant_name ??
                      "—"}
                  </span>
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-6 py-2">
                <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                  <span>
                    Issued{" "}
                    {dateFormatter.format(new Date(selectedKey.created_at))}
                  </span>
                  <Badge variant={selectedKey.revoked ? "destructive" : "secondary"}>
                    {selectedKey.revoked ? "revoked" : "active"}
                  </Badge>
                  <span>
                    Last used{" "}
                    {selectedKey.last_used_at
                      ? dateFormatter.format(new Date(selectedKey.last_used_at))
                      : "—"}
                  </span>
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-1">
                    <p className="text-sm font-medium text-muted-foreground">
                      Budget limit
                    </p>
                    <p className="text-lg font-semibold text-foreground">
                      {formatBudgetValue(selectedKey)}
                    </p>
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium text-muted-foreground">
                      Warning threshold
                    </p>
                    <p className="text-lg font-semibold text-foreground">
                      {formatWarningThresholdValue(selectedKey)}
                    </p>
                  </div>
                </div>
                <div className="space-y-2">
                  <p className="text-sm font-medium text-muted-foreground">
                    Budget reset schedule
                  </p>
                  <Badge variant="outline">
                    {formatScheduleLabel(selectedKey.budget_refresh_schedule)}
                  </Badge>
                </div>
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-medium text-muted-foreground">
                      Rate limits (RPM/TPM refresh every minute)
                    </p>
                  </div>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <RateLimitCard
                      title="Per-key limit"
                      details={selectedKey.rate_limits?.key}
                    />
                    <RateLimitCard
                      title="Tenant limit"
                      details={selectedKey.rate_limits?.tenant}
                    />
                  </div>
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setSelectedKey(null)}>
                  Close
                </Button>
              </DialogFooter>
            </>
          ) : null}
        </DialogContent>
      </Dialog>

      <IssuedKeyDialog
        issuedKey={
          issuedKey
            ? { token: issuedKey.token, secret: issuedKey.secret }
            : null
        }
        onCopy={handleCopy}
        onClose={() => setIssuedKey(null)}
      />
    </div>
  );
}
