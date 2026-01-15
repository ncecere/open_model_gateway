import { useState, useCallback } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Key } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { TabsContent } from "@/components/ui/tabs";
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
  listAdminKeys,
  createAdminKey,
  revokeAdminKey,
} from "@/api/admin-keys";
import {
  AdminKeyDirectoryCard,
  AdminKeyCreateDialog,
  AdminKeyDetailsDialog,
  AdminKeyIssuedDialog,
  BulkAdminKeyActionBar,
  useAdminKeyFilters,
  useAdminKeyDialogs,
  isKeyActive,
} from "@/features/admin-keys";

export function AdminKeysTab() {
  const { toast } = useToast();
  const queryClient = useQueryClient();

  // Data fetching
  const adminKeysQuery = useQuery({
    queryKey: ["admin-keys"],
    queryFn: listAdminKeys,
  });
  const adminKeys = adminKeysQuery.data ?? [];

  // Filters and selection
  const {
    searchTerm,
    setSearchTerm,
    scopeFilter,
    setScopeFilter,
    statusFilter,
    setStatusFilter,
    filteredKeys,
    stats,
  } = useAdminKeyFilters(adminKeys);

  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [bulkRevokeLoading, setBulkRevokeLoading] = useState(false);

  // Dialogs
  const { createDialog, detailsDialog, issuedDialog, revokeDialog } =
    useAdminKeyDialogs();

  // Mutations
  const createMutation = useMutation({
    mutationFn: createAdminKey,
    onSuccess: (data) => {
      toast({ title: "Admin token created" });
      issuedDialog.setToken(data.token);
      createDialog.reset();
      void queryClient.invalidateQueries({ queryKey: ["admin-keys"] });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to create token",
        description: "Check your permissions and try again.",
      });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: revokeAdminKey,
    onSuccess: () => {
      toast({ title: "Token revoked" });
      revokeDialog.close();
      setSelectedIds(new Set());
      void queryClient.invalidateQueries({ queryKey: ["admin-keys"] });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to revoke token",
      });
    },
  });

  // Handlers
  const handleCreate = useCallback(() => {
    const form = createDialog.form;
    if (!form.name.trim()) {
      toast({ variant: "destructive", title: "Name is required" });
      return;
    }
    const expiresDays = Number(form.expiresDays);
    if (expiresDays <= 0) {
      toast({ variant: "destructive", title: "Expiry must be at least 1 day" });
      return;
    }
    createMutation.mutate({
      name: form.name.trim(),
      scope: form.scope,
      expires_in_seconds: expiresDays * 24 * 60 * 60,
    });
  }, [createDialog.form, createMutation, toast]);

  const handleRevoke = useCallback(
    (record: { id: string }) => {
      revokeMutation.mutate(record.id);
    },
    [revokeMutation],
  );

  const handleBulkRevoke = useCallback(async () => {
    setBulkRevokeLoading(true);
    const keysToRevoke = adminKeys.filter(
      (k) => selectedIds.has(k.id) && isKeyActive(k),
    );
    try {
      for (const key of keysToRevoke) {
        await revokeAdminKey(key.id);
      }
      toast({ title: `${keysToRevoke.length} token(s) revoked` });
      setSelectedIds(new Set());
      revokeDialog.close();
      void queryClient.invalidateQueries({ queryKey: ["admin-keys"] });
    } catch {
      toast({ variant: "destructive", title: "Failed to revoke tokens" });
    } finally {
      setBulkRevokeLoading(false);
    }
  }, [adminKeys, selectedIds, toast, queryClient, revokeDialog]);

  const handleRevokeFromDetails = useCallback(
    (record: { id: string }) => {
      detailsDialog.close();
      revokeDialog.openWith([adminKeys.find((k) => k.id === record.id)!]);
    },
    [detailsDialog, revokeDialog, adminKeys],
  );

  const handleConfirmRevoke = useCallback(() => {
    if (revokeDialog.records.length === 1) {
      handleRevoke(revokeDialog.records[0]);
    } else {
      handleBulkRevoke();
    }
  }, [revokeDialog.records, handleRevoke, handleBulkRevoke]);

  return (
    <TabsContent value="admin-keys" className="space-y-6" forceMount>
      {/* Summary Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Tokens</CardTitle>
            <Key className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.total}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active</CardTitle>
            <div className="h-2 w-2 rounded-full bg-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.active}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Expired</CardTitle>
            <div className="h-2 w-2 rounded-full bg-amber-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.expired}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Revoked</CardTitle>
            <div className="h-2 w-2 rounded-full bg-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.revoked}</div>
          </CardContent>
        </Card>
      </div>

      {/* Directory Card */}
      <AdminKeyDirectoryCard
        keys={adminKeys}
        filteredKeys={filteredKeys}
        searchTerm={searchTerm}
        setSearchTerm={setSearchTerm}
        scopeFilter={scopeFilter}
        setScopeFilter={setScopeFilter}
        statusFilter={statusFilter}
        setStatusFilter={setStatusFilter}
        isLoading={adminKeysQuery.isLoading}
        selectedIds={selectedIds}
        onSelectionChange={setSelectedIds}
        onViewDetails={detailsDialog.openWith}
        onRevoke={(record) => revokeDialog.openWith([record])}
        onCreate={() => createDialog.setOpen(true)}
      />

      {/* Bulk Action Bar */}
      <BulkAdminKeyActionBar
        selectedCount={selectedIds.size}
        onRevoke={() => {
          const records = adminKeys.filter(
            (k) => selectedIds.has(k.id) && isKeyActive(k),
          );
          revokeDialog.openWith(records);
        }}
        onClear={() => setSelectedIds(new Set())}
        isLoading={bulkRevokeLoading}
      />

      {/* Create Dialog */}
      <AdminKeyCreateDialog
        open={createDialog.open}
        onOpenChange={createDialog.setOpen}
        form={createDialog.form}
        onChange={createDialog.setForm}
        onSubmit={handleCreate}
        loading={createMutation.isPending}
      />

      {/* Issued Token Dialog */}
      <AdminKeyIssuedDialog
        open={issuedDialog.open}
        token={issuedDialog.token}
        onClose={issuedDialog.close}
      />

      {/* Details Dialog */}
      <AdminKeyDetailsDialog
        open={detailsDialog.open}
        onOpenChange={(open) => {
          if (!open) detailsDialog.close();
          else detailsDialog.setOpen(open);
        }}
        record={detailsDialog.record}
        onRevoke={handleRevokeFromDetails}
      />

      {/* Revoke Confirmation Dialog */}
      <AlertDialog open={revokeDialog.open} onOpenChange={(open) => !open && revokeDialog.close()}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Revoke {revokeDialog.records.length > 1 ? "tokens" : "token"}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              {revokeDialog.records.length === 1
                ? `"${revokeDialog.records[0]?.name}" will stop working immediately.`
                : `${revokeDialog.records.length} tokens will stop working immediately.`}{" "}
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => revokeDialog.close()}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmRevoke}
              disabled={revokeMutation.isPending || bulkRevokeLoading}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Revoke
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </TabsContent>
  );
}
