import { useState, useCallback } from "react";
import { Key, RefreshCcw, Search } from "lucide-react";
import { PageHeader } from "@/components/layouts";

import type { ApiKeyRecord, CreateApiKeyResponse } from "@/api/tenants";
import { revokeTenantApiKey } from "@/api/tenants";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
  AdminKeyCreateDialog,
  AdminKeyDetailsDialog,
  IssuedKeyDialog,
  BulkKeyActionBar,
  useAdminKeyMutations,
  useKeysPageData,
  useKeysFilter,
} from "@/features/api-keys";

export function KeysPage() {
  const { toast } = useToast();
  const { createKeyMutation, revokeKeyMutation } = useAdminKeyMutations();

  const {
    tenants,
    budgetDefaults,
    selectedTenantId,
    setSelectedTenantId,
    tenantBudgetMap,
    defaultKeyRateLimit,
    effectiveTenantRateLimit,
    keysQuery,
    formatBudgetValue,
    formatWarningThresholdValue,
  } = useKeysPageData();

  const keys: ApiKeyRecord[] = keysQuery.data?.api_keys ?? [];

  const {
    searchTerm,
    setSearchTerm,
    issuerFilter,
    setIssuerFilter,
    statusFilter,
    setStatusFilter,
    filteredKeys,
    activeKeys,
    revokedKeys,
  } = useKeysFilter(keys);

  const [createOpen, setCreateOpen] = useState(false);
  const [issuedKey, setIssuedKey] = useState<CreateApiKeyResponse | null>(null);
  const [selectedKey, setSelectedKey] = useState<ApiKeyRecord | null>(null);
  const [pendingRevokeKey, setPendingRevokeKey] = useState<ApiKeyRecord | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [bulkRevokeLoading, setBulkRevokeLoading] = useState(false);
  const [pendingBulkRevoke, setPendingBulkRevoke] = useState(false);

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

  const handleCreateKey = async (tenantId: string, payload: Parameters<typeof createKeyMutation.mutateAsync>[0]["payload"]) => {
    const result = await createKeyMutation.mutateAsync({ tenantId, payload });
    setIssuedKey(result);
    return result;
  };

  const handleBulkRevoke = useCallback(async () => {
    setBulkRevokeLoading(true);
    const keysToRevoke = keys.filter(
      (k) => selectedIds.has(k.id) && !k.revoked
    );
    try {
      for (const key of keysToRevoke) {
        await revokeTenantApiKey(key.tenant_id, key.id);
      }
      toast({ title: `${keysToRevoke.length} key(s) revoked` });
      setSelectedIds(new Set());
      setPendingBulkRevoke(false);
      void keysQuery.refetch();
    } catch {
      toast({ variant: "destructive", title: "Failed to revoke keys" });
    } finally {
      setBulkRevokeLoading(false);
    }
  }, [keys, selectedIds, toast, keysQuery]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="API Keys"
        description="Issue, rotate, and revoke tenant-scoped virtual keys with quota controls."
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={() => keysQuery.refetch()}
              loading={keysQuery.isFetching}
            >
              <RefreshCcw className="h-4 w-4" />
            </Button>
            <AdminKeyCreateDialog
              open={createOpen}
              onOpenChange={setCreateOpen}
              selectedTenantId={selectedTenantId}
              onTenantChange={setSelectedTenantId}
              tenants={tenants}
              tenantBudgetMap={tenantBudgetMap}
              budgetDefaults={budgetDefaults}
              defaultKeyRateLimit={defaultKeyRateLimit}
              effectiveTenantRateLimit={effectiveTenantRateLimit}
              isSubmitting={createKeyMutation.isPending}
              onSubmit={handleCreateKey}
            />
          </div>
        }
      />

      {/* Summary Stats */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Keys</CardTitle>
            <Key className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{keys.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active</CardTitle>
            <div className="h-2 w-2 rounded-full bg-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{activeKeys.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Revoked</CardTitle>
            <div className="h-2 w-2 rounded-full bg-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{revokedKeys.length}</div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-col gap-4 lg:flex-row lg:items-start">
          <div className="flex-1">
            <CardTitle>Key registry</CardTitle>
            <p className="text-sm text-muted-foreground">
              {filteredKeys.length} of {keys.length} keys
              {filteredKeys.length !== keys.length && " matching filters"}
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
            selectedIds={selectedIds}
            onSelectionChange={setSelectedIds}
          />
        </CardContent>
      </Card>

      {/* Bulk Action Bar */}
      <BulkKeyActionBar
        selectedCount={selectedIds.size}
        onRevoke={() => setPendingBulkRevoke(true)}
        onClear={() => setSelectedIds(new Set())}
        isLoading={bulkRevokeLoading}
      />

      {/* Bulk Revoke Confirmation Dialog */}
      <AlertDialog
        open={pendingBulkRevoke}
        onOpenChange={(open) => !open && setPendingBulkRevoke(false)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke {selectedIds.size} keys?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. All selected keys will stop working
              immediately.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={bulkRevokeLoading}
              onClick={handleBulkRevoke}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Revoke all
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Single Revoke Confirmation Dialog */}
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

      <AdminKeyDetailsDialog
        selectedKey={selectedKey}
        onClose={() => setSelectedKey(null)}
        formatBudgetValue={formatBudgetValue}
        formatWarningThresholdValue={formatWarningThresholdValue}
      />

      <IssuedKeyDialog
        issuedKey={
          issuedKey ? { token: issuedKey.token, secret: issuedKey.secret } : null
        }
        onCopy={handleCopy}
        onClose={() => setIssuedKey(null)}
      />
    </div>
  );
}
