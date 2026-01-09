import { useState, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
  type TenantMembership,
  type TenantModelEntry,
  type TenantSummary,
  type TenantLimitPayload,
  type MemberBudgetUpdate,
} from "@/api/user/tenants";
import { searchDirectoryUsers, type DirectoryUser } from "@/api/user/directory";
import { useToast } from "@/hooks/use-toast";
import { DetailStat } from "./DetailStat";
import { MANAGEABLE_ROLES, renderMemberBudget } from "../utils";

export type TenantDetailDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tenantId: string | undefined;
  summary: TenantSummary | undefined;
  summaryLoading: boolean;
};

export function TenantDetailDialog({
  open,
  onOpenChange,
  tenantId,
  summary,
  summaryLoading,
}: TenantDetailDialogProps) {
  const { toast } = useToast();
  const queryClient = useQueryClient();

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
    if (!open) {
      setInviteEmail("");
      setInviteRole("user");
    }
  }, [open]);

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

  const canManageMembers = summary && (summary.role === "owner" || summary.role === "admin");
  const canManageLimits = Boolean(canManageMembers);
  const canManageModels = Boolean(canManageMembers);

  const membershipsQuery = useQuery({
    queryKey: ["user-tenant-memberships", tenantId],
    queryFn: () => (tenantId ? listTenantMemberships(tenantId) : Promise.resolve([])),
    enabled: Boolean(open && tenantId && canManageMembers),
  });

  const modelsQuery = useQuery({
    queryKey: ["user-tenant-models", tenantId],
    queryFn: () => (tenantId ? listTenantModels(tenantId) : Promise.resolve([])),
    enabled: Boolean(open && tenantId),
  });

  const limitsQuery = useQuery({
    queryKey: ["user-tenant-limits", tenantId],
    queryFn: () => (tenantId ? getTenantLimits(tenantId) : Promise.reject()),
    enabled: Boolean(open && tenantId),
  });

  useEffect(() => {
    if (!limitsQuery.data) return;
    setLimitForm({
      requestsPerMinute: limitsQuery.data.effective.requests_per_minute.toString(),
      tokensPerMinute: limitsQuery.data.effective.tokens_per_minute.toString(),
      parallelRequests: limitsQuery.data.effective.parallel_requests.toString(),
    });
  }, [limitsQuery.data]);

  const suggestionQuery = useQuery({
    queryKey: ["user-directory", inviteEmail],
    queryFn: () => searchDirectoryUsers(inviteEmail.trim()),
    enabled:
      Boolean(open && tenantId && canManageMembers) && inviteEmail.trim().length >= 2,
    staleTime: 30_000,
  });

  const inviteMutation = useMutation({
    mutationFn: () => {
      if (!tenantId) return Promise.reject(new Error("tenant not selected"));
      return inviteTenantMember(tenantId, { email: inviteEmail, role: inviteRole });
    },
    onSuccess: () => {
      toast({ title: "Member added" });
      queryClient.invalidateQueries({ queryKey: ["user-tenant-memberships", tenantId] });
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

  const removeMutation = useMutation({
    mutationFn: (userId: string) => {
      if (!tenantId) return Promise.reject(new Error("tenant not selected"));
      return removeTenantMember(tenantId, userId);
    },
    onSuccess: () => {
      toast({ title: "Membership removed" });
      queryClient.invalidateQueries({ queryKey: ["user-tenant-memberships", tenantId] });
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to remove member",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const budgetMutation = useMutation({
    mutationFn: (payload: MemberBudgetUpdate) => {
      if (!tenantId || !editingMember) return Promise.reject(new Error("member not selected"));
      return updateTenantMemberBudget(tenantId, editingMember.user_id, payload);
    },
    onSuccess: () => {
      toast({ title: "Member budget updated" });
      queryClient.invalidateQueries({ queryKey: ["user-tenant-memberships", tenantId] });
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
    mutationFn: (payload: { requests_per_minute: number; tokens_per_minute: number; parallel_requests: number }) => {
      if (!tenantId) return Promise.reject(new Error("tenant not selected"));
      return updateTenantLimits(tenantId, payload);
    },
    onSuccess: () => {
      toast({ title: "Tenant limits updated" });
      queryClient.invalidateQueries({ queryKey: ["user-tenant-limits", tenantId] });
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
      if (!tenantId) return Promise.reject(new Error("tenant not selected"));
      return deleteTenantLimits(tenantId);
    },
    onSuccess: () => {
      toast({ title: "Tenant limits reset" });
      queryClient.invalidateQueries({ queryKey: ["user-tenant-limits", tenantId] });
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to reset limits",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const attachModelMutation = useMutation({
    mutationFn: (alias: string) => {
      if (!tenantId) return Promise.reject(new Error("tenant not selected"));
      return attachTenantModel(tenantId, alias);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["user-tenant-models", tenantId] });
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
      if (!tenantId) return Promise.reject(new Error("tenant not selected"));
      return detachTenantModel(tenantId, alias);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["user-tenant-models", tenantId] });
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to detach model",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const handleLimitSubmit = () => {
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
  };

  const handleBudgetSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    const payload: MemberBudgetUpdate = {};
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
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="flex h-[70vh] max-w-3xl flex-col overflow-hidden">
          <DialogHeader>
            <DialogTitle>
              Tenant details{summary ? ` · ${summary.name}` : ""}
            </DialogTitle>
          </DialogHeader>
          {summaryLoading || !summary ? (
            <p className="text-sm text-muted-foreground">Loading tenant details…</p>
          ) : (
            <Tabs defaultValue="overview" className="flex h-full min-h-0 flex-col space-y-4 text-sm">
              <TabsList className="w-fit">
                <TabsTrigger value="overview">Overview</TabsTrigger>
                {canManageModels ? <TabsTrigger value="models">Models</TabsTrigger> : null}
                {canManageMembers ? <TabsTrigger value="members">Members</TabsTrigger> : null}
              </TabsList>

              <TabsContent value="overview" className="flex-1 min-h-0 space-y-6 overflow-y-auto pr-1">
                <OverviewTab
                  summary={summary}
                  limitsQuery={limitsQuery}
                  limitForm={limitForm}
                  setLimitForm={setLimitForm}
                  canManageLimits={canManageLimits}
                  onLimitSubmit={handleLimitSubmit}
                  onClearLimits={() => clearLimitsMutation.mutate()}
                  limitMutationPending={limitMutation.isPending}
                  clearLimitsMutationPending={clearLimitsMutation.isPending}
                />
              </TabsContent>

              {canManageModels ? (
                <TabsContent value="models" className="flex-1 min-h-0 space-y-4 overflow-y-auto pr-1">
                  <ModelsTab
                    modelsQuery={modelsQuery}
                    canManage={canManageModels}
                    onAttach={(alias) => attachModelMutation.mutate(alias)}
                    onDetach={(alias) => detachModelMutation.mutate(alias)}
                    isUpdating={attachModelMutation.isPending || detachModelMutation.isPending}
                  />
                </TabsContent>
              ) : null}

              {canManageMembers ? (
                <TabsContent value="members" className="flex-1 min-h-0 space-y-4 overflow-y-auto pr-1">
                  <MembersTab
                    membershipsQuery={membershipsQuery}
                    summaryRole={summary.role}
                    inviteEmail={inviteEmail}
                    inviteRole={inviteRole}
                    suggestions={suggestionQuery.data ?? []}
                    loadingSuggestions={suggestionQuery.isFetching}
                    onInviteEmailChange={setInviteEmail}
                    onInviteRoleChange={(value) => setInviteRole(value as MembershipRole)}
                    onSuggestionSelect={(user) => setInviteEmail(user.email)}
                    onInvite={(event) => {
                      event.preventDefault();
                      inviteMutation.mutate();
                    }}
                    inviting={inviteMutation.isPending}
                    onEditBudget={setEditingMember}
                    onRemove={(userId) => removeMutation.mutate(userId)}
                    removingUserId={removeMutation.isPending ? removeMutation.variables : undefined}
                  />
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
            <form className="space-y-3 text-sm" onSubmit={handleBudgetSubmit}>
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
                    setBudgetForm((prev) => ({ ...prev, warningThreshold: event.target.value }))
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
                <Button type="button" variant="outline" onClick={() => setEditingMember(null)}>
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
    </>
  );
}

// --- Internal Tab Components ---

type LimitFormState = {
  requestsPerMinute: string;
  tokensPerMinute: string;
  parallelRequests: string;
};

type OverviewTabProps = {
  summary: TenantSummary;
  limitsQuery: ReturnType<typeof useQuery<TenantLimitPayload>>;
  limitForm: LimitFormState;
  setLimitForm: React.Dispatch<React.SetStateAction<LimitFormState>>;
  canManageLimits: boolean;
  onLimitSubmit: () => void;
  onClearLimits: () => void;
  limitMutationPending: boolean;
  clearLimitsMutationPending: boolean;
};

function OverviewTab({
  summary,
  limitsQuery,
  limitForm,
  setLimitForm,
  canManageLimits,
  onLimitSubmit,
  onClearLimits,
  limitMutationPending,
  clearLimitsMutationPending,
}: OverviewTabProps) {
  return (
    <>
      <div className="grid gap-4 md:grid-cols-3">
        <DetailStat label="Status" value={summary.status} />
        <DetailStat label="Role" value={summary.role} />
        <DetailStat label="Created" value={new Date(summary.created_at).toLocaleDateString()} />
      </div>
      <Separator />
      <div className="grid gap-4 pt-2 md:grid-cols-3">
        <DetailStat label="Budget limit" value={`$${summary.budget.limit_usd.toFixed(2)}`} />
        <DetailStat label="Remaining" value={`$${summary.budget.remaining_usd.toFixed(2)}`} />
        <DetailStat label="Refresh schedule" value={summary.budget.refresh_schedule} />
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
              <DetailStat label="Effective RPM" value={limitsQuery.data.effective.requests_per_minute.toString()} />
              <DetailStat label="Effective TPM" value={limitsQuery.data.effective.tokens_per_minute.toString()} />
              <DetailStat label="Effective concurrency" value={limitsQuery.data.effective.parallel_requests.toString()} />
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
                        setLimitForm((prev) => ({ ...prev, requestsPerMinute: event.target.value }))
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
                        setLimitForm((prev) => ({ ...prev, tokensPerMinute: event.target.value }))
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
                        setLimitForm((prev) => ({ ...prev, parallelRequests: event.target.value }))
                      }
                    />
                  </div>
                </div>
                <div className="flex flex-wrap justify-end gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onClearLimits}
                    disabled={clearLimitsMutationPending}
                  >
                    Reset override
                  </Button>
                  <Button size="sm" onClick={onLimitSubmit} disabled={limitMutationPending}>
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
    </>
  );
}

type ModelsTabProps = {
  modelsQuery: ReturnType<typeof useQuery<TenantModelEntry[]>>;
  canManage: boolean;
  onAttach: (alias: string) => void;
  onDetach: (alias: string) => void;
  isUpdating: boolean;
};

function ModelsTab({ modelsQuery, canManage, onAttach, onDetach, isUpdating }: ModelsTabProps) {
  return (
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
                  canManage={canManage}
                  onToggle={(next) => (next ? onAttach(model.alias) : onDetach(model.alias))}
                  isUpdating={isUpdating}
                />
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">No approved models available yet.</p>
      )}
    </section>
  );
}

export type TenantModelRowProps = {
  model: TenantModelEntry;
  canManage: boolean;
  onToggle: (next: boolean) => void;
  isUpdating: boolean;
};

export function TenantModelRow({
  model,
  canManage,
  onToggle,
  isUpdating,
}: TenantModelRowProps) {
  const disabled = !canManage || (!model.tenant_assignable && !model.attached) || !model.enabled;
  return (
    <tr className="border-t">
      <td className="px-3 py-2 font-medium">{model.alias}</td>
      <td className="px-3 py-2 text-xs text-muted-foreground">{model.provider}</td>
      <td className="px-3 py-2 text-xs text-muted-foreground">{model.model_type}</td>
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
          <Switch checked={model.attached} onCheckedChange={onToggle} disabled={disabled || isUpdating} />
        </div>
      </td>
    </tr>
  );
}

type MembersTabProps = {
  membershipsQuery: ReturnType<typeof useQuery<TenantMembership[]>>;
  summaryRole: string;
  inviteEmail: string;
  inviteRole: MembershipRole;
  suggestions: DirectoryUser[];
  loadingSuggestions: boolean;
  onInviteEmailChange: (value: string) => void;
  onInviteRoleChange: (value: string) => void;
  onSuggestionSelect: (user: DirectoryUser) => void;
  onInvite: (event: React.FormEvent<HTMLFormElement>) => void;
  inviting: boolean;
  onEditBudget: (member: TenantMembership) => void;
  onRemove: (userId: string) => void;
  removingUserId: string | undefined;
};

function MembersTab({
  membershipsQuery,
  summaryRole,
  inviteEmail,
  inviteRole,
  suggestions,
  loadingSuggestions,
  onInviteEmailChange,
  onInviteRoleChange,
  onSuggestionSelect,
  onInvite,
  inviting,
  onEditBudget,
  onRemove,
  removingUserId,
}: MembersTabProps) {
  return (
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
                  canEdit={summaryRole === "owner" || (summaryRole === "admin" && member.role !== "owner")}
                  canEditBudget
                  onEditBudget={() => onEditBudget(member)}
                  onRemove={() => onRemove(member.user_id)}
                  removing={removingUserId === member.user_id}
                />
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">No members yet.</p>
      )}
      <InviteForm
        email={inviteEmail}
        role={inviteRole}
        suggestions={suggestions}
        loadingSuggestions={loadingSuggestions}
        onEmailChange={onInviteEmailChange}
        onRoleChange={onInviteRoleChange}
        onSuggestionSelect={onSuggestionSelect}
        onSubmit={onInvite}
        submitting={inviting}
      />
    </section>
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
  onEditBudget: () => void;
  onRemove: () => void;
  removing: boolean;
}) {
  return (
    <tr className="border-t text-sm">
      <td className="py-2">
        <div className="flex flex-col">
          <span className="font-medium">{member.email}</span>
          {member.self ? <span className="text-xs text-muted-foreground">You</span> : null}
        </div>
      </td>
      <td className="py-2 capitalize">{member.role}</td>
      <td className="py-2 text-sm text-muted-foreground">
        {new Date(member.created_at).toLocaleDateString()}
      </td>
      <td className="py-2 text-sm text-muted-foreground">{renderMemberBudget(member.budget)}</td>
      <td className="py-2 text-right">
        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" disabled={!canEditBudget} onClick={onEditBudget}>
            Budget
          </Button>
          <Button
            variant="ghost"
            size="sm"
            disabled={!canEdit || member.self || removing}
            onClick={onRemove}
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
  if (trimmed.length < 2) return null;
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
        No user matches "{trimmed}". Create the account from the admin portal first.
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
