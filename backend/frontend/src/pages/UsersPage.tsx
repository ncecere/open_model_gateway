import { useCallback, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Eye, Mail, MoreHorizontal, RefreshCcw, UserPlus } from "lucide-react";
import { PageHeader } from "@/components/layouts";

import { type PersonalTenantRecord, type TenantStatus } from "@/api/tenants";
import {
  createUser,
  getUserTenants,
  sendUserInvite,
  type UserTenantMembership,
} from "@/api/users";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { BudgetMeter } from "@/ui/kit/BudgetMeter";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DataTable, type DataTableColumn } from "@/ui/kit/DataTable";
import {
  usePersonalTenantsQuery,
  usePersonalTenantFilters,
} from "@/features/users/hooks/usePersonalTenants";
import { useToast } from "@/hooks/use-toast";

export function UsersPage() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const personalTenantsQuery = usePersonalTenantsQuery();
  const records: PersonalTenantRecord[] = personalTenantsQuery.data?.personal_tenants ?? [];
  const { searchTerm, setSearchTerm, statusFilter, setStatusFilter, filteredRecords } =
    usePersonalTenantFilters(records);
  const [selectedUser, setSelectedUser] = useState<PersonalTenantRecord | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [newUserName, setNewUserName] = useState("");
  const [newUserEmail, setNewUserEmail] = useState("");
  const [newUserPassword, setNewUserPassword] = useState("");
  const [newUserInvite, setNewUserInvite] = useState(true);
  const [inviteTarget, setInviteTarget] = useState<string | null>(null);

  const userTenantsQuery = useQuery<UserTenantMembership[]>({
    queryKey: ["user-tenants", selectedUser?.user_id],
    queryFn: () => getUserTenants(selectedUser!.user_id),
    enabled: Boolean(selectedUser?.user_id),
  });
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
      setCreateOpen(false);
      setNewUserEmail("");
      setNewUserName("");
      setNewUserPassword("");
      setNewUserInvite(true);
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
      if (inviteTarget === userId) {
        setInviteTarget(null);
      }
    },
    onError: (_, userId) => {
      toast({
        variant: "destructive",
        title: "Failed to send invite",
        description: "Check SMTP configuration and try again.",
      });
      if (inviteTarget === userId) {
        setInviteTarget(null);
      }
    },
  });

  const handleCreateUser = () => {
    if (!newUserEmail.trim() || !newUserName.trim()) {
      toast({ variant: "destructive", title: "Email and name are required" });
      return;
    }
    createMutation.mutate({
      email: newUserEmail.trim(),
      name: newUserName.trim(),
      password: newUserPassword.trim() || undefined,
      send_invite: newUserInvite,
    });
  };

  const handleSendInvite = useCallback(
    (userId: string) => {
      setInviteTarget(userId);
      sendInviteMutation.mutate(userId);
    },
    [sendInviteMutation],
  );

  const personalTenantColumns = useMemo(() => {
    return [
      {
        header: "User",
        cell: (record: PersonalTenantRecord) => (
          <div className="flex flex-col">
            <span className="font-medium">{record.user_name}</span>
            <span className="text-xs text-muted-foreground">{record.user_email}</span>
          </div>
        ),
      },
      {
        header: "Status",
        cell: (record: PersonalTenantRecord) => (
          <span className="capitalize">{record.status}</span>
        ),
      },
      {
        header: "Budget",
        cell: (record: PersonalTenantRecord) => (
          <BudgetMeter
            used={record.budget_used_usd ?? 0}
            limit={record.budget_limit_usd ?? 0}
            warningThreshold={record.warning_threshold ?? 0.8}
          />
        ),
        cellClassName: "min-w-[220px]",
      },
      {
        header: "Tenants",
        cell: (record: PersonalTenantRecord) => (
          <Badge variant="secondary">{record.membership_count ?? 1} tenants</Badge>
        ),
      },
      {
        header: "Created",
        cell: (record: PersonalTenantRecord) => (
          <span className="text-sm text-muted-foreground">
            {new Date(record.created_at).toLocaleDateString()}
          </span>
        ),
      },
      {
        header: "Tenant ID",
        cell: (record: PersonalTenantRecord) => (
          <span className="font-mono text-xs">{record.tenant_id}</span>
        ),
      },
      {
        header: "Actions",
        headerClassName: "text-right",
        cellClassName: "text-right",
        cell: (record: PersonalTenantRecord) => (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" aria-label="Open user actions">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>Actions</DropdownMenuLabel>
                <DropdownMenuItem onSelect={() => setSelectedUser(record)}>
                  <Eye className="mr-2 h-4 w-4" /> View details
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={
                    (inviteTarget !== null && inviteTarget !== record.user_id) ||
                    sendInviteMutation.isPending
                  }
                  onSelect={(event) => {
                    event.preventDefault();
                    handleSendInvite(record.user_id);
                  }}
                >
                  <Mail className="mr-2 h-4 w-4" /> Send invite email
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ),
      },
    ];
  }, [handleSendInvite, inviteTarget, sendInviteMutation.isPending]);

  const membershipColumns: DataTableColumn<UserTenantMembership>[] = useMemo(
    () => [
      {
        header: "Tenant",
        cell: (membership) => membership.tenant_name,
      },
      {
        header: "Role",
        cell: (membership) => <span className="capitalize">{membership.role}</span>,
      },
      {
        header: "Status",
        cell: (membership) => <span className="capitalize">{membership.status}</span>,
      },
      {
        header: "Joined",
        cell: (membership) =>
          new Date(membership.joined_at).toLocaleDateString(),
      },
    ],
    [],
  );

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
            <Button onClick={() => setCreateOpen(true)}>
              <UserPlus className="h-4 w-4" />
              New User
            </Button>
          </div>
        }
      />

      <Card>
        <CardHeader className="gap-4">
          <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
            <div>
              <CardTitle>Personal tenants</CardTitle>
              <p className="text-sm text-muted-foreground">
                {records.length} users with seeded personal tenants
              </p>
            </div>
            <div className="flex w-full flex-col gap-2 md:max-w-xl md:flex-row">
              <Input
                placeholder="Search by name or email"
                value={searchTerm}
                onChange={(event) => setSearchTerm(event.target.value)}
                className="w-full"
              />
              <Select
                value={statusFilter}
                onValueChange={(value) =>
                  setStatusFilter(value as "all" | TenantStatus)
                }
              >
                <SelectTrigger className="w-full md:w-[200px]">
                  <SelectValue placeholder="Filter status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="suspended">Suspended</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <DataTable
            data={filteredRecords}
            columns={personalTenantColumns}
            getKey={(record) => record.tenant_id}
            isLoading={personalTenantsQuery.isLoading}
            emptyState="No users match the current filters."
          />
        </CardContent>
      </Card>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New user</DialogTitle>
            <DialogDescription>
              Create an account and optionally send an invite email.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="new-user-email">Email</Label>
              <Input
                id="new-user-email"
                placeholder="user@example.com"
                value={newUserEmail}
                onChange={(event) => setNewUserEmail(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new-user-name">Name</Label>
              <Input
                id="new-user-name"
                placeholder="Full name"
                value={newUserName}
                onChange={(event) => setNewUserName(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new-user-password">
                Password <span className="text-xs text-muted-foreground">(optional)</span>
              </Label>
              <Input
                id="new-user-password"
                type="password"
                placeholder="Set initial password"
                value={newUserPassword}
                onChange={(event) => setNewUserPassword(event.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Leave blank to rely on SSO or send invite instructions without a password.
              </p>
            </div>
            <div className="flex items-center justify-between rounded-md border px-3 py-2">
              <div>
                <Label className="text-sm">Send invite email</Label>
                <p className="text-xs text-muted-foreground">
                  Email the user with admin/user portal links after creation.
                </p>
              </div>
              <Switch
                checked={newUserInvite}
                onCheckedChange={(checked) => setNewUserInvite(checked)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)} disabled={createMutation.isPending}>
              Cancel
            </Button>
            <Button onClick={handleCreateUser} disabled={createMutation.isPending}>
              {createMutation.isPending ? "Creating…" : "Create user"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={Boolean(selectedUser)} onOpenChange={(open) => !open && setSelectedUser(null)}>
        <DialogContent className="max-w-3xl">
          {selectedUser ? (
            <>
              <DialogHeader>
                <DialogTitle>{selectedUser.user_name || selectedUser.user_email}</DialogTitle>
                <DialogDescription>{selectedUser.user_email}</DialogDescription>
              </DialogHeader>
              <div className="space-y-6 py-2">
                <div className="grid gap-4 md:grid-cols-3">
                  <StatCard
                    label="Status"
                    value={selectedUser.status}
                  />
                  <StatCard
                    label="Personal tenant created"
                    value={new Date(selectedUser.created_at).toLocaleDateString()}
                  />
                  <StatCard
                    label="Tenants"
                    value={(selectedUser.membership_count ?? 1).toString()}
                  />
                </div>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm font-medium text-muted-foreground">
                      Budget overview
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <BudgetMeter
                      used={selectedUser.budget_used_usd ?? 0}
                      limit={selectedUser.budget_limit_usd ?? 0}
                      warningThreshold={selectedUser.warning_threshold ?? 0.8}
                    />
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader>
                    <CardTitle>Tenant memberships</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <DataTable
                      data={userTenantsQuery.data ?? []}
                      columns={membershipColumns}
                      getKey={(membership) => `${membership.tenant_id}-${membership.role}`}
                      isLoading={userTenantsQuery.isLoading}
                      emptyState="No additional tenant memberships found."
                      dense
                    />
                  </CardContent>
                </Card>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setSelectedUser(null)}>
                  Close
                </Button>
              </DialogFooter>
            </>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {label}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold text-foreground">{value}</p>
      </CardContent>
    </Card>
  );
}
