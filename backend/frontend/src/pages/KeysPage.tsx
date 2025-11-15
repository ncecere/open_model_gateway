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
  listAdminApiKeys,
  listPersonalTenants,
  listTenants,
  revokeTenantApiKey,
} from "@/api/tenants";
import { getBudgetDefaults } from "@/api/budgets";
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
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
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
import {
  deleteApiKeyGuardrails,
  getApiKeyGuardrails,
  upsertApiKeyGuardrails,
  type GuardrailConfig,
} from "@/api/guardrails";
import { formatKeywordInput, parseKeywordInput } from "@/utils/guardrails";

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
  const [keyGuardrailLoading, setKeyGuardrailLoading] = useState(false);
  const [keyGuardrailSaving, setKeyGuardrailSaving] = useState(false);
  const [keyGuardrailOverride, setKeyGuardrailOverride] = useState(false);
  const [keyGuardrailHadOverride, setKeyGuardrailHadOverride] = useState(false);
  const [keyGuardrailEnabled, setKeyGuardrailEnabled] = useState(true);
  const [keyGuardrailPromptKeywords, setKeyGuardrailPromptKeywords] =
    useState("");
  const [keyGuardrailResponseKeywords, setKeyGuardrailResponseKeywords] =
    useState("");
  const [keyGuardrailModerationEnabled, setKeyGuardrailModerationEnabled] =
    useState(false);
  const [keyGuardrailModerationProvider, setKeyGuardrailModerationProvider] =
    useState("keyword");
  const [keyGuardrailModerationAction, setKeyGuardrailModerationAction] =
    useState("block");
  const [keyGuardrailWebhookURL, setKeyGuardrailWebhookURL] = useState("");
  const [keyGuardrailWebhookHeader, setKeyGuardrailWebhookHeader] =
    useState("");
  const [keyGuardrailWebhookValue, setKeyGuardrailWebhookValue] = useState("");
  const [keyGuardrailWebhookTimeout, setKeyGuardrailWebhookTimeout] =
    useState("5");

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
    if (budgetUsd && Number.isFinite(parsedBudget)) {
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

    try {
      const result = await createKeyMutation.mutateAsync({
        tenantId: selectedTenantId,
        payload,
      });
      setIssuedKey(result);
      setKeyName("");
      setBudgetUsd("");
      setWarningThreshold("");
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

  const buildKeyGuardrailPayload = (): GuardrailConfig => {
    const promptKeywords = parseKeywordInput(keyGuardrailPromptKeywords);
    const responseKeywords = parseKeywordInput(keyGuardrailResponseKeywords);
    const moderationProvider = keyGuardrailModerationProvider.trim();

    const payload: GuardrailConfig = {
      enabled: keyGuardrailEnabled,
      prompt: { blocked_keywords: promptKeywords },
      response: { blocked_keywords: responseKeywords },
    };

    if (
      keyGuardrailModerationEnabled ||
      moderationProvider ||
      keyGuardrailModerationAction
    ) {
      const timeoutValue = Number.parseInt(
        keyGuardrailWebhookTimeout.trim(),
        10,
      );
      payload.moderation = {
        enabled: keyGuardrailModerationEnabled,
        provider: moderationProvider || undefined,
        action: keyGuardrailModerationAction,
        webhook_url: keyGuardrailWebhookURL.trim() || undefined,
        webhook_auth_header:
          keyGuardrailWebhookHeader.trim() || undefined,
        webhook_auth_value:
          keyGuardrailWebhookValue.trim() || undefined,
        timeout_seconds:
          Number.isFinite(timeoutValue) && timeoutValue > 0
            ? timeoutValue
            : undefined,
      };
    }

    return payload;
  };

  const handleSaveKeyGuardrails = async () => {
    if (!selectedKey) return;
    setKeyGuardrailSaving(true);
    try {
      if (keyGuardrailOverride) {
        const payload = buildKeyGuardrailPayload();
        await upsertApiKeyGuardrails(
          selectedKey.tenant_id,
          selectedKey.id,
          payload,
        );
        setKeyGuardrailHadOverride(true);
        toast({ title: "Guardrail policy updated" });
      } else if (keyGuardrailHadOverride) {
        await deleteApiKeyGuardrails(selectedKey.tenant_id, selectedKey.id);
        setKeyGuardrailHadOverride(false);
        toast({ title: "Guardrail policy removed" });
      } else {
        toast({ title: "Guardrail policy updated" });
      }
    } catch (error) {
      console.error(error);
      toast({
        variant: "destructive",
        title: "Failed to update guardrails",
        description: "Retry in a moment.",
      });
    } finally {
      setKeyGuardrailSaving(false);
    }
  };

  useEffect(() => {
    setKeyGuardrailOverride(false);
    setKeyGuardrailHadOverride(false);
    setKeyGuardrailEnabled(true);
    setKeyGuardrailPromptKeywords("");
    setKeyGuardrailResponseKeywords("");
    setKeyGuardrailModerationEnabled(false);
    setKeyGuardrailModerationProvider("keyword");
    setKeyGuardrailModerationAction("block");
    setKeyGuardrailWebhookURL("");
    setKeyGuardrailWebhookHeader("");
    setKeyGuardrailWebhookValue("");
    setKeyGuardrailWebhookTimeout("5");

    if (!selectedKey) {
      setKeyGuardrailLoading(false);
      setKeyGuardrailSaving(false);
      return;
    }

    setKeyGuardrailLoading(true);
    getApiKeyGuardrails(selectedKey.tenant_id, selectedKey.id)
      .then(({ config }) => {
        const hasOverride = hasGuardrailConfig(config);
        setKeyGuardrailOverride(hasOverride);
        setKeyGuardrailHadOverride(hasOverride);
        setKeyGuardrailEnabled(
          config.enabled ?? (hasOverride ? true : false),
        );
        setKeyGuardrailPromptKeywords(
          formatKeywordInput(config.prompt?.blocked_keywords),
        );
        setKeyGuardrailResponseKeywords(
          formatKeywordInput(config.response?.blocked_keywords),
        );
        setKeyGuardrailModerationEnabled(
          config.moderation?.enabled ?? false,
        );
        setKeyGuardrailModerationProvider(
          config.moderation?.provider || "keyword",
        );
        setKeyGuardrailModerationAction(
          config.moderation?.action ?? "block",
        );
        setKeyGuardrailWebhookURL(config.moderation?.webhook_url ?? "");
        setKeyGuardrailWebhookHeader(
          config.moderation?.webhook_auth_header ?? "",
        );
        setKeyGuardrailWebhookValue(
          config.moderation?.webhook_auth_value ?? "",
        );
        setKeyGuardrailWebhookTimeout(
          config.moderation?.timeout_seconds != null
            ? config.moderation.timeout_seconds.toString()
            : "5",
        );
      })
      .catch(() => {
        toast({
          variant: "destructive",
          title: "Failed to load guardrail policy",
          description: "Try reopening the key details dialog.",
        });
      })
      .finally(() => setKeyGuardrailLoading(false));
  }, [selectedKey, toast]);

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
                      placeholder="200"
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
                <Separator />
                <div className="space-y-3">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="text-sm font-medium text-foreground">
                        Guardrail policy
                      </p>
                      <p className="text-sm text-muted-foreground">
                        Override the inherited guardrails for this key.
                      </p>
                    </div>
                    <Switch
                      checked={keyGuardrailOverride}
                      disabled={keyGuardrailLoading || keyGuardrailSaving}
                      onCheckedChange={setKeyGuardrailOverride}
                    />
                  </div>
                  {keyGuardrailOverride ? (
                    <div className="space-y-4 rounded-lg border p-4">
                      <div className="flex items-start justify-between gap-4">
                        <div>
                          <p className="text-sm font-medium text-foreground">
                            Enforce guardrails
                          </p>
                          <p className="text-sm text-muted-foreground">
                            Disable temporarily without removing the policy.
                          </p>
                        </div>
                        <Switch
                          checked={keyGuardrailEnabled}
                          disabled={keyGuardrailLoading || keyGuardrailSaving}
                          onCheckedChange={setKeyGuardrailEnabled}
                        />
                      </div>
                      <div className="grid gap-4 md:grid-cols-2">
                        <div className="space-y-2">
                          <Label htmlFor="key-guardrail-prompt">
                            Prompt keywords
                          </Label>
                          <Textarea
                            id="key-guardrail-prompt"
                            value={keyGuardrailPromptKeywords}
                            onChange={(event) =>
                              setKeyGuardrailPromptKeywords(event.target.value)
                            }
                            placeholder="fraud\nhate"
                            disabled={keyGuardrailLoading || keyGuardrailSaving}
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="key-guardrail-response">
                            Response keywords
                          </Label>
                          <Textarea
                            id="key-guardrail-response"
                            value={keyGuardrailResponseKeywords}
                            onChange={(event) =>
                              setKeyGuardrailResponseKeywords(
                                event.target.value,
                              )
                            }
                            placeholder="password\npii"
                            disabled={keyGuardrailLoading || keyGuardrailSaving}
                          />
                        </div>
                      </div>
                      <div className="space-y-2">
                        <Label>Moderation provider</Label>
                        <Select
                          value={keyGuardrailModerationProvider}
                          onValueChange={setKeyGuardrailModerationProvider}
                          disabled={keyGuardrailLoading || keyGuardrailSaving}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="keyword">Keyword only</SelectItem>
                            <SelectItem value="webhook">Webhook</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-2">
                          <Label>Moderation action</Label>
                          <Select
                            value={keyGuardrailModerationAction}
                            onValueChange={setKeyGuardrailModerationAction}
                            disabled={keyGuardrailLoading || keyGuardrailSaving}
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="block">Block</SelectItem>
                              <SelectItem value="warn">Warn</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <div className="space-y-2">
                          <Label>Moderation check</Label>
                          <div className="flex items-center justify-between rounded-lg border px-4 py-3">
                            <p className="text-sm text-muted-foreground">
                              Send prompts/responses for classification.
                            </p>
                            <Switch
                              checked={keyGuardrailModerationEnabled}
                              disabled={
                                keyGuardrailLoading || keyGuardrailSaving
                              }
                              onCheckedChange={setKeyGuardrailModerationEnabled}
                            />
                          </div>
                        </div>
                      </div>
                      {keyGuardrailModerationProvider === "webhook" ? (
                        <div className="space-y-3 rounded-lg border p-4">
                          <div className="space-y-2">
                            <Label htmlFor="key-guardrail-webhook-url">
                              Webhook URL
                            </Label>
                            <Input
                              id="key-guardrail-webhook-url"
                              value={keyGuardrailWebhookURL}
                              onChange={(event) =>
                                setKeyGuardrailWebhookURL(event.target.value)
                              }
                              placeholder="https://example.com/moderate"
                              disabled={keyGuardrailLoading || keyGuardrailSaving}
                            />
                          </div>
                          <div className="grid gap-4 sm:grid-cols-2">
                            <div className="space-y-2">
                              <Label htmlFor="key-guardrail-webhook-header">
                                Auth header
                              </Label>
                              <Input
                                id="key-guardrail-webhook-header"
                                value={keyGuardrailWebhookHeader}
                                onChange={(event) =>
                                  setKeyGuardrailWebhookHeader(
                                    event.target.value,
                                  )
                                }
                                placeholder="Authorization"
                                disabled={keyGuardrailLoading || keyGuardrailSaving}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="key-guardrail-webhook-value">
                                Auth value
                              </Label>
                              <Input
                                id="key-guardrail-webhook-value"
                                value={keyGuardrailWebhookValue}
                                onChange={(event) =>
                                  setKeyGuardrailWebhookValue(
                                    event.target.value,
                                  )
                                }
                                placeholder="Bearer ..."
                                disabled={keyGuardrailLoading || keyGuardrailSaving}
                              />
                            </div>
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor="key-guardrail-webhook-timeout">
                              Timeout (seconds)
                            </Label>
                            <Input
                              id="key-guardrail-webhook-timeout"
                              value={keyGuardrailWebhookTimeout}
                              onChange={(event) =>
                                setKeyGuardrailWebhookTimeout(
                                  event.target.value,
                                )
                              }
                              disabled={keyGuardrailLoading || keyGuardrailSaving}
                            />
                          </div>
                          <p className="text-xs text-muted-foreground">
                            Webhook receives {"{ stage, content }"} and
                            responds with {"{ action, violations[], category }"}.
                          </p>
                        </div>
                      ) : null}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      Inheriting tenant guardrails.
                    </p>
                  )}
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setSelectedKey(null)}>
                  Close
                </Button>
                <Button
                  onClick={handleSaveKeyGuardrails}
                  disabled={
                    keyGuardrailSaving ||
                    keyGuardrailLoading ||
                    !selectedKey
                  }
                >
                  {keyGuardrailSaving ? "Saving…" : "Save guardrails"}
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

function hasGuardrailConfig(config?: GuardrailConfig): boolean {
  if (!config) return false;
  return Object.keys(config).length > 0;
}
