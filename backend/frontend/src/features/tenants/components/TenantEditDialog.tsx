import type { BudgetDefaults } from "@/api/budgets";
import type { ModelCatalogEntry } from "@/api/model-catalog";
import type { TenantStatus } from "@/api/tenants";
import type { RateLimitDefaults } from "@/api/rate-limits";
import { FormDialog } from "@/components/dialogs";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
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
import { Trash2, UserRoundPlus } from "lucide-react";

import {
  INHERIT_SCHEDULE,
  type TenantEditDialogState,
} from "../hooks/useTenantDialogs";
import { ModelAccessSelector } from "./ModelAccessSelector";
import { currencyFormatter } from "../utils";
import type { MembershipRecord } from "@/api/tenants";

type TenantEditDialogProps = {
  dialog: TenantEditDialogState;
  statusOptions: TenantStatus[];
  budgetDefaults?: BudgetDefaults;
  rateLimitDefaults?: RateLimitDefaults;
  modelCatalog: ModelCatalogEntry[];
  isModelCatalogLoading: boolean;
  isSubmitting: boolean;
  editModelsLoading: boolean;
  editBudgetLoading: boolean;
  editRateLoading: boolean;
  onToggleModel: (alias: string, checked: boolean) => void;
  onSelectAllModels: () => void;
  onClearModels: () => void;
  onSubmit: () => void;
  memberships: MembershipRecord[];
  membershipsLoading: boolean;
  onInviteMember: () => void;
  onRemoveMember: (member: MembershipRecord) => void;
  isRemovingMember: boolean;
  activeTab: string;
  onTabChange: (value: string) => void;
};

