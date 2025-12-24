import { useMemo, useState, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { BudgetMeter } from "@/ui/kit/BudgetMeter";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { MoreHorizontal, Eye, Settings2 } from "lucide-react";
import { Switch } from "@/components/ui/switch";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  listTenantMemberships,
  inviteTenantMember,
  removeTenantMember,
  updateTenantMemberBudget,
  getTenantLimits,
  updateTenantLimits,
  deleteTenantLimits,
  listTenantModels,
  attachTenantModel,
  detachTenantModel,
  type MembershipRole,
  type MemberBudget,
  type TenantMembership,
  type TenantModelEntry,
} from "@/api/user/tenants";
import { searchDirectoryUsers, type DirectoryUser } from "@/api/user/directory";
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
  const [editingMember, setEditingMember] = useState<TenantMembership | null>(null);
  const [budgetForm, setBudgetForm] = useState({
    budgetUSD: "",
    warningThreshold: "",
    tokenCap: "",
  });
  const [limitForm, setLimitForm] = useState({
    requestsPerMinute: "",
    tokensPerMinute: "",
    parallelRequests: "",
  });

  useEffect(() => {
    if (!detailOpen) {
      setInviteEmail("");
      setInviteRole("user");
    }
  }, [detailOpen]);

  useEffect(() => {
    if (!editingMember) {
      setBudgetForm({ budgetUSD: "", warningThreshold: "", tokenCap: "" });
      return;
    }
    const budget = editingMember.budget ?? {};
    setBudgetForm({
      budgetUSD: budget.budget_usd?.toString() ?? "",
      warningThreshold: budget.warning_threshold?.toString() ?? "",
      tokenCap: budget.token_cap?.toString() ?? "",
    });
  }, [editingMember]);

  const openTenantDetails = (tenantId: string) => {
    setSelectedTenant(tenantId);
    setDetailOpen(true);
  };

  const canManageMembers =
    summaryQuery.data &&
    (summaryQuery.data.role === "owner" || summaryQuery.data.role === "admin");
  const canManageLimits = Boolean(canManageMembers);
  const canManageModels = Boolean(canManageMembers);

  const membershipsQuery = useQuery({
    queryKey: ["user-tenant-memberships", selectedTenant],
    queryFn: () =>
      selectedTenant ? listTenantMemberships(selectedTenant) : Promise.resolve([]),
    enabled: Boolean(detailOpen && selectedTenant && canManageMembers),
  });

  const modelsQuery = useQuery({
    queryKey: ["user-tenant-models", selectedTenant],
    queryFn: () => (selectedTenant ? listTenantModels(selectedTenant) : Promise.resolve([])),
    enabled: Boolean(detailOpen && selectedTenant),
  });

  const limitsQuery = useQuery({
    queryKey: ["user-tenant-limits", selectedTenant],
    queryFn: () => (selectedTenant ? getTenantLimits(selectedTenant) : Promise.reject()),
    enabled: Boolean(detailOpen && selectedTenant),
  });

  useEffect(() => {
    if (!limitsQuery.data) return;
    setLimitForm({
      requestsPerMinute: limitsQuery.data.effective.requests_per_minute.toString(),
      tokensPerMinute: limitsQuery.data.effective.tokens_per_minute.toString(),
      parallelRequests: limitsQuery.data.effective.parallel_requests.toString(),
    });
  }, [limitsQuery.data]);

  const inviteMutation = useMutation({
    mutationFn: () => {
      if (!selectedTenant) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return inviteTenantMember(selectedTenant, {
        email: inviteEmail,
        role: inviteRole,
      });
    },
    onSuccess: () => {
      toast({ title: "Member added" });
      queryClient.invalidateQueries({
        queryKey: ["user-tenant-memberships", selectedTenant],
      });
      setInviteEmail("");
      setInviteRole("user");
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to add member",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const suggestionQuery = useQuery({
    queryKey: ["user-directory", inviteEmail],
    queryFn: () => searchDirectoryUsers(inviteEmail.trim()),
    enabled:
      Boolean(detailOpen && selectedTenant && canManageMembers) &&
      inviteEmail.trim().length >= 2,
    staleTime: 30_000,
  });
  const directorySuggestions = suggestionQuery.data ?? [];

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

  const attachModelMutation = useMutation({
    mutationFn: (alias: string) => {
      if (!selectedTenant) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return attachTenantModel(selectedTenant, alias);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["user-tenant-models", selectedTenant],
      });
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to attach model",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const detachModelMutation = useMutation({
    mutationFn: (alias: string) => {
      if (!selectedTenant) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return detachTenantModel(selectedTenant, alias);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["user-tenant-models", selectedTenant],
      });
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to detach model",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const budgetMutation = useMutation({
    mutationFn: (payload: MemberBudgetUpdatePayload) => {
      if (!selectedTenant || !editingMember) {
        return Promise.reject(new Error("member not selected"));
      }
      return updateTenantMemberBudget(selectedTenant, editingMember.user_id, payload);
    },
    onSuccess: () => {
      toast({ title: "Member budget updated" });
      queryClient.invalidateQueries({
        queryKey: ["user-tenant-memberships", selectedTenant],
      });
      setEditingMember(null);
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to update budget",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const limitMutation = useMutation({
    mutationFn: (payload: TenantLimitPayloadRequest) => {
      if (!selectedTenant) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return updateTenantLimits(selectedTenant, payload);
    },
    onSuccess: () => {
      toast({ title: "Tenant limits updated" });
      queryClient.invalidateQueries({
        queryKey: ["user-tenant-limits", selectedTenant],
      });
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to update limits",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const clearLimitsMutation = useMutation({
    mutationFn: () => {
      if (!selectedTenant) {
        return Promise.reject(new Error("tenant not selected"));
      }
      return deleteTenantLimits(selectedTenant);
    },
    onSuccess: () => {
      toast({ title: "Tenant limits reset" });
      queryClient.invalidateQueries({
        queryKey: ["user-tenant-limits", selectedTenant],
      });
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to reset limits",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const totalMemberships = filteredTenants.length;
  const activeMemberships = filteredTenants.filter((tenant) => tenant.status === "active").length;
  const managedMemberships = filteredTenants.filter((tenant) =>
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
                    <th className="pb-3 font-medium min-w-[220px]">Budget</th>
                    <th className="pb-3 font-medium min-w-[120px] pl-4">Role</th>
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
                      <td className="py-3 pr-4 min-w-[220px]">
                        <BudgetMeter
                          used={tenant.budget_used_usd ?? 0}
                          limit={tenant.budget_limit_usd ?? 0}
                          warningThreshold={tenant.warning_threshold ?? 0.8}
                        />
                      </td>
                      <td className="py-3 min-w-[120px] pl-4 capitalize">{tenant.role}</td>
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
        <DialogContent className="flex h-[70vh] max-w-3xl flex-col overflow-hidden">
          <DialogHeader>
            <DialogTitle>
              Tenant details
              {summaryQuery.data ? ` · ${summaryQuery.data.name}` : ""}
            </DialogTitle>
          </DialogHeader>
          {summaryQuery.isLoading || !summaryQuery.data ? (
            <p className="text-sm text-muted-foreground">Loading tenant details…</p>
          ) : (
            <Tabs defaultValue="overview" className="flex h-full min-h-0 flex-col space-y-4 text-sm">
              <TabsList className="w-fit">
                <TabsTrigger value="overview">Overview</TabsTrigger>
                {canManageModels ? <TabsTrigger value="models">Models</TabsTrigger> : null}
                {canManageMembers ? <TabsTrigger value="members">Members</TabsTrigger> : null}
              </TabsList>
              <TabsContent value="overview" className="flex-1 min-h-0 space-y-6 overflow-y-auto pr-1">
                <div className="grid gap-4 md:grid-cols-3">
                  <DetailStat label="Status" value={summaryQuery.data.status} />
                  <DetailStat label="Role" value={summaryQuery.data.role} />
                  <DetailStat label="Created" value={new Date(summaryQuery.data.created_at).toLocaleDateString()} />
                </div>
                <Separator />
                <div className="grid gap-4 pt-2 md:grid-cols-3">
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
                <Separator />
                <section className="space-y-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-base font-semibold">Tenant limits</p>
                      <p className="text-xs text-muted-foreground">
                        Limits apply across the tenant and cannot exceed the global ceiling.
                      </p>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => limitsQuery.refetch()}
                      disabled={limitsQuery.isLoading}
                    >
                      Refresh
                    </Button>
                  </div>
                  {limitsQuery.isLoading ? (
                    <div className="space-y-2">
                      {[...Array(2)].map((_, idx) => (
                        <div key={idx} className="h-10 animate-pulse rounded bg-muted" />
                      ))}
                    </div>
                  ) : limitsQuery.data ? (
                    <div className="space-y-3 rounded border p-3">
                      <div className="grid gap-3 md:grid-cols-3">
                        <DetailStat
                          label="Effective RPM"
                          value={limitsQuery.data.effective.requests_per_minute.toString()}
                        />
                        <DetailStat
                          label="Effective TPM"
                          value={limitsQuery.data.effective.tokens_per_minute.toString()}
                        />
                        <DetailStat
                          label="Effective concurrency"
                          value={limitsQuery.data.effective.parallel_requests.toString()}
                        />
                      </div>
                      <div className="grid gap-3 md:grid-cols-3 text-xs text-muted-foreground">
                        <span>Ceiling RPM: {limitsQuery.data.ceiling.requests_per_minute}</span>
                        <span>Ceiling TPM: {limitsQuery.data.ceiling.tokens_per_minute}</span>
                        <span>Ceiling concurrency: {limitsQuery.data.ceiling.parallel_requests}</span>
                      </div>
                      {canManageLimits ? (
                        <div className="space-y-3">
                          <Separator />
                          <div className="grid gap-3 md:grid-cols-3">
                            <div>
                              <Label htmlFor="tenant-limit-rpm">RPM</Label>
                              <Input
                                id="tenant-limit-rpm"
                                type="number"
                                min={1}
                                value={limitForm.requestsPerMinute}
                                onChange={(event) =>
                                  setLimitForm((prev) => ({
                                    ...prev,
                                    requestsPerMinute: event.target.value,
                                  }))
                                }
                              />
                            </div>
                            <div>
                              <Label htmlFor="tenant-limit-tpm">TPM</Label>
                              <Input
                                id="tenant-limit-tpm"
                                type="number"
                                min={1}
                                value={limitForm.tokensPerMinute}
                                onChange={(event) =>
                                  setLimitForm((prev) => ({
                                    ...prev,
                                    tokensPerMinute: event.target.value,
                                  }))
                                }
                              />
                            </div>
                            <div>
                              <Label htmlFor="tenant-limit-parallel">Concurrency</Label>
                              <Input
                                id="tenant-limit-parallel"
                                type="number"
                                min={1}
                                value={limitForm.parallelRequests}
                                onChange={(event) =>
                                  setLimitForm((prev) => ({
                                    ...prev,
                                    parallelRequests: event.target.value,
                                  }))
                                }
                              />
                            </div>
                          </div>
                          <div className="flex flex-wrap justify-end gap-2">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => clearLimitsMutation.mutate()}
                              disabled={clearLimitsMutation.isPending}
                            >
                              Reset override
                            </Button>
                            <Button
                              size="sm"
                              onClick={() => {
                                const rpm = parseInt(limitForm.requestsPerMinute, 10);
                                const tpm = parseInt(limitForm.tokensPerMinute, 10);
                                const parallel = parseInt(limitForm.parallelRequests, 10);
                                if (!rpm || !tpm || !parallel) {
                                  toast({
                                    variant: "destructive",
                                    title: "Invalid limits",
                                    description: "All values must be positive numbers.",
                                  });
                                  return;
                                }
                                limitMutation.mutate({
                                  requests_per_minute: rpm,
                                  tokens_per_minute: tpm,
                                  parallel_requests: parallel,
                                });
                              }}
                              disabled={limitMutation.isPending}
                            >
                              Save limits
                            </Button>
                          </div>
                        </div>
                      ) : null}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">Limits unavailable.</p>
                  )}
                </section>
              </TabsContent>
              {canManageModels ? (
                <TabsContent value="models" className="flex-1 min-h-0 space-y-4 overflow-y-auto pr-1">
                  <section className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-base font-semibold">Model access</p>
                        <p className="text-xs text-muted-foreground">
                          Attach approved aliases to make them available for this tenant.
                        </p>
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => modelsQuery.refetch()}
                        disabled={modelsQuery.isLoading}
                      >
                        Refresh
                      </Button>
                    </div>
                    {modelsQuery.isLoading ? (
                      <div className="space-y-2">
                        {[...Array(3)].map((_, idx) => (
                          <div key={idx} className="h-10 animate-pulse rounded bg-muted" />
                        ))}
                      </div>
                    ) : modelsQuery.data && modelsQuery.data.length ? (
                      <div className="max-h-[48vh] overflow-x-auto overflow-y-auto rounded border">
                        <table className="w-full text-sm">
                          <thead className="sticky top-0 bg-muted/40 text-left text-xs uppercase text-muted-foreground">
                            <tr>
                              <th className="px-3 py-2 font-medium">Alias</th>
                              <th className="px-3 py-2 font-medium">Provider</th>
                              <th className="px-3 py-2 font-medium">Type</th>
                              <th className="px-3 py-2 font-medium">Status</th>
                              <th className="px-3 py-2 text-right font-medium">Access</th>
                            </tr>
                          </thead>
                          <tbody>
                            {modelsQuery.data.map((model) => (
                              <TenantModelRow
                                key={model.alias}
                                model={model}
                                canManage={canManageModels}
                                onToggle={(next) => {
                                  if (next) {
                                    attachModelMutation.mutate(model.alias);
                                  } else {
                                    detachModelMutation.mutate(model.alias);
                                  }
                                }}
                                isUpdating={
                                  attachModelMutation.isPending ||
                                  detachModelMutation.isPending
                                }
                              />
                            ))}
                          </tbody>
                        </table>
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground">
                        No approved models available yet.
                      </p>
                    )}
                  </section>
                </TabsContent>
              ) : null}
              {canManageMembers ? (
                <TabsContent value="members" className="flex-1 min-h-0 space-y-4 overflow-y-auto pr-1">
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
                              <th className="pb-2 font-medium">Budget</th>
                              <th className="pb-2 font-medium text-right">Actions</th>
                            </tr>
                          </thead>
                          <tbody>
                            {membershipsQuery.data.map((member) => (
                              <MemberRow
                                key={member.user_id}
                                member={member}
                                canEdit={Boolean(summaryQuery.data && summaryQuery.data.role === "owner") || (summaryQuery.data?.role === "admin" && member.role !== "owner")}
                                canEditBudget={canManageMembers}
                                onEditBudget={(target) => setEditingMember(target)}
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
                      suggestions={directorySuggestions}
                      loadingSuggestions={suggestionQuery.isFetching}
                      onEmailChange={setInviteEmail}
                      onRoleChange={(value) => setInviteRole(value as MembershipRole)}
                      onSuggestionSelect={(user) => setInviteEmail(user.email)}
                      onSubmit={(event) => {
                        event.preventDefault();
                        inviteMutation.mutate();
                      }}
                      submitting={inviteMutation.isPending}
                    />
                  </section>
                </TabsContent>
              ) : null}
            </Tabs>
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={Boolean(editingMember)} onOpenChange={(open) => !open && setEditingMember(null)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Update member budget</DialogTitle>
          </DialogHeader>
          {editingMember ? (
            <form
              className="space-y-3 text-sm"
              onSubmit={(event) => {
                event.preventDefault();
                const payload: MemberBudgetUpdatePayload = {};
                if (budgetForm.budgetUSD.trim() !== "") {
                  payload.budget_usd = Number(budgetForm.budgetUSD);
                }
                if (budgetForm.warningThreshold.trim() !== "") {
                  payload.warning_threshold = Number(budgetForm.warningThreshold);
                }
                if (budgetForm.tokenCap.trim() !== "") {
                  payload.token_cap = Number(budgetForm.tokenCap);
                }
                if (Object.keys(payload).length === 0) {
                  toast({
                    variant: "destructive",
                    title: "No changes",
                    description: "Enter at least one budget value to update.",
                  });
                  return;
                }
                if (
                  (payload.budget_usd !== undefined && Number.isNaN(payload.budget_usd)) ||
                  (payload.warning_threshold !== undefined && Number.isNaN(payload.warning_threshold)) ||
                  (payload.token_cap !== undefined && Number.isNaN(payload.token_cap))
                ) {
                  toast({
                    variant: "destructive",
                    title: "Invalid budget values",
                    description: "Please enter valid numeric values.",
                  });
                  return;
                }
                budgetMutation.mutate(payload);
              }}
            >
              <div>
                <Label htmlFor="member-budget-usd">Budget USD</Label>
                <Input
                  id="member-budget-usd"
                  type="number"
                  min={0}
                  step="0.01"
                  value={budgetForm.budgetUSD}
                  onChange={(event) =>
                    setBudgetForm((prev) => ({ ...prev, budgetUSD: event.target.value }))
                  }
                />
              </div>
              <div>
                <Label htmlFor="member-warning">Warning threshold</Label>
                <Input
                  id="member-warning"
                  type="number"
                  min={0}
                  max={1}
                  step="0.05"
                  value={budgetForm.warningThreshold}
                  onChange={(event) =>
                    setBudgetForm((prev) => ({
                      ...prev,
                      warningThreshold: event.target.value,
                    }))
                  }
                />
              </div>
              <div>
                <Label htmlFor="member-token-cap">Token cap</Label>
                <Input
                  id="member-token-cap"
                  type="number"
                  min={0}
                  step="1"
                  value={budgetForm.tokenCap}
                  onChange={(event) =>
                    setBudgetForm((prev) => ({ ...prev, tokenCap: event.target.value }))
                  }
                />
              </div>
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setEditingMember(null)}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={budgetMutation.isPending}>
                  {budgetMutation.isPending ? "Saving..." : "Save"}
                </Button>
              </div>
            </form>
          ) : null}
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
  canEditBudget,
  onEditBudget,
  onRemove,
  removing,
}: {
  member: TenantMembership;
  canEdit: boolean;
  canEditBudget: boolean;
  onEditBudget: (member: TenantMembership) => void;
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
      <td className="py-2 text-sm text-muted-foreground">
        {renderMemberBudget(member.budget)}
      </td>
      <td className="py-2 text-right">
        <div className="flex justify-end gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={!canEditBudget}
            onClick={() => onEditBudget(member)}
          >
            Budget
          </Button>
          <Button
            variant="ghost"
            size="sm"
            disabled={!canEdit || member.self || removing}
            onClick={() => onRemove(member.user_id)}
          >
            Remove
          </Button>
        </div>
      </td>
    </tr>
  );
}

function InviteForm({
  email,
  role,
  suggestions,
  loadingSuggestions,
  onEmailChange,
  onRoleChange,
  onSuggestionSelect,
  onSubmit,
  submitting,
}: {
  email: string;
  role: MembershipRole;
  suggestions: DirectoryUser[];
  loadingSuggestions: boolean;
  onEmailChange: (value: string) => void;
  onRoleChange: (value: string) => void;
  onSuggestionSelect: (user: DirectoryUser) => void;
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
          <p className="text-xs text-muted-foreground">
            Only existing users can be added. Ask an administrator to create new accounts first.
          </p>
          <DirectorySuggestions
            query={email}
            loading={loadingSuggestions}
            suggestions={suggestions}
            onSelect={onSuggestionSelect}
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
      <div className="flex items-end justify-end">
        <Button type="submit" disabled={submitting} className="w-full md:w-auto">
          {submitting ? "Adding…" : "Add member"}
        </Button>
      </div>
    </form>
  );
}

function DirectorySuggestions({
  query,
  loading,
  suggestions,
  onSelect,
}: {
  query: string;
  loading: boolean;
  suggestions: DirectoryUser[];
  onSelect: (user: DirectoryUser) => void;
}) {
  const trimmed = query.trim();
  if (trimmed.length < 2) {
    return null;
  }
  if (loading) {
    return (
      <div className="mt-2 space-y-2">
        <Skeleton className="h-8 w-full" />
      </div>
    );
  }
  if (suggestions.length === 0) {
    return (
      <p className="mt-2 text-xs text-muted-foreground">
        No user matches “{trimmed}”. Create the account from the admin portal first.
      </p>
    );
  }
  return (
    <div className="mt-2 space-y-1 rounded border bg-muted/50 p-2 text-sm">
      <p className="text-xs font-medium uppercase text-muted-foreground">Suggestions</p>
      {suggestions.map((user) => (
        <button
          key={user.id}
          type="button"
          className="flex w-full flex-col rounded px-2 py-1 text-left transition hover:bg-background"
          onClick={() => onSelect(user)}
        >
          <span className="font-medium">{user.name || user.email}</span>
          <span className="text-xs text-muted-foreground">{user.email}</span>
        </button>
      ))}
    </div>
  );
}

export function TenantModelRow({
  model,
  canManage,
  onToggle,
  isUpdating,
}: {
  model: TenantModelEntry;
  canManage: boolean;
  onToggle: (next: boolean) => void;
  isUpdating: boolean;
}) {
  const disabled =
    !canManage || (!model.tenant_assignable && !model.attached) || !model.enabled;
  return (
    <tr className="border-t">
      <td className="px-3 py-2 font-medium">{model.alias}</td>
      <td className="px-3 py-2 text-xs text-muted-foreground">{model.provider}</td>
      <td className="px-3 py-2 text-xs text-muted-foreground">
        {model.model_type}
      </td>
      <td className="px-3 py-2">
        <Badge variant={model.enabled ? "secondary" : "outline"}>
          {model.enabled ? "enabled" : "disabled"}
        </Badge>
      </td>
      <td className="px-3 py-2 text-right">
        <div className="flex items-center justify-end gap-2">
          {!model.tenant_assignable && !model.attached ? (
            <span className="text-xs text-muted-foreground">Not assignable</span>
          ) : null}
          <Switch
            checked={model.attached}
            onCheckedChange={onToggle}
            disabled={disabled || isUpdating}
          />
        </div>
      </td>
    </tr>
  );
}

type MemberBudgetUpdatePayload = {
  budget_usd?: number;
  warning_threshold?: number;
  token_cap?: number;
};

type TenantLimitPayloadRequest = {
  requests_per_minute: number;
  tokens_per_minute: number;
  parallel_requests: number;
};

export function renderMemberBudget(budget?: MemberBudget) {
  if (!budget || (!budget.budget_usd && !budget.warning_threshold && !budget.token_cap)) {
    return "Default";
  }
  const pieces: string[] = [];
  if (budget.budget_usd && budget.budget_usd > 0) {
    pieces.push(`$${budget.budget_usd.toFixed(2)}`);
  }
  if (budget.token_cap && budget.token_cap > 0) {
    pieces.push(`${budget.token_cap.toLocaleString()} tokens`);
  }
  if (budget.warning_threshold && budget.warning_threshold > 0) {
    pieces.push(`${Math.round(budget.warning_threshold * 100)}% warn`);
  }
  return pieces.join(" · ");
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
