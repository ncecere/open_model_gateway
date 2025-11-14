import { useMemo, useState, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { MoreHorizontal, Eye, Settings2 } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  useTenantSummaryQuery,
  useUserTenantsQuery,
} from "../hooks/useUserData";
import { Separator } from "@/components/ui/separator";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  listTenantMemberships,
  inviteTenantMember,
  removeTenantMember,
  type MembershipRole,
  type TenantMembership,
} from "@/api/user/tenants";
import { useToast } from "@/hooks/use-toast";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const MANAGEABLE_ROLES: MembershipRole[] = ["owner", "admin", "viewer", "user"];

export function UserTenantsPage() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const { data: tenants, isLoading } = useUserTenantsQuery();
  const allTenants = tenants ?? [];
  const filteredTenants = useMemo(
    () => allTenants.filter((tenant) => !tenant.is_personal),
    [allTenants],
  );
  const [selectedTenant, setSelectedTenant] = useState<string | undefined>(undefined);
  const [detailOpen, setDetailOpen] = useState(false);
  const summaryQuery = useTenantSummaryQuery(detailOpen ? selectedTenant : undefined);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<MembershipRole>("user");
  const [invitePassword, setInvitePassword] = useState("");

  useEffect(() => {
    if (!detailOpen) {
      setInviteEmail("");
      setInviteRole("user");
      setInvitePassword("");
    }
  }, [detailOpen]);

  const openTenantDetails = (tenantId: string) => {
    setSelectedTenant(tenantId);
    setDetailOpen(true);
  };

  const canManageMembers =
    summaryQuery.data &&
    (summaryQuery.data.role === "owner" || summaryQuery.data.role === "admin");

  const membershipsQuery = useQuery({
    queryKey: ["user-tenant-memberships", selectedTenant],
    queryFn: () =>
      selectedTenant ? listTenantMemberships(selectedTenant) : Promise.resolve([]),
    enabled: Boolean(detailOpen && selectedTenant && canManageMembers),
  });

  const inviteMutation = useMutation({
    mutationFn: () => {
      if (!selectedTenant) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return inviteTenantMember(selectedTenant, {
        email: inviteEmail,
        role: inviteRole,
        password: invitePassword || undefined,
      });
    },
    onSuccess: () => {
      toast({ title: "Member invited" });
      queryClient.invalidateQueries({
        queryKey: ["user-tenant-memberships", selectedTenant],
      });
      setInviteEmail("");
      setInvitePassword("");
      setInviteRole("user");
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to invite member",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const removeMutation = useMutation({
    mutationFn: (userId: string) => {
      if (!selectedTenant) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return removeTenantMember(selectedTenant, userId);
    },
    onSuccess: () => {
      toast({ title: "Membership removed" });
      queryClient.invalidateQueries({
        queryKey: ["user-tenant-memberships", selectedTenant],
      });
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to remove member",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const totalMemberships = allTenants.length;
  const activeMemberships = allTenants.filter((tenant) => tenant.status === "active").length;
  const managedMemberships = allTenants.filter((tenant) =>
    tenant.role === "owner" || tenant.role === "admin",
  ).length;

  return (
    <div className="space-y-6">
      <header className="space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight">Tenants</h1>
        <p className="text-sm text-muted-foreground">
          Owners and admins can manage memberships, invite teammates, and review tenant budgets from here.
        </p>
      </header>

      <section className="grid gap-4 md:grid-cols-3">
        <OverviewCard label="Memberships" value={totalMemberships} help="Total tenants you belong to" />
        <OverviewCard label="Active" value={activeMemberships} help="Tenants currently active" />
        <OverviewCard label="Managed" value={managedMemberships} help="Tenants where you are owner/admin" />
      </section>

      <Card>
        <CardHeader className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <CardTitle>Tenant directory</CardTitle>
            <p className="text-xs text-muted-foreground">
              {filteredTenants.length} membership{filteredTenants.length === 1 ? "" : "s"}
            </p>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(3)].map((_, idx) => (
                <div key={idx} className="h-12 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : filteredTenants.length ? (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-xs uppercase text-muted-foreground">
                    <th className="pb-3 font-medium">Name</th>
                    <th className="pb-3 font-medium">Status</th>
                    <th className="pb-3 font-medium">Role</th>
                    <th className="pb-3 font-medium text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredTenants.map((tenant) => (
                    <tr key={tenant.tenant_id} className="border-t text-sm">
                      <td className="py-3 font-medium">{tenant.name}</td>
                      <td className="py-3">
                        <Badge variant={tenant.status === "active" ? "secondary" : "outline"}>
                          {tenant.status}
                        </Badge>
                      </td>
                      <td className="py-3 capitalize">{tenant.role}</td>
                      <td className="py-3 text-right">
                        <TenantActions
                          role={tenant.role}
                          disabled={tenant.is_personal}
                          onManage={() => openTenantDetails(tenant.tenant_id)}
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              You are not part of any shared tenants.
            </p>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={detailOpen}
        onOpenChange={(open) => {
          setDetailOpen(open);
          if (!open) {
            setSelectedTenant(undefined);
          }
        }}
      >
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>
              Tenant details
              {summaryQuery.data ? ` · ${summaryQuery.data.name}` : ""}
            </DialogTitle>
          </DialogHeader>
          {summaryQuery.isLoading || !summaryQuery.data ? (
            <p className="text-sm text-muted-foreground">Loading tenant details…</p>
          ) : (
            <div className="space-y-6 text-sm">
              <div className="grid gap-4 md:grid-cols-3">
                <DetailStat label="Status" value={summaryQuery.data.status} />
                <DetailStat label="Role" value={summaryQuery.data.role} />
                <DetailStat label="Created" value={new Date(summaryQuery.data.created_at).toLocaleDateString()} />
              </div>
              <Separator />
              <div className="grid gap-4 md:grid-cols-3">
                <DetailStat
                  label="Budget limit"
                  value={`$${summaryQuery.data.budget.limit_usd.toFixed(2)}`}
                />
                <DetailStat
                  label="Remaining"
                  value={`$${summaryQuery.data.budget.remaining_usd.toFixed(2)}`}
                />
                <DetailStat
                  label="Refresh schedule"
                  value={summaryQuery.data.budget.refresh_schedule}
                />
              </div>
              {canManageMembers ? (
                <section className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-base font-semibold">Memberships</p>
                      <p className="text-xs text-muted-foreground">
                        Invite teammates or adjust roles. Owners can grant owner access; admins may manage other roles.
                      </p>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => membershipsQuery.refetch()}
                      disabled={membershipsQuery.isLoading}
                    >
                      Refresh
                    </Button>
                  </div>
                  {membershipsQuery.isLoading ? (
                    <div className="space-y-2">
                      {[...Array(3)].map((_, idx) => (
                        <div key={idx} className="h-10 animate-pulse rounded bg-muted" />
                      ))}
                    </div>
                  ) : membershipsQuery.data && membershipsQuery.data.length ? (
                    <div className="overflow-x-auto">
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="text-left text-xs uppercase text-muted-foreground">
                            <th className="pb-2 font-medium">Email</th>
                            <th className="pb-2 font-medium">Role</th>
                            <th className="pb-2 font-medium">Joined</th>
                            <th className="pb-2 font-medium text-right">Actions</th>
                          </tr>
                        </thead>
                        <tbody>
                          {membershipsQuery.data.map((member) => (
                            <MemberRow
                              key={member.user_id}
                              member={member}
                              canEdit={Boolean(summaryQuery.data && summaryQuery.data.role === "owner") || (summaryQuery.data?.role === "admin" && member.role !== "owner")}
                              onRemove={(id) => removeMutation.mutate(id)}
                              removing={removeMutation.isPending && removeMutation.variables === member.user_id}
                            />
                          ))}
                        </tbody>
                      </table>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      No members yet.
                    </p>
                  )}
                  <InviteForm
                    email={inviteEmail}
                    role={inviteRole}
                    password={invitePassword}
                    onEmailChange={setInviteEmail}
                    onRoleChange={(value) => setInviteRole(value as MembershipRole)}
                    onPasswordChange={setInvitePassword}
                    onSubmit={(event) => {
                      event.preventDefault();
                      inviteMutation.mutate();
                    }}
                    submitting={inviteMutation.isPending}
                  />
                </section>
              ) : null}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function OverviewCard({ label, value, help }: { label: string; value: number; help: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm text-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold">{value}</p>
        <p className="text-xs text-muted-foreground">{help}</p>
      </CardContent>
    </Card>
  );
}

function DetailStat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-muted-foreground">{label}</p>
      <p className="font-semibold">{value}</p>
    </div>
  );
}

function MemberRow({
  member,
  canEdit,
  onRemove,
  removing,
}: {
  member: TenantMembership;
  canEdit: boolean;
  onRemove: (userId: string) => void;
  removing: boolean;
}) {
  return (
    <tr className="border-t text-sm">
      <td className="py-2">
        <div className="flex flex-col">
          <span className="font-medium">{member.email}</span>
          {member.self ? (
            <span className="text-xs text-muted-foreground">You</span>
          ) : null}
        </div>
      </td>
      <td className="py-2 capitalize">{member.role}</td>
      <td className="py-2 text-sm text-muted-foreground">
        {new Date(member.created_at).toLocaleDateString()}
      </td>
      <td className="py-2 text-right">
        <Button
          variant="ghost"
          size="sm"
          disabled={!canEdit || member.self || removing}
          onClick={() => onRemove(member.user_id)}
        >
          Remove
        </Button>
      </td>
    </tr>
  );
}

function InviteForm({
  email,
  role,
  password,
  onEmailChange,
  onRoleChange,
  onPasswordChange,
  onSubmit,
  submitting,
}: {
  email: string;
  role: MembershipRole;
  password: string;
  onEmailChange: (value: string) => void;
  onRoleChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
  submitting: boolean;
}) {
  return (
    <form onSubmit={onSubmit} className="space-y-3 rounded border p-3">
      <div className="grid gap-3 md:grid-cols-3">
        <div className="md:col-span-2">
          <Label htmlFor="invite-email">Email</Label>
          <Input
            id="invite-email"
            type="email"
            value={email}
            onChange={(event) => onEmailChange(event.target.value)}
            placeholder="name@example.com"
            required
          />
        </div>
        <div>
          <Label htmlFor="invite-role">Role</Label>
          <Select value={role} onValueChange={onRoleChange}>
            <SelectTrigger id="invite-role">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MANAGEABLE_ROLES.map((value) => (
                <SelectItem key={value} value={value}>
                  {value.charAt(0).toUpperCase() + value.slice(1)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <div className="md:col-span-2">
          <Label htmlFor="invite-password">Password (optional)</Label>
          <Input
            id="invite-password"
            type="password"
            value={password}
            onChange={(event) => onPasswordChange(event.target.value)}
            placeholder="Set an initial password"
          />
        </div>
        <div className="flex items-end">
          <Button type="submit" disabled={submitting} className="w-full">
            {submitting ? "Inviting…" : "Send invite"}
          </Button>
        </div>
      </div>
    </form>
  );
}

interface TenantActionsProps {
  role: string;
  onManage: () => void;
  disabled?: boolean;
}

function TenantActions({ role, onManage, disabled }: TenantActionsProps) {
  const canEdit = !disabled && (role === "owner" || role === "admin");
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" disabled={disabled}>
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={onManage} disabled={disabled}>
          <Eye className="mr-2 h-4 w-4" /> Manage
        </DropdownMenuItem>
        {canEdit ? (
          <DropdownMenuItem disabled>
            <Settings2 className="mr-2 h-4 w-4" /> Admin settings
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