export function TenantEditDialog({
  dialog,
  statusOptions,
  budgetDefaults,
  rateLimitDefaults,
  modelCatalog,
  isModelCatalogLoading,
  isSubmitting,
  editModelsLoading,
  editBudgetLoading,
  editRateLoading,
  onToggleModel,
  onSelectAllModels,
  onClearModels,
  onSubmit,
  memberships,
  membershipsLoading,
  onInviteMember,
  onRemoveMember,
  isRemovingMember,
  activeTab,
  onTabChange,
}: TenantEditDialogProps) {
  const handleToggle = (alias: string, checked: boolean) => {
    onToggleModel(alias, checked);
  };

  const fullyDisabled = !dialog.tenant;
  const rateLimitDisabled = editRateLoading || fullyDisabled;

  return (
    <FormDialog
      open={dialog.open}
      onOpenChange={dialog.setOpen}
      title={dialog.tenant ? `Edit ${dialog.tenant.name}` : "Edit tenant"}
      description="Update tenant metadata, status, and budget overrides."
      maxWidth="max-w-3xl"
      contentClassName="flex h-[85vh] flex-col overflow-hidden"
      bodyClassName="flex-1 min-h-0 overflow-hidden"
      useForm={false}
      isSubmitting={isSubmitting}
      submitText="Save changes"
      submittingText="Saving…"
      submitDisabled={editBudgetLoading || fullyDisabled}
      onSubmit={onSubmit}
    >
      {dialog.tenant ? (
        <Tabs
          value={activeTab}
          onValueChange={onTabChange}
          className="flex h-full min-h-0 flex-col space-y-4"
        >
          <TabsList className="w-fit">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="alerts">Alerts</TabsTrigger>
            <TabsTrigger value="models">Models</TabsTrigger>
            <TabsTrigger value="members">Members</TabsTrigger>
          </TabsList>
            <TabsContent value="overview" className="flex-1 min-h-0 space-y-4 overflow-y-auto pr-1">
              <div className="space-y-2">
                <Label htmlFor="edit-tenant-name">Name</Label>
                <Input
                  id="edit-tenant-name"
                  value={dialog.name}
                  onChange={(event) => dialog.setName(event.target.value)}
                  disabled={editBudgetLoading}
                />
              </div>
              <div className="space-y-2">
                <Label>Status</Label>
                <Select
                  value={dialog.status}
                  onValueChange={(value) =>
                    dialog.setStatus(value as TenantStatus)
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {statusOptions.map((status) => (
                      <SelectItem key={status} value={status}>
                        {status}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Separator />
              <div className="space-y-2">
                <Label>Rate limit override (optional)</Label>
                <p className="text-xs text-muted-foreground">
                  Leave blank to inherit project defaults. Current defaults:{" "}
                  {rateLimitDefaults
                    ? `${rateLimitDefaults.requests_per_minute} RPM, ${rateLimitDefaults.tokens_per_minute} TPM, ${rateLimitDefaults.parallel_requests_tenant} parallel`
                    : "loading…"}
                </p>
                <div className="grid gap-4 md:grid-cols-3">
                  <div className="space-y-2">
                    <Label htmlFor="edit-tenant-rpm">Requests per minute</Label>
                    <Input
                      id="edit-tenant-rpm"
                      value={dialog.requestsPerMinute}
                      onChange={(event) => dialog.setRequestsPerMinute(event.target.value)}
                      placeholder={
                        rateLimitDefaults
                          ? `${rateLimitDefaults.requests_per_minute}`
                          : "e.g. 100"
                      }
                      disabled={rateLimitDisabled}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-tenant-tpm">Tokens per minute</Label>
                    <Input
                      id="edit-tenant-tpm"
                      value={dialog.tokensPerMinute}
                      onChange={(event) => dialog.setTokensPerMinute(event.target.value)}
                      placeholder={
                        rateLimitDefaults
                          ? `${rateLimitDefaults.tokens_per_minute}`
                          : "e.g. 200000"
                      }
                      disabled={rateLimitDisabled}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-tenant-parallel">Parallel requests</Label>
                    <Input
                      id="edit-tenant-parallel"
                      value={dialog.parallelRequests}
                      onChange={(event) => dialog.setParallelRequests(event.target.value)}
                      placeholder={
                        rateLimitDefaults
                          ? `${rateLimitDefaults.parallel_requests_tenant}`
                          : "e.g. 10"
                      }
                      disabled={rateLimitDisabled}
                    />
                  </div>
                </div>
              </div>
            </TabsContent>
            <TabsContent value="alerts" className="flex-1 min-h-0 space-y-3 overflow-y-auto pr-1">
              <div className="grid gap-3 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="edit-tenant-budget">
                    Budget override (leave blank to inherit)
                  </Label>
                  <Input
                    id="edit-tenant-budget"
                    value={dialog.budgetUsd}
                    onChange={(event) => dialog.setBudgetUsd(event.target.value)}
                    placeholder={
                      budgetDefaults
                        ? `Default ${currencyFormatter.format(budgetDefaults.default_usd)}`
                        : "e.g. 250"
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-tenant-threshold">
                    Warning threshold (0-1)
                  </Label>
                  <Input
                    id="edit-tenant-threshold"
                    value={dialog.warningThreshold}
                    onChange={(event) => dialog.setWarningThreshold(event.target.value)}
                    placeholder="0.8"
                  />
                </div>
              </div>
              <div className="grid gap-3 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>Refresh schedule</Label>
                  <Select
                    value={dialog.refreshSchedule}
                    onValueChange={dialog.setRefreshSchedule}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Use default schedule" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={INHERIT_SCHEDULE}>
                        Use project default
                      </SelectItem>
                      <SelectItem value="calendar_month">Calendar month</SelectItem>
                      <SelectItem value="weekly">Weekly</SelectItem>
                      <SelectItem value="rolling_7d">Rolling 7 days</SelectItem>
                      <SelectItem value="rolling_30d">Rolling 30 days</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-alert-cooldown">
                    Alert cooldown (seconds)
                  </Label>
                  <Input
                    id="edit-alert-cooldown"
                    value={dialog.alertCooldown}
                    onChange={(event) => dialog.setAlertCooldown(event.target.value)}
                    placeholder="3600"
                  />
                </div>
              </div>
              <Separator />
              <div className="space-y-2">
                <Label htmlFor="edit-alert-emails">
                  Alert emails (comma-separated)
                </Label>
                <Textarea
                  id="edit-alert-emails"
                  value={dialog.alertEmails}
                  onChange={(event) => dialog.setAlertEmails(event.target.value)}
                  placeholder="alerts@example.com, ops@example.com"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-alert-webhooks">
                  Alert webhooks (comma-separated URLs)
                </Label>
                <Textarea
                  id="edit-alert-webhooks"
                  value={dialog.alertWebhooks}
                  onChange={(event) => dialog.setAlertWebhooks(event.target.value)}
                  placeholder="https://hooks.slack.com/..."
                />
              </div>
            </TabsContent>
            <TabsContent value="models" className="flex-1 min-h-0 overflow-y-hidden pr-1">
              <ModelAccessSelector
                title="Model access"
                description="Choose which catalog entries this tenant can use."
                models={modelCatalog}
                selected={dialog.selectedModels}
                onToggle={handleToggle}
                onSelectAll={onSelectAllModels}
                onClear={onClearModels}
                isLoading={isModelCatalogLoading || editModelsLoading}
                disabled={isSubmitting || editModelsLoading}
                containerClassName="flex h-full min-h-0 flex-col"
                listClassName="flex-1 min-h-0 max-h-none"
              />
            </TabsContent>
            <TabsContent value="members" className="flex-1 min-h-0 space-y-4 overflow-y-auto pr-1">
              <div className="flex items-center justify-between">
                <p className="text-sm text-muted-foreground">
                  Invite teammates or adjust roles. Owners can grant owner access; admins may manage other roles.
                </p>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onInviteMember}
                >
                  <UserRoundPlus className="mr-2 h-4 w-4" /> Invite member
                </Button>
              </div>
              {membershipsLoading ? (
                <div className="space-y-2">
                  {[...Array(3)].map((_, idx) => (
                    <Skeleton key={idx} className="h-10 w-full" />
                  ))}
                </div>
              ) : memberships.length ? (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="text-left text-xs uppercase text-muted-foreground">
                        <th className="pb-2 font-medium">Email</th>
                        <th className="pb-2 font-medium">Role</th>
                        <th className="pb-2 font-medium">Added</th>
                        <th className="pb-2 font-medium text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {memberships.map((member) => (
                        <tr key={member.user_id} className="border-t text-sm">
                          <td className="py-2 font-medium">{member.email}</td>
                          <td className="py-2 capitalize">{member.role}</td>
                          <td className="py-2 text-sm text-muted-foreground">
                            {new Date(member.created_at).toLocaleDateString()}
                          </td>
                          <td className="py-2 text-right">
                            <AlertDialog>
                              <AlertDialogTrigger asChild>
                                <Button
                                  variant="outline"
                                  size="icon"
                                  disabled={isRemovingMember}
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </AlertDialogTrigger>
                              <AlertDialogContent>
                                <AlertDialogHeader>
                                  <AlertDialogTitle>Remove member</AlertDialogTitle>
                                  <AlertDialogDescription>
                                    This will revoke {member.email}'s access to tenant resources.
                                  </AlertDialogDescription>
                                </AlertDialogHeader>
                                <AlertDialogFooter>
                                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                                  <AlertDialogAction onClick={() => onRemoveMember(member)}>
                                    Remove
                                  </AlertDialogAction>
                                </AlertDialogFooter>
                              </AlertDialogContent>
                            </AlertDialog>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No members yet. Invite a collaborator to get started.
                </p>
              )}
          </TabsContent>
        </Tabs>
      ) : (
        <p className="text-sm text-muted-foreground">
          Select a tenant to edit.
        </p>
      )}
    </FormDialog>
  );
}
