import type { BudgetDefaults } from "@/api/budgets";
import type { ModelCatalogEntry } from "@/api/model-catalog";
import type { TenantStatus } from "@/api/tenants";
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
import { Switch } from "@/components/ui/switch";

import {
  INHERIT_SCHEDULE,
  type TenantEditDialogState,
} from "../hooks/useTenantDialogs";
import { ModelAccessSelector } from "./ModelAccessSelector";
import { currencyFormatter } from "../utils";

type TenantEditDialogProps = {
  dialog: TenantEditDialogState;
  statusOptions: TenantStatus[];
  budgetDefaults?: BudgetDefaults;
  modelCatalog: ModelCatalogEntry[];
  isModelCatalogLoading: boolean;
  isSubmitting: boolean;
  editModelsLoading: boolean;
  editBudgetLoading: boolean;
  guardrailLoading: boolean;
  onToggleModel: (alias: string, checked: boolean) => void;
  onSelectAllModels: () => void;
  onClearModels: () => void;
  onSubmit: () => void;
};

export function TenantEditDialog({
  dialog,
  statusOptions,
  budgetDefaults,
  modelCatalog,
  isModelCatalogLoading,
  isSubmitting,
  editModelsLoading,
  editBudgetLoading,
  guardrailLoading,
  onToggleModel,
  onSelectAllModels,
  onClearModels,
  onSubmit,
}: TenantEditDialogProps) {
  const handleToggle = (alias: string, checked: boolean) => {
    onToggleModel(alias, checked);
  };

  const fullyDisabled = !dialog.tenant;

  return (
    <Dialog open={dialog.open} onOpenChange={dialog.setOpen}>
      <DialogContent className="sm:max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {dialog.tenant ? `Edit ${dialog.tenant.name}` : "Edit tenant"}
          </DialogTitle>
          <DialogDescription>
            Update tenant metadata, status, and budget overrides.
          </DialogDescription>
        </DialogHeader>
        {dialog.tenant ? (
          <div className="space-y-4 py-2">
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
            <Separator />
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
            />
            <Separator />
            <div className="space-y-3">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <Label>Guardrail policy</Label>
                  <p className="text-sm text-muted-foreground">
                    Override prompt/response guardrails for this tenant.
                  </p>
                </div>
                <Switch
                  checked={dialog.guardrailOverride}
                  disabled={guardrailLoading || fullyDisabled || isSubmitting}
                  onCheckedChange={(checked) =>
                    dialog.setGuardrailOverride(checked)
                  }
                />
              </div>
              {dialog.guardrailOverride ? (
                <div className="space-y-4 rounded-lg border p-4">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="text-sm font-medium text-foreground">
                        Enforce guardrails
                      </p>
                      <p className="text-sm text-muted-foreground">
                        Temporarily disable without removing this policy.
                      </p>
                    </div>
                    <Switch
                      checked={dialog.guardrailEnabled}
                      disabled={guardrailLoading || isSubmitting}
                      onCheckedChange={(checked) =>
                        dialog.setGuardrailEnabled(checked)
                      }
                    />
                  </div>
                  <div className="grid gap-4 md:grid-cols-2">
                    <div className="space-y-2">
                      <Label htmlFor="tenant-guardrail-prompt">
                        Prompt keywords
                      </Label>
                      <Textarea
                        id="tenant-guardrail-prompt"
                        value={dialog.guardrailPromptKeywords}
                        onChange={(event) =>
                          dialog.setGuardrailPromptKeywords(event.target.value)
                        }
                        placeholder="fraud\nhate"
                        disabled={guardrailLoading || isSubmitting}
                      />
                      <p className="text-xs text-muted-foreground">
                        Block requests containing these keywords before they
                        reach providers.
                      </p>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="tenant-guardrail-response">
                        Response keywords
                      </Label>
                      <Textarea
                        id="tenant-guardrail-response"
                        value={dialog.guardrailResponseKeywords}
                        onChange={(event) =>
                          dialog.setGuardrailResponseKeywords(
                            event.target.value,
                          )
                        }
                        placeholder="password\npii"
                        disabled={guardrailLoading || isSubmitting}
                      />
                      <p className="text-xs text-muted-foreground">
                        Block responses that contain any of these keywords.
                      </p>
                    </div>
                  </div>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-2">
                      <Label>Moderation provider</Label>
                      <Select
                        value={dialog.guardrailModerationProvider}
                        onValueChange={dialog.setGuardrailModerationProvider}
                        disabled={guardrailLoading || isSubmitting}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="keyword">
                            Keyword only
                          </SelectItem>
                          <SelectItem value="webhook">Webhook</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label>Moderation action</Label>
                      <Select
                        value={dialog.guardrailModerationAction}
                        onValueChange={(value) =>
                          dialog.setGuardrailModerationAction(value)
                        }
                        disabled={guardrailLoading || isSubmitting}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="block">Block</SelectItem>
                          <SelectItem value="warn">Warn</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  <div className="flex items-start justify-between gap-4 rounded-lg border px-4 py-3">
                    <div>
                      <p className="text-sm font-medium text-foreground">
                        Enable moderation checks
                      </p>
                      <p className="text-sm text-muted-foreground">
                        Run prompts and responses through the provider for
                        classification.
                      </p>
                    </div>
                    <Switch
                      checked={dialog.guardrailModerationEnabled}
                      disabled={guardrailLoading || isSubmitting}
                      onCheckedChange={(checked) =>
                        dialog.setGuardrailModerationEnabled(checked)
                      }
                    />
                  </div>
                  {dialog.guardrailModerationProvider === "webhook" ? (
                    <div className="space-y-3 rounded-lg border p-4">
                      <div className="space-y-2">
                        <Label htmlFor="tenant-guardrail-webhook-url">
                          Webhook URL
                        </Label>
                        <Input
                          id="tenant-guardrail-webhook-url"
                          value={dialog.guardrailWebhookURL}
                          onChange={(event) =>
                            dialog.setGuardrailWebhookURL(event.target.value)
                          }
                          placeholder="https://example.com/moderate"
                          disabled={guardrailLoading || isSubmitting}
                        />
                      </div>
                      <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-2">
                          <Label htmlFor="tenant-guardrail-webhook-header">
                            Auth header
                          </Label>
                          <Input
                            id="tenant-guardrail-webhook-header"
                            value={dialog.guardrailWebhookHeader}
                            onChange={(event) =>
                              dialog.setGuardrailWebhookHeader(
                                event.target.value,
                              )
                            }
                            placeholder="Authorization"
                            disabled={guardrailLoading || isSubmitting}
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="tenant-guardrail-webhook-value">
                            Auth value
                          </Label>
                          <Input
                            id="tenant-guardrail-webhook-value"
                            value={dialog.guardrailWebhookValue}
                            onChange={(event) =>
                              dialog.setGuardrailWebhookValue(
                                event.target.value,
                              )
                            }
                            placeholder="Bearer ..."
                            disabled={guardrailLoading || isSubmitting}
                          />
                        </div>
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="tenant-guardrail-webhook-timeout">
                          Timeout (seconds)
                        </Label>
                        <Input
                          id="tenant-guardrail-webhook-timeout"
                          value={dialog.guardrailWebhookTimeout}
                          onChange={(event) =>
                            dialog.setGuardrailWebhookTimeout(
                              event.target.value,
                            )
                          }
                          disabled={guardrailLoading || isSubmitting}
                        />
                      </div>
                      <p className="text-xs text-muted-foreground">
                        The webhook receives {"{ stage, content }"} and should
                        respond with {"{ action, violations[], category }"}.
                      </p>
                    </div>
                  ) : null}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Inheriting project-level guardrails.
                </p>
              )}
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            Select a tenant to edit.
          </p>
        )}
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => dialog.setOpen(false)}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button
            onClick={onSubmit}
            disabled={
              isSubmitting ||
              editBudgetLoading ||
              editModelsLoading ||
              guardrailLoading ||
              fullyDisabled
            }
          >
            {isSubmitting ? "Saving…" : "Save changes"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
