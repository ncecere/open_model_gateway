import { useState, useCallback } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { RefreshCcw, UserPlus, Users } from "lucide-react";
import { PageHeader } from "@/components/layouts";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
import { createUser, sendUserInvite } from "@/api/users";
import { updateTenantStatus, type TenantStatus } from "@/api/tenants";

import {
  UserDirectoryCard,
  UserEditorDialog,
  UserDetailsDialog,
  BulkUserActionBar,
  usePersonalTenantsQuery,
  usePersonalTenantFilters,
  useUserDialogs,
  type PersonalTenantRecord,
} from "@/features/users";

export function UsersPage() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const personalTenantsQuery = usePersonalTenantsQuery();
  const records = personalTenantsQuery.data?.personal_tenants ?? [];
  const { searchTerm, setSearchTerm, statusFilter, setStatusFilter, filteredRecords } =
    usePersonalTenantFilters(records);

  const {
    createDialog,
    editDialog,
    detailsDialog,
    deleteDialog,
    handleClone,
  } = useUserDialogs();

  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [invitePending, setInvitePending] = useState<string | null>(null);
  const [bulkActionLoading, setBulkActionLoading] = useState(false);

  // Mutations
  const createMutation = useMutation({
    mutationFn: createUser,
    onSuccess: (data, variables) => {
      toast({
        title: "User created",
        description: data.invite_sent
          ? "Invite email sent."
          : variables.send_invite
            ? "User created but the invite email could not be sent."
            : "Invite email not sent.",
      });
      createDialog.reset();
      void personalTenantsQuery.refetch();
      void queryClient.invalidateQueries({ queryKey: ["users", "directory"] });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to create user",
        description: "Double-check the inputs and try again.",
      });
    },
  });

  const sendInviteMutation = useMutation({
    mutationFn: (userId: string) => sendUserInvite(userId),
    onSuccess: (_, userId) => {
      toast({ title: "Invite email sent" });
      if (invitePending === userId) {
        setInvitePending(null);
      }
    },
    onError: (_, userId) => {
      toast({
        variant: "destructive",
        title: "Failed to send invite",
        description: "Check SMTP configuration and try again.",
      });
      if (invitePending === userId) {
        setInvitePending(null);
      }
    },
  });

  const statusMutation = useMutation({
    mutationFn: ({ tenantId, status }: { tenantId: string; status: TenantStatus }) =>
      updateTenantStatus({ tenantId, status }),
    onSuccess: () => {
      void personalTenantsQuery.refetch();
    },
  });

  // Handlers
  const handleCreateUser = useCallback(() => {
    const form = createDialog.form;
    if (!form.email.trim() || !form.name.trim()) {
      toast({ variant: "destructive", title: "Email and name are required" });
      return;
    }
    createMutation.mutate({
      email: form.email.trim(),
      name: form.name.trim(),
      password: form.password.trim() || undefined,
      send_invite: form.sendInvite,
    });
  }, [createDialog.form, createMutation, toast]);

  const handleEditUser = useCallback(() => {
    const record = editDialog.record;
    const form = editDialog.form;
    if (!record) return;

    if (form.status !== record.status) {
      statusMutation.mutate(
        { tenantId: record.tenant_id, status: form.status },
        {
          onSuccess: () => {
            toast({ title: "User status updated" });
            editDialog.close();
          },
          onError: () => {
            toast({
              variant: "destructive",
              title: "Failed to update status",
            });
          },
        },
      );
    } else {
      editDialog.close();
    }
  }, [editDialog, statusMutation, toast]);

  const handleSendInvite = useCallback(
    (userId: string) => {
      setInvitePending(userId);
      sendInviteMutation.mutate(userId);
    },
    [sendInviteMutation],
  );

  const handleBulkActivate = useCallback(async () => {
    setBulkActionLoading(true);
    const selectedRecords = records.filter((r) => selectedIds.has(r.tenant_id));
    try {
      for (const record of selectedRecords) {
        if (record.status !== "active") {
          await updateTenantStatus({ tenantId: record.tenant_id, status: "active" });
        }
      }
      toast({ title: `${selectedRecords.length} user(s) activated` });
      setSelectedIds(new Set());
      void personalTenantsQuery.refetch();
    } catch {
      toast({ variant: "destructive", title: "Failed to activate users" });
    } finally {
      setBulkActionLoading(false);
    }
  }, [records, selectedIds, toast, personalTenantsQuery]);

  const handleBulkSuspend = useCallback(async () => {
    setBulkActionLoading(true);
    const selectedRecords = records.filter((r) => selectedIds.has(r.tenant_id));
    try {
      for (const record of selectedRecords) {
        if (record.status !== "suspended") {
          await updateTenantStatus({ tenantId: record.tenant_id, status: "suspended" });
        }
      }
      toast({ title: `${selectedRecords.length} user(s) suspended` });
      setSelectedIds(new Set());
      void personalTenantsQuery.refetch();
    } catch {
      toast({ variant: "destructive", title: "Failed to suspend users" });
    } finally {
      setBulkActionLoading(false);
    }
  }, [records, selectedIds, toast, personalTenantsQuery]);

  const handleBulkDelete = useCallback(() => {
    const selectedRecords = records.filter((r) => selectedIds.has(r.tenant_id));
    deleteDialog.openWith(selectedRecords);
  }, [records, selectedIds, deleteDialog]);

  const handleConfirmDelete = useCallback(() => {
    // Note: Delete API not implemented - show message
    toast({
      variant: "destructive",
      title: "Delete not available",
      description: "User deletion is not yet supported via the API.",
    });
    deleteDialog.close();
    setSelectedIds(new Set());
  }, [toast, deleteDialog]);

  const handleViewDetails = useCallback(
    (record: PersonalTenantRecord) => {
      detailsDialog.openWith(record);
    },
    [detailsDialog],
  );

  const handleEditFromDetails = useCallback(
    (record: PersonalTenantRecord) => {
      detailsDialog.close();
      editDialog.openWith(record);
    },
    [detailsDialog, editDialog],
  );

  // Stats
  const activeCount = records.filter((r) => r.status === "active").length;
  const suspendedCount = records.filter((r) => r.status === "suspended").length;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Users"
        description="Personal tenants seeded for each user. Audit defaults, usage, and budget consumption."
        actions={
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={() => personalTenantsQuery.refetch()}
              loading={personalTenantsQuery.isFetching}
              title="Refresh"
            >
              <RefreshCcw className="h-4 w-4" />
            </Button>
            <Button onClick={() => createDialog.setOpen(true)}>
              <UserPlus className="h-4 w-4" />
              New User
            </Button>
          </div>
        }
      />

      {/* Summary stats */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Users</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{records.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active</CardTitle>
            <div className="h-2 w-2 rounded-full bg-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{activeCount}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Suspended</CardTitle>
            <div className="h-2 w-2 rounded-full bg-amber-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{suspendedCount}</div>
          </CardContent>
        </Card>
      </div>

      {/* User directory */}
      <UserDirectoryCard
        records={records}
        filteredRecords={filteredRecords}
        searchTerm={searchTerm}
        setSearchTerm={setSearchTerm}
        statusFilter={statusFilter}
        setStatusFilter={setStatusFilter}
        isLoading={personalTenantsQuery.isLoading}
        selectedIds={selectedIds}
        onSelectionChange={setSelectedIds}
        onView={handleViewDetails}
        onEdit={(record) => editDialog.openWith(record)}
        onClone={handleClone}
        onDelete={(record) => deleteDialog.openWith([record])}
        onSendInvite={handleSendInvite}
        invitePending={invitePending}
      />

      {/* Bulk action bar */}
      <BulkUserActionBar
        selectedCount={selectedIds.size}
        onActivate={handleBulkActivate}
        onSuspend={handleBulkSuspend}
        onDelete={handleBulkDelete}
        onClear={() => setSelectedIds(new Set())}
        isLoading={bulkActionLoading}
      />

      {/* Create dialog */}
      <UserEditorDialog
        open={createDialog.open}
        onOpenChange={createDialog.setOpen}
        form={createDialog.form}
        onChange={createDialog.setForm}
        onSubmit={handleCreateUser}
        loading={createMutation.isPending}
        mode="create"
      />

      {/* Edit dialog */}
      <UserEditorDialog
        open={editDialog.open}
        onOpenChange={(open) => {
          if (!open) editDialog.close();
          else editDialog.setOpen(open);
        }}
        form={editDialog.form}
        onChange={editDialog.setForm}
        onSubmit={handleEditUser}
        loading={statusMutation.isPending}
        mode="edit"
        record={editDialog.record}
      />

      {/* Details dialog */}
      <UserDetailsDialog
        open={detailsDialog.open}
        onOpenChange={(open) => {
          if (!open) detailsDialog.close();
          else detailsDialog.setOpen(open);
        }}
        record={detailsDialog.record}
        onEdit={handleEditFromDetails}
      />

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteDialog.open} onOpenChange={deleteDialog.setOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete user{deleteDialog.records.length > 1 ? "s" : ""}?</AlertDialogTitle>
            <AlertDialogDescription>
              {deleteDialog.records.length === 1
                ? `This will permanently delete ${deleteDialog.records[0]?.user_name || deleteDialog.records[0]?.user_email} and all associated data.`
                : `This will permanently delete ${deleteDialog.records.length} users and all associated data.`}
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => deleteDialog.close()}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
