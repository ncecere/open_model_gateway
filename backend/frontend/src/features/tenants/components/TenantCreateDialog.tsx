import type { BudgetDefaults } from "@/api/budgets";
import type { ModelCatalogEntry } from "@/api/model-catalog";
import type { TenantStatus } from "@/api/tenants";
import type { RateLimitDefaults } from "@/api/rate-limits";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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

import { ModelAccessSelector } from "./ModelAccessSelector";
import { INHERIT_SCHEDULE, type TenantCreateDialogState } from "../hooks/useTenantDialogs";
import { currencyFormatter } from "../utils";

type TenantCreateDialogProps = {
  dialog: TenantCreateDialogState;
  statusOptions: TenantStatus[];
  budgetDefaults?: BudgetDefaults;
  rateLimitDefaults?: RateLimitDefaults;
  modelCatalog: ModelCatalogEntry[];
  modelAliases: string[];
  isModelCatalogLoading: boolean;
  isSubmitting: boolean;
  onSubmit: () => void;
};

export function TenantCreateDialog({
  dialog,
  statusOptions,
  budgetDefaults,
  rateLimitDefaults,
  modelCatalog,
  modelAliases,
  isModelCatalogLoading,
  isSubmitting,
  onSubmit,
}: TenantCreateDialogProps) {
  const handleToggle = (alias: string, checked: boolean) => {
    dialog.setSelectedModels((prev) => {
      if (checked) {
        if (prev.includes(alias)) {
          return prev;
        }
        return [...prev, alias];
      }
      return prev.filter((item) => item !== alias);
    });
  };

  const handleSelectAll = () => {
    dialog.setSelectedModels(modelAliases);
  };

  const handleClear = () => {
    dialog.setSelectedModels([]);
  };

  return (
    <Dialog open={dialog.open} onOpenChange={dialog.setOpen}>
      <DialogContent className="sm:max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create tenant</DialogTitle>
          <DialogDescription>
            Provide the tenant name, lifecycle status, and optional budget overrides.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="tenant-name">Name</Label>
            <Input
              id="tenant-name"
              value={dialog.name}
              onChange={(event) => dialog.setName(event.target.value)}
              placeholder="Acme Corp"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label>Status</Label>
            <Select
              value={dialog.status}
              onValueChange={(value) => dialog.setStatus(value as TenantStatus)}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select a status" />
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
                <Label htmlFor="tenant-rpm">Requests per minute</Label>
                <Input
                  id="tenant-rpm"
                  value={dialog.requestsPerMinute}
                  onChange={(event) => dialog.setRequestsPerMinute(event.target.value)}
                  placeholder={
                    rateLimitDefaults
                      ? `${rateLimitDefaults.requests_per_minute}`
                      : "e.g. 100"
                  }
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tenant-tpm">Tokens per minute</Label>
                <Input
                  id="tenant-tpm"
                  value={dialog.tokensPerMinute}
                  onChange={(event) => dialog.setTokensPerMinute(event.target.value)}
                  placeholder={
                    rateLimitDefaults
                      ? `${rateLimitDefaults.tokens_per_minute}`
                      : "e.g. 200000"
                  }
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tenant-parallel">Parallel requests</Label>
                <Input
                  id="tenant-parallel"
                  value={dialog.parallelRequests}
                  onChange={(event) => dialog.setParallelRequests(event.target.value)}
                  placeholder={
                    rateLimitDefaults
                      ? `${rateLimitDefaults.parallel_requests_tenant}`
                      : "e.g. 10"
                  }
                />
              </div>
            </div>
          </div>
          <Separator />
          <div className="space-y-2">
            <Label htmlFor="tenant-budget">Budget override (optional)</Label>
            <Input
              id="tenant-budget"
              value={dialog.budgetUsd}
              onChange={(event) => dialog.setBudgetUsd(event.target.value)}
              placeholder={
                budgetDefaults
                  ? `Default ${currencyFormatter.format(budgetDefaults.default_usd)}`
                  : "e.g. 200"
              }
            />
            <p className="text-xs text-muted-foreground">
              Leave blank to inherit project defaults. Provide a value to create a tenant-specific budget,
              warning threshold, and alert routing.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="tenant-threshold">Warning threshold (0-1)</Label>
            <Input
              id="tenant-threshold"
              value={dialog.warningThreshold}
              onChange={(event) => dialog.setWarningThreshold(event.target.value)}
              placeholder="0.75"
            />
          </div>
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
                <SelectItem value={INHERIT_SCHEDULE}>Use project default</SelectItem>
                <SelectItem value="calendar_month">Calendar month</SelectItem>
                <SelectItem value="weekly">Weekly</SelectItem>
                <SelectItem value="rolling_7d">Rolling 7 days</SelectItem>
                <SelectItem value="rolling_30d">Rolling 30 days</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="tenant-alert-emails">Alert emails (comma-separated)</Label>
            <Textarea
              id="tenant-alert-emails"
              value={dialog.alertEmails}
              onChange={(event) => dialog.setAlertEmails(event.target.value)}
              placeholder="alerts@example.com, ops@example.com"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="tenant-alert-webhooks">Alert webhooks (comma-separated URLs)</Label>
            <Textarea
              id="tenant-alert-webhooks"
              value={dialog.alertWebhooks}
              onChange={(event) => dialog.setAlertWebhooks(event.target.value)}
              placeholder="https://hooks.slack.com/..."
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="tenant-alert-cooldown">Alert cooldown (seconds)</Label>
            <Input
              id="tenant-alert-cooldown"
              value={dialog.alertCooldown}
              onChange={(event) => dialog.setAlertCooldown(event.target.value)}
              placeholder={
                budgetDefaults?.alert?.cooldown_seconds != null
                  ? `${budgetDefaults.alert.cooldown_seconds} (default)`
                  : "3600"
              }
            />
          </div>
          <Separator />
          <ModelAccessSelector
            title="Model access"
            description="Select which catalog entries this tenant can call."
            models={modelCatalog}
            selected={dialog.selectedModels}
            onToggle={handleToggle}
            onSelectAll={handleSelectAll}
            onClear={handleClear}
            isLoading={isModelCatalogLoading}
            disabled={isSubmitting}
          />
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => dialog.setOpen(false)}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button onClick={onSubmit} disabled={isSubmitting}>
            {isSubmitting ? "Creating…" : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
