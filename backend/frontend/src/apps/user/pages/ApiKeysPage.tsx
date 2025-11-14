import { useEffect, useMemo, useState } from "react";
import { useQueries } from "@tanstack/react-query";
import { Copy, Key, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { BudgetMeter } from "@/ui/kit/BudgetMeter";
import { useToast } from "@/hooks/use-toast";
import { computeNextResetDate, formatScheduleLabel } from "@/features/api-keys";
import type { UserAPIKey } from "../../../api/user/api-keys";
import {
  useCreateUserAPIKeyMutation,
  useCreateTenantAPIKeyMutation,
  useRevokeUserAPIKeyMutation,
  useRevokeTenantAPIKeyMutation,
  useTenantAPIKeysQuery,
  useUserAPIKeysQuery,
  useUserTenantsQuery,
  useAllTenantAPIKeysQueries,
} from "../hooks/useUserData";
import { getTenantSummary, type TenantBudgetSummary } from "../../../api/user/tenants";

const dateFormatter = new Intl.DateTimeFormat(undefined, {
  year: "numeric",
  month: "short",
  day: "numeric",
});

type BudgetMeta = {
  limit: number | null;
  used: number;
  warning: number;
  schedule: string;
};

type IssuedSecret =
  | null
  | {
      scope: "personal" | "tenant";
      tenantName?: string;
      name: string;
      prefix: string;
      secret: string;
      token: string;
    };

type RevokedRow = UserAPIKey & { tenantLabel: string };

export function UserApiKeysPage() {
  const [activeTab, setActiveTab] = useState<"personal" | "tenant" | "revoked">(
    "personal",
  );
  const { data: personalKeys, isLoading: personalLoading } = useUserAPIKeysQuery();
  const { data: tenants } = useUserTenantsQuery();
  const tenantOptions = useMemo(
    () => (tenants ?? []).filter((tenant) => !tenant.is_personal),
    [tenants],
  );
  const tenantIds = tenantOptions.map((tenant) => tenant.tenant_id);
  const [selectedTenantId, setSelectedTenantId] = useState<string>();
  useEffect(() => {
    if (!selectedTenantId && tenantOptions.length) {
      setSelectedTenantId(tenantOptions[0].tenant_id);
    }
  }, [selectedTenantId, tenantOptions]);

  const { data: tenantKeyData, isFetching: tenantKeysLoading } =
    useTenantAPIKeysQuery(selectedTenantId);
  const allTenantKeyQueries = useAllTenantAPIKeysQueries(tenantIds);
  const uniqueBudgetTenantIds = useMemo(() => {
    const ids = new Set<string>();
    (personalKeys ?? []).forEach((key) => ids.add(key.tenant_id));
    tenantOptions.forEach((tenant) => ids.add(tenant.tenant_id));
    return Array.from(ids);
  }, [personalKeys, tenantOptions]);

  const tenantSummaryQueries = useQueries({
    queries: uniqueBudgetTenantIds.map((tenantId) => ({
      queryKey: ["user-tenant-summary", tenantId],
      queryFn: () => getTenantSummary(tenantId),
      enabled: Boolean(tenantId),
    })),
  });

  const tenantBudgetMap = useMemo(() => {
    const map = new Map<string, TenantBudgetSummary>();
    tenantSummaryQueries.forEach((query, index) => {
      const tenantId = uniqueBudgetTenantIds[index];
      if (tenantId && query.data?.budget) {
        map.set(tenantId, query.data.budget);
      }
    });
    return map;
  }, [tenantSummaryQueries, uniqueBudgetTenantIds]);
  const tenantRole = tenantKeyData?.role;
  const tenantKeys = tenantKeyData?.api_keys ?? [];
  const tenantActiveKeys = tenantKeys.filter((key) => !key.revoked);
  const selectedTenant = tenantOptions.find(
    (tenant) => tenant.tenant_id === selectedTenantId,
  );
  const canManageTenant =
    tenantRole === "owner" || tenantRole === "admin";

  const createMutation = useCreateUserAPIKeyMutation();
  const revokeMutation = useRevokeUserAPIKeyMutation();
  const tenantCreateMutation = useCreateTenantAPIKeyMutation();
  const tenantRevokeMutation = useRevokeTenantAPIKeyMutation();
  const { toast } = useToast();

  const [name, setName] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [tenantCreateOpen, setTenantCreateOpen] = useState(false);
  const [tenantKeyName, setTenantKeyName] = useState("");
  const [issuedSecret, setIssuedSecret] = useState<IssuedSecret>(null);

  const handleCreate = async () => {
    if (!name.trim()) {
      toast({ variant: "destructive", title: "Name is required" });
      return;
    }
    try {
      const result = await createMutation.mutateAsync({
        name: name.trim(),
      });
      setIssuedSecret({
        scope: "personal",
        name: result.api_key.name,
        prefix: result.api_key.prefix,
        secret: result.secret,
        token: result.token,
      });
      setName("");
      setCreateOpen(false);
      toast({ title: "API key created", description: "Copy the secret now." });
    } catch (error) {
      console.error(error);
      toast({ variant: "destructive", title: "Failed to create key" });
    }
  };

  const handleTenantCreate = async () => {
    if (!selectedTenantId) {
      toast({
        variant: "destructive",
        title: "Select a tenant",
        description: "Choose a tenant before creating a key.",
      });
      return;
    }
    if (!tenantKeyName.trim()) {
      toast({ variant: "destructive", title: "Name is required" });
      return;
    }
    try {
      const result = await tenantCreateMutation.mutateAsync({
        tenantId: selectedTenantId,
        payload: {
          name: tenantKeyName.trim(),
        },
      });
      setIssuedSecret({
        scope: "tenant",
        tenantName: selectedTenant?.name,
        name: result.api_key.name,
        prefix: result.api_key.prefix,
        secret: result.secret,
        token: result.token,
      });
      setTenantKeyName("");
      setTenantCreateOpen(false);
      toast({
        title: "Tenant API key created",
        description: `Key issued for ${selectedTenant?.name ?? "tenant"}.`,
      });
    } catch (error) {
      console.error(error);
      toast({
        variant: "destructive",
        title: "Failed to create tenant key",
      });
    }
  };

  const handleCopy = (value: string) => {
    navigator.clipboard.writeText(value).then(() => {
      toast({ title: "Copied to clipboard" });
    });
  };

  const handleRevoke = async (id: string) => {
    try {
      await revokeMutation.mutateAsync(id);
      toast({ title: "API key revoked" });
    } catch (error) {
      console.error(error);
      toast({ variant: "destructive", title: "Failed to revoke key" });
    }
  };

  const handleTenantRevoke = async (keyId: string) => {
    if (!selectedTenantId) {
      toast({ variant: "destructive", title: "Select a tenant first" });
      return;
    }
    try {
      await tenantRevokeMutation.mutateAsync({
        tenantId: selectedTenantId,
        apiKeyId: keyId,
      });
      toast({ title: "Tenant API key revoked" });
    } catch (error) {
      console.error(error);
      toast({ variant: "destructive", title: "Failed to revoke tenant key" });
    }
  };

  const keys = personalKeys ?? [];
  const activeKeys = keys.filter((key) => !key.revoked);
  const revokedKeys = keys.filter((key) => key.revoked);
  const revokedRows: RevokedRow[] = useMemo(() => {
    const tenantRevoked = allTenantKeyQueries.flatMap((query, index) => {
      const tenantMeta = tenantOptions[index];
      if (!tenantMeta || !query.data) {
        return [];
      }
      return query.data.api_keys
        .filter((key) => key.revoked)
        .map((key) => ({ ...key, tenantLabel: tenantMeta.name }));
    });
    return [
      ...revokedKeys.map((key) => ({ ...key, tenantLabel: "Personal" })),
      ...tenantRevoked,
    ];
  }, [revokedKeys, allTenantKeyQueries, tenantOptions]);
  const revokedLoading =
    personalLoading ||
    allTenantKeyQueries.some((query) => query.isLoading || query.isFetching);

  const formatRole = (role?: string) =>
    role ? role.charAt(0).toUpperCase() + role.slice(1) : "—";

  const resolveBudgetMeta = (key: UserAPIKey): BudgetMeta => {
    const tenantBudget = tenantBudgetMap.get(key.tenant_id);
    const fallbackLimit = tenantBudget?.limit_usd ?? null;
    const limit =
      key.quota?.budget_usd ??
      (typeof key.quota?.budget_cents === "number"
        ? key.quota.budget_cents / 100
        : fallbackLimit);
    const used = tenantBudget?.used_usd ?? 0;
    const warning =
      key.quota?.warning_threshold ??
      tenantBudget?.warning_threshold ??
      0.8;
    const schedule =
      key.budget_refresh_schedule ||
      tenantBudget?.refresh_schedule ||
      "calendar_month";
    return { limit, used, warning, schedule };
  };

  const formatResetValue = (key: UserAPIKey) => {
    const { schedule } = resolveBudgetMeta(key);
    const label = formatScheduleLabel(schedule);
    const next = computeNextResetDate(schedule);
    if (!next) {
      return label;
    }
    return `${label} · ${dateFormatter.format(next)}`;
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">API Keys</h1>
          <p className="text-sm text-muted-foreground">
            Personal keys are always available. Tenant keys respect the role of
            each membership.
          </p>
        </div>
      </div>

      {issuedSecret ? (
        <Card>
          <CardHeader>
            <CardTitle>Save this secret</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <p>
              {issuedSecret.scope === "tenant"
                ? `Issued for ${issuedSecret.tenantName ?? "selected tenant"}.`
                : "Issued for your personal tenant."}{" "}
              This is the only time the full secret will be shown.
            </p>
            <div className="rounded-md border p-3">
              <p className="text-xs text-muted-foreground">Token</p>
              <div className="mt-1 flex items-center justify-between gap-4">
                <code className="truncate">{issuedSecret.token}</code>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleCopy(issuedSecret.token)}
                >
                  <Copy className="mr-1 size-4" /> Copy
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      ) : null}

      <Tabs
        value={activeTab}
        onValueChange={(value) => setActiveTab(value as "personal" | "tenant")}
        className="space-y-6"
      >
        <TabsList>
          <TabsTrigger value="personal">Personal keys</TabsTrigger>
          <TabsTrigger value="tenant" disabled={!tenantOptions.length}>
            Tenant keys
          </TabsTrigger>
          <TabsTrigger value="revoked">Revoked keys</TabsTrigger>
        </TabsList>
        <TabsContent value="personal" className="space-y-6">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-xl font-semibold">Personal tenant</h2>
              <p className="text-sm text-muted-foreground">
                Keys scoped to your personal tenant inherit default model/
                budget controls.
              </p>
            </div>
            <Dialog open={createOpen} onOpenChange={setCreateOpen}>
              <DialogTrigger asChild>
                <Button>
                  <Key className="mr-2 size-4" />
                  Create key
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Create personal API key</DialogTitle>
                  <DialogDescription>
                    Keys are scoped to your personal tenant and inherit default
                    budgets.
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 py-2">
                  <div className="space-y-2">
                    <Label htmlFor="key-name">Name</Label>
                    <Input
                      id="key-name"
                      value={name}
                      onChange={(event) => setName(event.target.value)}
                      placeholder="My personal key"
                    />
                  </div>
                </div>
                <DialogFooter>
                  <Button
                    onClick={handleCreate}
                    disabled={createMutation.isPending}
                  >
                    Issue key
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          <section className="grid gap-6">
            <KeyTable
              title="Active keys"
              loading={personalLoading}
              keys={activeKeys}
              variant="active"
              onRevoke={handleRevoke}
              allowRevoke
              getBudgetMeta={resolveBudgetMeta}
              formatResetValue={formatResetValue}
            />
          </section>
        </TabsContent>

        <TabsContent value="tenant" className="space-y-6">
          {tenantOptions.length === 0 ? (
            <Card>
              <CardHeader>
                <CardTitle>No shared tenants</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  Join a tenant as an owner or admin to manage shared API keys.
                </p>
              </CardContent>
            </Card>
          ) : (
            <>
              <div className="flex flex-col gap-4 lg:flex-row lg:items-end">
                <div className="flex-1 space-y-2">
                  <Label>Select tenant</Label>
                  <Select
                    value={selectedTenantId}
                    onValueChange={(value) => setSelectedTenantId(value)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select tenant" />
                    </SelectTrigger>
                    <SelectContent>
                      {tenantOptions.map((tenant) => (
                        <SelectItem
                          key={tenant.tenant_id}
                          value={tenant.tenant_id}
                        >
                          {tenant.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex-1 space-y-1">
                  <Label>Role</Label>
                  <p className="rounded-md border px-3 py-2 text-sm">
                    {formatRole(tenantRole)}{" "}
                    {!canManageTenant ? "(read-only)" : "(manage keys allowed)"}
                  </p>
                </div>
                <Dialog
                  open={tenantCreateOpen}
                  onOpenChange={(open) => {
                    if (!canManageTenant) {
                      toast({
                        variant: "destructive",
                        title: "Insufficient role",
                        description:
                          "Only tenant owners and admins can create keys.",
                      });
                      return;
                    }
                    setTenantCreateOpen(open);
                  }}
                >
                  <DialogTrigger asChild>
                    <Button disabled={!canManageTenant}>
                      <Key className="mr-2 size-4" />
                      Issue key
                    </Button>
                  </DialogTrigger>
                  <DialogContent>
                    <DialogHeader>
                      <DialogTitle>Create tenant API key</DialogTitle>
                      <DialogDescription>
                        {selectedTenant
                          ? `Keys are scoped to ${selectedTenant.name}.`
                          : "Select a tenant to continue."}
                      </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-2">
                      <div className="space-y-2">
                        <Label htmlFor="tenant-key-name">Name</Label>
                        <Input
                          id="tenant-key-name"
                          value={tenantKeyName}
                          onChange={(event) => setTenantKeyName(event.target.value)}
                          placeholder="Shared key"
                        />
                      </div>
                    </div>
                    <DialogFooter>
                      <Button
                        onClick={handleTenantCreate}
                        disabled={tenantCreateMutation.isPending}
                      >
                        Issue key
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              </div>

              <section className="grid gap-6">
                <KeyTable
                  title="Active keys"
                  loading={tenantKeysLoading}
                  keys={tenantActiveKeys}
                  variant="active"
                  allowRevoke={canManageTenant}
                  onRevoke={handleTenantRevoke}
                  getBudgetMeta={resolveBudgetMeta}
                  formatResetValue={formatResetValue}
                />
              </section>
            </>
          )}
        </TabsContent>

        <TabsContent value="revoked" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Revoked keys</CardTitle>
            </CardHeader>
            <CardContent>
              {revokedLoading ? (
                <div className="space-y-2">
                  {[...Array(4)].map((_, idx) => (
                    <div key={idx} className="h-10 animate-pulse rounded bg-muted" />
                  ))}
                </div>
              ) : revokedRows.length ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Prefix</TableHead>
                      <TableHead>Scope</TableHead>
                      <TableHead>Revoked at</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {revokedRows.map((key) => (
                      <TableRow key={key.id}>
                        <TableCell>{key.name}</TableCell>
                        <TableCell>{key.prefix}</TableCell>
                        <TableCell>{key.tenantLabel}</TableCell>
                        <TableCell>
                          {key.revoked_at
                            ? new Date(key.revoked_at).toLocaleDateString()
                            : "—"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No revoked keys yet.
                </p>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

type KeyTableProps = {
  title: string;
  loading: boolean;
  keys: UserAPIKey[];
  variant: "active" | "revoked";
  allowRevoke?: boolean;
  onRevoke?: (id: string) => void;
  getBudgetMeta: (key: UserAPIKey) => BudgetMeta;
  formatResetValue: (key: UserAPIKey) => string;
};

function KeyTable({
  title,
  loading,
  keys,
  variant,
  allowRevoke = false,
  onRevoke,
  getBudgetMeta,
  formatResetValue,
}: KeyTableProps) {
  const hasKeys = keys.length > 0;
  const showActions = variant === "active" && allowRevoke;
  const showBudgetColumns = variant === "active";

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="space-y-2">
            {[...Array(4)].map((_, idx) => (
              <div key={idx} className="h-10 animate-pulse rounded bg-muted" />
            ))}
          </div>
        ) : hasKeys ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Prefix</TableHead>
                {showBudgetColumns ? (
                  <>
                    <TableHead>Budget</TableHead>
                    <TableHead>Reset schedule</TableHead>
                  </>
                ) : null}
                {variant === "active" && showActions ? (
                  <TableHead className="text-right">Actions</TableHead>
                ) : null}
                {variant === "revoked" ? <TableHead>Revoked at</TableHead> : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((key) => {
                const budgetMeta = getBudgetMeta(key);
                const hasBudget =
                  typeof budgetMeta.limit === "number" && budgetMeta.limit > 0;
                return (
                  <TableRow key={key.id}>
                    <TableCell>{key.name}</TableCell>
                    <TableCell>{key.prefix}</TableCell>
                    {showBudgetColumns ? (
                      <>
                        <TableCell className="min-w-[220px] align-top">
                          {hasBudget ? (
                            <div className="space-y-1.5">
                              <BudgetMeter
                                used={budgetMeta.used}
                                limit={budgetMeta.limit ?? 0}
                              />
                              <p className="text-xs text-muted-foreground">
                                Warn at {Math.round(budgetMeta.warning * 100)}%
                              </p>
                            </div>
                          ) : (
                            <p className="text-sm text-muted-foreground">
                              Inherits tenant defaults
                            </p>
                          )}
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {formatResetValue(key)}
                        </TableCell>
                      </>
                    ) : null}
                    {variant === "active" && showActions && onRevoke ? (
                      <TableCell className="flex justify-end gap-2">
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="text-destructive"
                            >
                              <Trash2 className="size-4" />
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Revoke API key</AlertDialogTitle>
                              <AlertDialogDescription>
                                This action cannot be undone. Requests using this
                                key will immediately fail.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => onRevoke(key.id)}
                              >
                                Revoke
                              </AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      </TableCell>
                    ) : null}
                    {variant === "revoked" ? (
                      <TableCell>
                        {key.revoked_at
                          ? new Date(key.revoked_at).toLocaleDateString()
                          : "—"}
                      </TableCell>
                    ) : null}
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        ) : (
          <p className="text-sm text-muted-foreground">No data yet.</p>
        )}
      </CardContent>
    </Card>
  );
}
