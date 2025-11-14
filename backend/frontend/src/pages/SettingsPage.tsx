import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";

import {
  getBudgetDefaults,
  updateBudgetDefaults,
} from "@/api/budgets";
import { listModelCatalog } from "@/api/model-catalog";
import type { ModelCatalogEntry } from "@/api/model-catalog";
import {
  addDefaultModel,
  listDefaultModels,
  removeDefaultModel,
} from "@/api/default-models";
import {
  getRateLimitDefaults,
  updateRateLimitDefaults,
} from "@/api/rate-limits";
import {
  getBatchSettings,
  getFileSettings,
  getAlertSettings,
  updateAlertSettings,
  updateBatchSettings,
  updateFileSettings,
  sendTestAlertEmail,
} from "@/api/runtime-settings";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { useToast } from "@/hooks/use-toast";
import { Separator } from "@/components/ui/separator";

const REFRESH_OPTIONS = [
  { value: "calendar_month", label: "Calendar month" },
  { value: "weekly", label: "Weekly" },
  { value: "rolling_7d", label: "Rolling 7 days" },
  { value: "rolling_30d", label: "Rolling 30 days" },
];

export function SettingsPage() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const defaultsQuery = useQuery({
    queryKey: ["budget-defaults", "settings"],
    queryFn: getBudgetDefaults,
  });
  const defaults = defaultsQuery.data;
  const budgetMetadataDescription = useMemo(() => {
    if (!defaults?.metadata) {
      return null;
    }
    const updatedLabel = defaults.metadata.updated_at
      ? new Date(defaults.metadata.updated_at).toLocaleString()
      : null;
    const updatedBy =
      defaults.metadata.updated_by?.name ??
      defaults.metadata.updated_by?.email ??
      null;
    if (!updatedLabel) {
      return null;
    }
    return updatedBy
      ? `Last updated ${updatedLabel} by ${updatedBy}.`
      : `Last updated ${updatedLabel}.`;
  }, [defaults?.metadata]);

  const defaultModelsQuery = useQuery({
    queryKey: ["default-models"],
    queryFn: listDefaultModels,
  });
  const modelCatalogQuery = useQuery({
    queryKey: ["model-catalog", "settings"],
    queryFn: listModelCatalog,
  });
  const rateLimitQuery = useQuery({
    queryKey: ["rate-limit-defaults"],
    queryFn: getRateLimitDefaults,
  });
  const fileSettingsQuery = useQuery({
    queryKey: ["file-settings"],
    queryFn: getFileSettings,
  });
  const batchSettingsQuery = useQuery({
    queryKey: ["batch-settings"],
    queryFn: getBatchSettings,
  });
  const alertSettingsQuery = useQuery({
    queryKey: ["alert-settings"],
    queryFn: getAlertSettings,
  });

  const [formBudget, setFormBudget] = useState("");
  const [formThreshold, setFormThreshold] = useState("");
  const [formSchedule, setFormSchedule] = useState("calendar_month");
  const [formCooldown, setFormCooldown] = useState("3600");
  const [formEmails, setFormEmails] = useState("");
  const [formWebhooks, setFormWebhooks] = useState("");
  const [newDefaultModel, setNewDefaultModel] = useState("");
  const [formRequestsPerMinute, setFormRequestsPerMinute] = useState("");
  const [formTokensPerMinute, setFormTokensPerMinute] = useState("");
  const [formParallelKey, setFormParallelKey] = useState("");
  const [formParallelTenant, setFormParallelTenant] = useState("");
  const [fileMaxSize, setFileMaxSize] = useState("");
  const [fileDefaultTTL, setFileDefaultTTL] = useState("");
  const [fileMaxTTL, setFileMaxTTL] = useState("");
  const [batchMaxRequests, setBatchMaxRequests] = useState("");
  const [batchMaxConcurrency, setBatchMaxConcurrency] = useState("");
  const [batchDefaultTTL, setBatchDefaultTTL] = useState("");
  const [batchMaxTTL, setBatchMaxTTL] = useState("");
  const [smtpHost, setSmtpHost] = useState("");
  const [smtpPort, setSmtpPort] = useState("");
  const [smtpUsername, setSmtpUsername] = useState("");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [smtpFrom, setSmtpFrom] = useState("");
  const [smtpUseTLS, setSmtpUseTLS] = useState(true);
  const [smtpSkipVerify, setSmtpSkipVerify] = useState(false);
  const [smtpTimeout, setSmtpTimeout] = useState("");
  const [webhookTimeout, setWebhookTimeout] = useState("");
  const [webhookRetries, setWebhookRetries] = useState("");
  const [testEmail, setTestEmail] = useState("");

  useEffect(() => {
    if (!defaults) {
      return;
    }
    setFormBudget(defaults.default_usd.toString());
    setFormThreshold(defaults.warning_threshold_perc.toString());
    setFormSchedule(defaults.refresh_schedule);
    setFormCooldown((defaults.alert.cooldown_seconds ?? 3600).toString());
    setFormEmails((defaults.alert.emails ?? []).join(", "));
    setFormWebhooks((defaults.alert.webhooks ?? []).join(", "));
  }, [defaults]);

  const rateLimitDefaults = rateLimitQuery.data;
  const fileSettings = fileSettingsQuery.data;
  const batchSettings = batchSettingsQuery.data;

  useEffect(() => {
    if (!rateLimitDefaults) {
      return;
    }
    setFormRequestsPerMinute(rateLimitDefaults.requests_per_minute.toString());
    setFormTokensPerMinute(rateLimitDefaults.tokens_per_minute.toString());
    setFormParallelKey(rateLimitDefaults.parallel_requests_key.toString());
    setFormParallelTenant(
      rateLimitDefaults.parallel_requests_tenant.toString(),
    );
  }, [rateLimitDefaults]);

  useEffect(() => {
    if (!fileSettings) {
      return;
    }
    setFileMaxSize(fileSettings.max_size_mb.toString());
    setFileDefaultTTL(fileSettings.default_ttl_seconds.toString());
    setFileMaxTTL(fileSettings.max_ttl_seconds.toString());
  }, [fileSettings]);

  useEffect(() => {
    if (!batchSettings) {
      return;
    }
    setBatchMaxRequests(batchSettings.max_requests.toString());
    setBatchMaxConcurrency(batchSettings.max_concurrency.toString());
    setBatchDefaultTTL(batchSettings.default_ttl_seconds.toString());
    setBatchMaxTTL(batchSettings.max_ttl_seconds.toString());
  }, [batchSettings]);

  useEffect(() => {
    const alertSettings = alertSettingsQuery.data;
    if (!alertSettings) {
      return;
    }
    setSmtpHost(alertSettings.smtp?.host ?? "");
    setSmtpPort(
      alertSettings.smtp?.port != null ? alertSettings.smtp.port.toString() : ""
    );
    setSmtpUsername(alertSettings.smtp?.username ?? "");
    setSmtpPassword(alertSettings.smtp?.password ?? "");
    setSmtpFrom(alertSettings.smtp?.from ?? "");
    setSmtpUseTLS(Boolean(alertSettings.smtp?.use_tls));
    setSmtpSkipVerify(Boolean(alertSettings.smtp?.skip_tls_verify));
    setSmtpTimeout(
      alertSettings.smtp?.connect_timeout_seconds != null
        ? alertSettings.smtp.connect_timeout_seconds.toString()
        : ""
    );
    setWebhookTimeout(
      alertSettings.webhook?.timeout_seconds != null
        ? alertSettings.webhook.timeout_seconds.toString()
        : ""
    );
    setWebhookRetries(
      alertSettings.webhook?.max_retries != null
        ? alertSettings.webhook.max_retries.toString()
        : ""
    );
  }, [alertSettingsQuery.data]);

  const updateMutation = useMutation({
    mutationFn: updateBudgetDefaults,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["budget-defaults", "settings"] });
      toast({ title: "Defaults updated" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to update defaults",
        description: "Check the form values and try again.",
      });
    },
  });

  const rateLimitMutation = useMutation({
    mutationFn: updateRateLimitDefaults,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["rate-limit-defaults"],
      });
      toast({ title: "Rate limits updated" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to update rate limits",
        description: "Double-check the values and try again.",
      });
    },
  });

  const fileSettingsMutation = useMutation({
    mutationFn: updateFileSettings,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["file-settings"] });
      toast({ title: "File settings updated" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to update file settings",
        description: "Double-check the inputs and try again.",
      });
    },
  });

  const batchSettingsMutation = useMutation({
    mutationFn: updateBatchSettings,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["batch-settings"] });
      toast({ title: "Batch settings updated" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to update batch settings",
        description: "Double-check the inputs and try again.",
      });
    },
  });

  const alertSettingsMutation = useMutation({
    mutationFn: updateAlertSettings,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-settings"] });
      toast({ title: "Alert transports updated" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to update alert transports",
        description: "Double-check the inputs and try again.",
      });
    },
  });

  const testEmailMutation = useMutation({
    mutationFn: sendTestAlertEmail,
    onSuccess: () => {
      toast({ title: "Test email sent" });
      setTestEmail("");
    },
    onError: (error: unknown) => {
      toast({
        variant: "destructive",
        title: "Failed to send test email",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const defaultsLoading = defaultsQuery.isLoading;
  const defaultModels = defaultModelsQuery.data ?? [];
  const modelsLoading =
    defaultModelsQuery.isLoading || modelCatalogQuery.isLoading;
  const rateLimitLoading = rateLimitQuery.isLoading;
  const fileSettingsLoading = fileSettingsQuery.isLoading;
  const batchSettingsLoading = batchSettingsQuery.isLoading;

  const catalogByAlias = useMemo(() => {
    const map = new Map<string, ModelCatalogEntry>();
    modelCatalogQuery.data?.forEach((entry) => map.set(entry.alias, entry));
    return map;
  }, [modelCatalogQuery.data]);

  const availableModelOptions = useMemo(() => {
    if (!modelCatalogQuery.data) {
      return [] as ModelCatalogEntry[];
    }
    return modelCatalogQuery.data.filter(
      (entry) => entry.enabled && !defaultModels.includes(entry.alias),
    );
  }, [modelCatalogQuery.data, defaultModels]);

  const handleReset = () => {
    if (defaults) {
      setFormBudget(defaults.default_usd.toString());
      setFormThreshold(defaults.warning_threshold_perc.toString());
      setFormSchedule(defaults.refresh_schedule);
      setFormCooldown((defaults.alert.cooldown_seconds ?? 3600).toString());
      setFormEmails((defaults.alert.emails ?? []).join(", "));
      setFormWebhooks((defaults.alert.webhooks ?? []).join(", "));
    }
  };

  const handleSave = () => {
    const budgetValue = Number.parseFloat(formBudget);
    const thresholdValue = Number.parseFloat(formThreshold);
    const cooldownValue = Number.parseInt(formCooldown, 10);
    if (!Number.isFinite(budgetValue) || budgetValue <= 0) {
      toast({ variant: "destructive", title: "Budget must be positive" });
      return;
    }
    if (
      !Number.isFinite(thresholdValue) ||
      thresholdValue <= 0 ||
      thresholdValue > 1
    ) {
      toast({ variant: "destructive", title: "Threshold must be between 0 and 1" });
      return;
    }
    if (!Number.isFinite(cooldownValue) || cooldownValue <= 0) {
      toast({ variant: "destructive", title: "Cooldown must be positive" });
      return;
    }

    updateMutation.mutate({
      default_usd: budgetValue,
      warning_threshold: thresholdValue,
      refresh_schedule: formSchedule,
      alert_emails: parseListInput(formEmails),
      alert_webhooks: parseListInput(formWebhooks),
      alert_cooldown_seconds: cooldownValue,
    });
  };

  const handleRateLimitReset = () => {
    if (!rateLimitDefaults) {
      return;
    }
    setFormRequestsPerMinute(
      rateLimitDefaults.requests_per_minute.toString(),
    );
    setFormTokensPerMinute(rateLimitDefaults.tokens_per_minute.toString());
    setFormParallelKey(rateLimitDefaults.parallel_requests_key.toString());
    setFormParallelTenant(
      rateLimitDefaults.parallel_requests_tenant.toString(),
    );
  };

  const handleRateLimitSave = () => {
    const rpm = Number.parseInt(formRequestsPerMinute, 10);
    const tpm = Number.parseInt(formTokensPerMinute, 10);
    const parallelKey = Number.parseInt(formParallelKey, 10);
    const parallelTenant = Number.parseInt(formParallelTenant, 10);
    if ([rpm, tpm, parallelKey, parallelTenant].some((value) => !Number.isFinite(value) || value <= 0)) {
      toast({
        variant: "destructive",
        title: "All rate limit values must be positive",
      });
      return;
    }
    rateLimitMutation.mutate({
      requests_per_minute: rpm,
      tokens_per_minute: tpm,
      parallel_requests_key: parallelKey,
      parallel_requests_tenant: parallelTenant,
    });
  };

  const handleFileSettingsReset = () => {
    if (!fileSettings) {
      return;
    }
    setFileMaxSize(fileSettings.max_size_mb.toString());
    setFileDefaultTTL(fileSettings.default_ttl_seconds.toString());
    setFileMaxTTL(fileSettings.max_ttl_seconds.toString());
  };

  const handleFileSettingsSave = () => {
    const maxSize = Number.parseInt(fileMaxSize, 10);
    const defaultTTL = Number.parseInt(fileDefaultTTL, 10);
    const maxTTL = Number.parseInt(fileMaxTTL, 10);
    if (!Number.isFinite(maxSize) || maxSize <= 0) {
      toast({ variant: "destructive", title: "Max size must be positive" });
      return;
    }
    if (!Number.isFinite(defaultTTL) || !Number.isFinite(maxTTL) || defaultTTL <= 0 || maxTTL <= 0) {
      toast({ variant: "destructive", title: "TTLs must be positive" });
      return;
    }
    if (defaultTTL > maxTTL) {
      toast({ variant: "destructive", title: "Default TTL cannot exceed max" });
      return;
    }
    fileSettingsMutation.mutate({
      max_size_mb: maxSize,
      default_ttl_seconds: defaultTTL,
      max_ttl_seconds: maxTTL,
    });
  };

  const handleBatchSettingsReset = () => {
    if (!batchSettings) {
      return;
    }
    setBatchMaxRequests(batchSettings.max_requests.toString());
    setBatchMaxConcurrency(batchSettings.max_concurrency.toString());
    setBatchDefaultTTL(batchSettings.default_ttl_seconds.toString());
    setBatchMaxTTL(batchSettings.max_ttl_seconds.toString());
  };

  const handleBatchSettingsSave = () => {
    const maxRequests = Number.parseInt(batchMaxRequests, 10);
    const maxConcurrency = Number.parseInt(batchMaxConcurrency, 10);
    const defaultTTL = Number.parseInt(batchDefaultTTL, 10);
    const maxTTL = Number.parseInt(batchMaxTTL, 10);
    if ([maxRequests, maxConcurrency].some((value) => !Number.isFinite(value) || value <= 0)) {
      toast({ variant: "destructive", title: "Limits must be positive" });
      return;
    }
    if ([defaultTTL, maxTTL].some((value) => !Number.isFinite(value) || value <= 0)) {
      toast({ variant: "destructive", title: "TTLs must be positive" });
      return;
    }
    if (defaultTTL > maxTTL) {
      toast({ variant: "destructive", title: "Default TTL cannot exceed max" });
      return;
    }
    batchSettingsMutation.mutate({
      max_requests: maxRequests,
      max_concurrency: maxConcurrency,
      default_ttl_seconds: defaultTTL,
      max_ttl_seconds: maxTTL,
    });
  };

  const addModelMutation = useMutation({
    mutationFn: addDefaultModel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["default-models"] });
      toast({ title: "Default model added" });
      setNewDefaultModel("");
    },
    onError: (error: unknown) => {
      console.error(error);
      toast({
        variant: "destructive",
        title: "Failed to add model",
      });
    },
  });

  const removeModelMutation = useMutation({
    mutationFn: removeDefaultModel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["default-models"] });
      toast({ title: "Default model removed" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to remove model",
      });
    },
  });

  const handleAddModel = () => {
    const alias = newDefaultModel.trim();
    if (!alias) {
      return;
    }
    addModelMutation.mutate(alias);
  };

  const handleRemoveModel = (alias: string) => {
    removeModelMutation.mutate(alias.trim());
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Review platform defaults and operational guardrails. Tenant-specific
          budgets can be edited directly from the Tenants tab.
        </p>
      </div>

      <Separator />

      <Card>
        <CardHeader>
          <div className="space-y-1">
            <CardTitle>Budget defaults</CardTitle>
            <p className="text-sm text-muted-foreground">
              {budgetMetadataDescription
                ? `${budgetMetadataDescription} Detailed history view coming soon.`
                : "Detailed history view coming soon. Save changes to capture metadata."}
            </p>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {defaultsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-6 w-1/3" />
              <Skeleton className="h-6 w-1/2" />
              <Skeleton className="h-6 w-2/3" />
            </div>
          ) : (
            <>
              <div className="grid gap-4 md:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="default-budget">Default monthly budget (USD)</Label>
                  <Input
                    id="default-budget"
                    value={formBudget}
                    onChange={(event) => setFormBudget(event.target.value)}
                    type="number"
                    min="0"
                    step="0.01"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="warning-threshold">Warning threshold (0-1)</Label>
                  <Input
                    id="warning-threshold"
                    value={formThreshold}
                    onChange={(event) => setFormThreshold(event.target.value)}
                    type="number"
                    min="0"
                    max="1"
                    step="0.01"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Refresh schedule</Label>
                  <Select
                    value={formSchedule}
                    onValueChange={setFormSchedule}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select schedule" />
                    </SelectTrigger>
                    <SelectContent>
                      {REFRESH_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="grid gap-4 md:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="alert-cooldown">Alert cooldown (seconds)</Label>
                  <Input
                    id="alert-cooldown"
                    value={formCooldown}
                    onChange={(event) => setFormCooldown(event.target.value)}
                    type="number"
                    min="60"
                  />
                </div>
                <div className="md:col-span-3 space-y-2">
                  <Label>Alert emails (comma or line separated)</Label>
                  <Textarea
                    value={formEmails}
                    onChange={(event) => setFormEmails(event.target.value)}
                    placeholder="alerts@example.com, ops@example.com"
                    rows={2}
                  />
                </div>
                <div className="md:col-span-3 space-y-2">
                  <Label>Alert webhooks (comma or line separated)</Label>
                  <Textarea
                    value={formWebhooks}
                    onChange={(event) => setFormWebhooks(event.target.value)}
                    placeholder="https://hooks.slack.com/..."
                    rows={2}
                  />
                </div>
              </div>
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={handleReset}
                  disabled={!defaults}
                >
                  Reset
                </Button>
                <Button
                  onClick={handleSave}
                  disabled={updateMutation.isPending || defaultsLoading}
                >
                  {updateMutation.isPending ? "Saving…" : "Save changes"}
                </Button>
              </div>
            </>
          )}
      </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Alert transports</CardTitle>
          <p className="text-sm text-muted-foreground">
            Manage SMTP and webhook delivery for budget alerts. Leave fields blank to disable a channel.
          </p>
        </CardHeader>
        <CardContent className="space-y-6">
          {alertSettingsQuery.isLoading ? (
            <Skeleton className="h-40 w-full" />
          ) : (
            <>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="smtp-host">SMTP host</Label>
                  <Input
                    id="smtp-host"
                    value={smtpHost}
                    onChange={(event) => setSmtpHost(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-port">SMTP port</Label>
                  <Input
                    id="smtp-port"
                    type="number"
                    value={smtpPort}
                    onChange={(event) => setSmtpPort(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-username">SMTP username</Label>
                  <Input
                    id="smtp-username"
                    value={smtpUsername}
                    onChange={(event) => setSmtpUsername(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-password">SMTP password / token</Label>
                  <Input
                    id="smtp-password"
                    type="password"
                    value={smtpPassword}
                    onChange={(event) => setSmtpPassword(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-from">From address</Label>
                  <Input
                    id="smtp-from"
                    value={smtpFrom}
                    onChange={(event) => setSmtpFrom(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-timeout">Connect timeout (seconds)</Label>
                  <Input
                    id="smtp-timeout"
                    type="number"
                    value={smtpTimeout}
                    onChange={(event) => setSmtpTimeout(event.target.value)}
                  />
                </div>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="flex items-center justify-between rounded-lg border p-3">
                  <div>
                    <p className="text-sm font-medium">Use TLS / STARTTLS</p>
                    <p className="text-xs text-muted-foreground">
                      Attempts STARTTLS if supported by the server.
                    </p>
                  </div>
                  <Switch checked={smtpUseTLS} onCheckedChange={setSmtpUseTLS} />
                </div>
                <div className="flex items-center justify-between rounded-lg border p-3">
                  <div>
                    <p className="text-sm font-medium">Skip TLS verification</p>
                    <p className="text-xs text-muted-foreground">
                      Only enable for local/self-signed servers.
                    </p>
                  </div>
                  <Switch
                    checked={smtpSkipVerify}
                    onCheckedChange={setSmtpSkipVerify}
                  />
                </div>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="webhook-timeout">Webhook timeout (seconds)</Label>
                  <Input
                    id="webhook-timeout"
                    type="number"
                    value={webhookTimeout}
                    onChange={(event) => setWebhookTimeout(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="webhook-retries">Webhook max retries</Label>
                  <Input
                    id="webhook-retries"
                    type="number"
                    value={webhookRetries}
                    onChange={(event) => setWebhookRetries(event.target.value)}
                  />
                </div>
              </div>
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={() => alertSettingsQuery.refetch()}
                  disabled={alertSettingsQuery.isLoading}
                >
                  Reset
                </Button>
                <Button
                  onClick={() =>
                    alertSettingsMutation.mutate({
                      smtp: {
                        host: smtpHost,
                        port: Number(smtpPort) || 0,
                        username: smtpUsername,
                        password: smtpPassword,
                        from: smtpFrom,
                        use_tls: smtpUseTLS,
                        skip_tls_verify: smtpSkipVerify,
                        connect_timeout_seconds:
                          Number(smtpTimeout) >= 0 ? Number(smtpTimeout) : 0,
                      },
                      webhook: {
                        timeout_seconds:
                          Number(webhookTimeout) >= 0 ? Number(webhookTimeout) : 0,
                        max_retries:
                          Number(webhookRetries) >= 0 ? Number(webhookRetries) : 0,
                      },
                    })
                  }
                  disabled={alertSettingsMutation.isPending}
                >
                  {alertSettingsMutation.isPending ? "Saving…" : "Save changes"}
                </Button>
              </div>
            </>
          )}
          <div className="space-y-2 border-t pt-4">
            <Label htmlFor="test-email">Send test email</Label>
            <div className="flex flex-col gap-2 md:flex-row">
              <Input
                id="test-email"
                placeholder="alerts@example.com"
                value={testEmail}
                onChange={(event) => setTestEmail(event.target.value)}
                className="md:flex-1"
              />
              <Button
                type="button"
                variant="outline"
                disabled={!testEmail || testEmailMutation.isPending}
                onClick={() => testEmailMutation.mutate(testEmail)}
              >
                {testEmailMutation.isPending ? "Sending…" : "Send test"}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Rate limit defaults</CardTitle>
          <p className="text-sm text-muted-foreground">
            Define the baseline RPM/TPM/parallel ceilings applied to every API key
            and tenant unless an override is configured.
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          {rateLimitLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-6 w-1/3" />
              <Skeleton className="h-6 w-1/2" />
              <Skeleton className="h-6 w-2/3" />
            </div>
          ) : (
            <>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="requests-per-minute">Requests per minute</Label>
                  <Input
                    id="requests-per-minute"
                    type="number"
                    min="1"
                    value={formRequestsPerMinute}
                    onChange={(event) => setFormRequestsPerMinute(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="tokens-per-minute">Tokens per minute</Label>
                  <Input
                    id="tokens-per-minute"
                    type="number"
                    min="1"
                    value={formTokensPerMinute}
                    onChange={(event) => setFormTokensPerMinute(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="parallel-key">Parallel requests (per key)</Label>
                  <Input
                    id="parallel-key"
                    type="number"
                    min="1"
                    value={formParallelKey}
                    onChange={(event) => setFormParallelKey(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="parallel-tenant">Parallel requests (per tenant)</Label>
                  <Input
                    id="parallel-tenant"
                    type="number"
                    min="1"
                    value={formParallelTenant}
                    onChange={(event) => setFormParallelTenant(event.target.value)}
                  />
                </div>
              </div>
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={handleRateLimitReset}
                  disabled={!rateLimitDefaults}
                >
                  Reset
                </Button>
                <Button
                  onClick={handleRateLimitSave}
                  disabled={rateLimitMutation.isPending || rateLimitLoading}
                >
                  {rateLimitMutation.isPending ? "Saving…" : "Save changes"}
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>File uploads</CardTitle>
          <p className="text-sm text-muted-foreground">
            Define the platform-wide guardrails for file uploads used by batch
            jobs and fine-tuning workflows.
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          {fileSettingsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-6 w-1/3" />
              <Skeleton className="h-6 w-1/2" />
              <Skeleton className="h-6 w-2/3" />
            </div>
          ) : (
            <>
              <div className="grid gap-4 md:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="file-max-size">Max upload size (MB)</Label>
                  <Input
                    id="file-max-size"
                    type="number"
                    min="1"
                    value={fileMaxSize}
                    onChange={(event) => setFileMaxSize(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="file-default-ttl">Default TTL (seconds)</Label>
                  <Input
                    id="file-default-ttl"
                    type="number"
                    min="60"
                    value={fileDefaultTTL}
                    onChange={(event) => setFileDefaultTTL(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="file-max-ttl">Max TTL (seconds)</Label>
                  <Input
                    id="file-max-ttl"
                    type="number"
                    min="60"
                    value={fileMaxTTL}
                    onChange={(event) => setFileMaxTTL(event.target.value)}
                  />
                </div>
              </div>
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={handleFileSettingsReset}
                  disabled={!fileSettings}
                >
                  Reset
                </Button>
                <Button
                  onClick={handleFileSettingsSave}
                  disabled={fileSettingsMutation.isPending || fileSettingsLoading}
                >
                  {fileSettingsMutation.isPending ? "Saving…" : "Save changes"}
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Batch jobs</CardTitle>
          <p className="text-sm text-muted-foreground">
            Tune the max requests, concurrency, and retention window enforced
            for /v1/batches submissions.
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          {batchSettingsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-6 w-1/3" />
              <Skeleton className="h-6 w-1/2" />
              <Skeleton className="h-6 w-2/3" />
            </div>
          ) : (
            <>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="batch-max-requests">Max requests per batch</Label>
                  <Input
                    id="batch-max-requests"
                    type="number"
                    min="1"
                    value={batchMaxRequests}
                    onChange={(event) => setBatchMaxRequests(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="batch-max-concurrency">Max concurrency</Label>
                  <Input
                    id="batch-max-concurrency"
                    type="number"
                    min="1"
                    value={batchMaxConcurrency}
                    onChange={(event) => setBatchMaxConcurrency(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="batch-default-ttl">Default TTL (seconds)</Label>
                  <Input
                    id="batch-default-ttl"
                    type="number"
                    min="60"
                    value={batchDefaultTTL}
                    onChange={(event) => setBatchDefaultTTL(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="batch-max-ttl">Max TTL (seconds)</Label>
                  <Input
                    id="batch-max-ttl"
                    type="number"
                    min="60"
                    value={batchMaxTTL}
                    onChange={(event) => setBatchMaxTTL(event.target.value)}
                  />
                </div>
              </div>
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={handleBatchSettingsReset}
                  disabled={!batchSettings}
                >
                  Reset
                </Button>
                <Button
                  onClick={handleBatchSettingsSave}
                  disabled={batchSettingsMutation.isPending || batchSettingsLoading}
                >
                  {batchSettingsMutation.isPending ? "Saving…" : "Save changes"}
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Default models</CardTitle>
          <p className="text-sm text-muted-foreground">
            These aliases are granted automatically to every personal tenant.
            Remove access here to hide models from users unless a tenant override
            explicitly re-enables them.
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          {modelsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-6 w-1/3" />
              <Skeleton className="h-6 w-1/2" />
              <Skeleton className="h-6 w-2/3" />
            </div>
          ) : (
            <>
              <div className="flex flex-wrap gap-2">
                {defaultModels.length ? (
                  defaultModels.map((alias) => {
                    const meta = catalogByAlias.get(alias);
                    const label = meta ? `${alias} · ${meta.provider}` : alias;
                    return (
                      <Badge
                        key={alias}
                        variant="secondary"
                        className="flex items-center gap-2"
                      >
                        <span>{label}</span>
                        <button
                          type="button"
                          className="rounded-full p-0.5 hover:text-destructive focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring"
                          onClick={() => handleRemoveModel(alias)}
                          disabled={removeModelMutation.isPending}
                          aria-label={`Remove ${alias}`}
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </Badge>
                    );
                  })
                ) : (
                  <p className="text-sm text-muted-foreground">
                    No default models selected yet.
                  </p>
                )}
              </div>

              <div className="flex flex-col gap-3 md:flex-row md:items-end">
                <div className="md:w-96 space-y-2">
                  <Label htmlFor="default-model-select">Add model</Label>
                  <Select
                    value={newDefaultModel}
                    onValueChange={setNewDefaultModel}
                    disabled={availableModelOptions.length === 0 || addModelMutation.isPending}
                  >
                    <SelectTrigger id="default-model-select">
                      <SelectValue placeholder={availableModelOptions.length ? "Select model" : "All enabled models already granted"} />
                    </SelectTrigger>
                    <SelectContent>
                      {availableModelOptions.map((entry) => (
                        <SelectItem key={entry.alias} value={entry.alias}>
                          {entry.alias} · {entry.provider}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <Button
                  onClick={handleAddModel}
                  disabled={!newDefaultModel || addModelMutation.isPending}
                >
                  {addModelMutation.isPending ? "Adding…" : "Add model"}
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Budget workflow</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-muted-foreground">
          <p>
            Tenant budgets, warning thresholds, alert channels, and model access
            policies are configured per-tenant from the Tenants page. Those
            values are persisted in Postgres and enforced by the router on every
            request.
          </p>
          <p>
            Defaults shown above come from the active configuration and provide
            the fallback when a tenant does not have its own override.
          </p>
          <p>
            Looking for observability hooks or advanced provider knobs? Those
            settings still live in configuration for now and will gain UI
            controls in a future iteration.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

function parseListInput(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((entry) => entry.trim())
    .filter(Boolean);
}
