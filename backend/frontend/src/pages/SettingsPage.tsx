import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FormProvider } from "react-hook-form";

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
import {
  createAdminKey,
  listAdminKeys,
  revokeAdminKey,
  type AdminKeyRecord,
  type AdminKeyScope,
} from "@/api/admin-keys";
import { useToast } from "@/hooks/use-toast";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { BudgetSettingsTab } from "@/pages/settings/BudgetSettingsTab";
import { RateLimitSettingsTab } from "@/pages/settings/RateLimitSettingsTab";
import { FileSettingsTab } from "@/pages/settings/FileSettingsTab";
import { BatchSettingsTab } from "@/pages/settings/BatchSettingsTab";
import { AlertSettingsTab } from "@/pages/settings/AlertSettingsTab";
import { DefaultModelsTab } from "@/pages/settings/DefaultModelsTab";
import { AdminKeysTab } from "@/pages/settings/AdminKeysTab";
import { OverviewSettingsTab } from "@/pages/settings/OverviewSettingsTab";
import { useSettingsForm } from "@/pages/settings/useSettingsForm";

const REFRESH_OPTIONS = [
  { value: "calendar_month", label: "Calendar month" },
  { value: "weekly", label: "Weekly" },
  { value: "rolling_7d", label: "Rolling 7 days" },
  { value: "rolling_30d", label: "Rolling 30 days" },
];

const TEST_EMAIL_OPTIONS = [
  { value: "budget", label: "Budget alert" },
  { value: "invite", label: "Invite preview" },
  { value: "smtp", label: "Transport smoke test" },
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
  const adminKeysQuery = useQuery({
    queryKey: ["admin-keys"],
    queryFn: listAdminKeys,
  });

  const [newDefaultModel, setNewDefaultModel] = useState("");
  const [activeTab, setActiveTab] = useState("budgets");
  const [testEmail, setTestEmail] = useState("");
  const [createAdminKeyOpen, setCreateAdminKeyOpen] = useState(false);
  const [issuedAdminToken, setIssuedAdminToken] = useState<string | null>(null);
  const [adminKeyName, setAdminKeyName] = useState("");
  const [adminKeyScope, setAdminKeyScope] = useState<AdminKeyScope>("admin");
  const [adminKeyExpiresDays, setAdminKeyExpiresDays] = useState("30");
  const [pendingAdminRevoke, setPendingAdminRevoke] =
    useState<AdminKeyRecord | null>(null);

  const rateLimitDefaults = rateLimitQuery.data;
  const fileSettings = fileSettingsQuery.data;
  const batchSettings = batchSettingsQuery.data;
  const alertSettings = alertSettingsQuery.data;

  const settingsForm = useSettingsForm({
    budgetDefaults: defaults,
    rateLimitDefaults,
    fileSettings,
    batchSettings,
    alertSettings,
  });
  const { form, resetBudget, resetRateLimits, resetFiles, resetBatches } =
    settingsForm;

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

  const [testEmailType, setTestEmailType] = useState("budget");

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

  const createAdminKeyMutation = useMutation({
    mutationFn: createAdminKey,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["admin-keys"] });
      setIssuedAdminToken(data.token);
      setCreateAdminKeyOpen(false);
      setAdminKeyName("");
      setAdminKeyScope("admin");
      setAdminKeyExpiresDays("30");
      toast({ title: "Admin key issued" });
    },
    onError: (error: unknown) => {
      toast({
        variant: "destructive",
        title: "Failed to create admin key",
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const revokeAdminKeyMutation = useMutation({
    mutationFn: (id: string) => revokeAdminKey(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-keys"] });
      setPendingAdminRevoke(null);
      toast({ title: "Admin key revoked" });
    },
    onError: () => {
      toast({
        variant: "destructive",
        title: "Failed to revoke admin key",
        description: "Try again in a moment.",
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

  const handleSave = () => {
    const values = form.getValues();
    const budgetValue = Number.parseFloat(values.budgetDefaultUsd);
    const thresholdValue = Number.parseFloat(values.budgetWarningThreshold);
    const cooldownValue = Number.parseInt(values.budgetAlertCooldown, 10);
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
      refresh_schedule: values.budgetRefreshSchedule,
      alert_emails: parseListInput(values.budgetAlertEmails),
      alert_webhooks: parseListInput(values.budgetAlertWebhooks),
      alert_cooldown_seconds: cooldownValue,
    });
  };

  const handleRateLimitSave = () => {
    const values = form.getValues();
    const rpm = Number.parseInt(values.rateRequestsPerMinute, 10);
    const tpm = Number.parseInt(values.rateTokensPerMinute, 10);
    const parallelKey = Number.parseInt(values.rateParallelKey, 10);
    const parallelTenant = Number.parseInt(values.rateParallelTenant, 10);
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

  const handleFileSettingsSave = () => {
    const values = form.getValues();
    const maxSize = Number.parseInt(values.fileMaxSizeMb, 10);
    const defaultTTL = Number.parseInt(values.fileDefaultTtlSeconds, 10);
    const maxTTL = Number.parseInt(values.fileMaxTtlSeconds, 10);
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

  const handleBatchSettingsSave = () => {
    const values = form.getValues();
    const maxRequests = Number.parseInt(values.batchMaxRequests, 10);
    const maxConcurrency = Number.parseInt(values.batchMaxConcurrency, 10);
    const defaultTTL = Number.parseInt(values.batchDefaultTtlSeconds, 10);
    const maxTTL = Number.parseInt(values.batchMaxTtlSeconds, 10);
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

  const handleAlertSave = () => {
    const values = form.getValues();
    alertSettingsMutation.mutate({
      smtp: {
        host: values.smtpHost,
        port: Number(values.smtpPort) || 0,
        username: values.smtpUsername,
        password: values.smtpPassword,
        from: values.smtpFrom,
        use_tls: values.smtpUseTLS,
        skip_tls_verify: values.smtpSkipVerify,
        connect_timeout_seconds:
          Number(values.smtpTimeout) >= 0 ? Number(values.smtpTimeout) : 0,
      },
      webhook: {
        timeout_seconds:
          Number(values.webhookTimeout) >= 0 ? Number(values.webhookTimeout) : 0,
        max_retries:
          Number(values.webhookRetries) >= 0 ? Number(values.webhookRetries) : 0,
      },
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

  const adminKeys = adminKeysQuery.data ?? [];
  const adminKeysLoading = adminKeysQuery.isLoading;

  const handleCreateAdminKey = () => {
    const name = adminKeyName.trim();
    const days = Number.parseInt(adminKeyExpiresDays, 10);
    if (!name) {
      toast({ variant: "destructive", title: "Name is required" });
      return;
    }
    if (!Number.isFinite(days) || days <= 0) {
      toast({ variant: "destructive", title: "Expiry must be positive" });
      return;
    }
    createAdminKeyMutation.mutate({
      name,
      scope: adminKeyScope,
      expires_in_seconds: days * 24 * 60 * 60,
    });
  };

  const handleCopy = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      toast({ title: "Copied to clipboard" });
    } catch {
      toast({
        variant: "destructive",
        title: "Copy failed",
        description: "Copy the value manually.",
      });
    }
  };

  const formatAdminKeyDate = (value?: string | null) => {
    if (!value) {
      return "—";
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return "—";
    }
    return date.toLocaleString();
  };

  const getAdminKeyStatus = (key: AdminKeyRecord) => {
    if (key.revoked_at) {
      return "revoked";
    }
    if (key.expires_at) {
      const expiresAt = new Date(key.expires_at);
      if (!Number.isNaN(expiresAt.getTime()) && expiresAt.getTime() < Date.now()) {
        return "expired";
      }
    }
    return "active";
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

      <Tabs
        value={activeTab}
        onValueChange={setActiveTab}
        className="space-y-6"
      >
        <TabsList className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-8">
          <TabsTrigger value="budgets">Budgets</TabsTrigger>
          <TabsTrigger value="alerts">Alerts</TabsTrigger>
          <TabsTrigger value="rate-limits">Rate limits</TabsTrigger>
          <TabsTrigger value="files">Files</TabsTrigger>
          <TabsTrigger value="batches">Batches</TabsTrigger>
          <TabsTrigger value="models">Default models</TabsTrigger>
          <TabsTrigger value="admin-keys">Admin tokens</TabsTrigger>
          <TabsTrigger value="overview">Overview</TabsTrigger>
        </TabsList>

        <FormProvider {...form}>
          <BudgetSettingsTab
            loading={defaultsLoading}
            hasDefaults={Boolean(defaults)}
            metadataDescription={budgetMetadataDescription}
            refreshOptions={REFRESH_OPTIONS}
            onReset={resetBudget}
            onSave={handleSave}
            saving={updateMutation.isPending}
          />
          <AlertSettingsTab
            loading={alertSettingsQuery.isLoading}
            onReset={() => alertSettingsQuery.refetch()}
            onSave={handleAlertSave}
            saving={alertSettingsMutation.isPending}
            testEmail={testEmail}
            setTestEmail={setTestEmail}
            testEmailType={testEmailType}
            setTestEmailType={setTestEmailType}
            testEmailOptions={TEST_EMAIL_OPTIONS}
            sendingTest={testEmailMutation.isPending}
            onSendTest={() =>
              testEmailMutation.mutate({
                email: testEmail,
                type: testEmailType,
              })
            }
          />
          <RateLimitSettingsTab
            loading={rateLimitLoading}
            hasDefaults={Boolean(rateLimitDefaults)}
            onReset={resetRateLimits}
            onSave={handleRateLimitSave}
            saving={rateLimitMutation.isPending}
          />
          <FileSettingsTab
            loading={fileSettingsLoading}
            hasDefaults={Boolean(fileSettings)}
            onReset={resetFiles}
            onSave={handleFileSettingsSave}
            saving={fileSettingsMutation.isPending}
          />
          <BatchSettingsTab
            loading={batchSettingsLoading}
            hasDefaults={Boolean(batchSettings)}
            onReset={resetBatches}
            onSave={handleBatchSettingsSave}
            saving={batchSettingsMutation.isPending}
          />
          <DefaultModelsTab
            loading={modelsLoading}
            defaultModels={defaultModels}
            availableModels={availableModelOptions}
            catalogByAlias={catalogByAlias}
            newDefaultModel={newDefaultModel}
            setNewDefaultModel={setNewDefaultModel}
            onAddModel={handleAddModel}
            onRemoveModel={handleRemoveModel}
            addPending={addModelMutation.isPending}
            removePending={removeModelMutation.isPending}
          />
          <AdminKeysTab
            adminKeys={adminKeys}
            loading={adminKeysLoading}
            createOpen={createAdminKeyOpen}
            setCreateOpen={setCreateAdminKeyOpen}
            issuedToken={issuedAdminToken}
            setIssuedToken={setIssuedAdminToken}
            pendingRevoke={pendingAdminRevoke}
            setPendingRevoke={setPendingAdminRevoke}
            adminKeyName={adminKeyName}
            setAdminKeyName={setAdminKeyName}
            adminKeyScope={adminKeyScope}
            setAdminKeyScope={setAdminKeyScope}
            adminKeyExpiresDays={adminKeyExpiresDays}
            setAdminKeyExpiresDays={setAdminKeyExpiresDays}
            createPending={createAdminKeyMutation.isPending}
            revokePending={revokeAdminKeyMutation.isPending}
            onCreate={handleCreateAdminKey}
            onRevoke={(id) => revokeAdminKeyMutation.mutate(id)}
            onCopy={handleCopy}
            formatDate={formatAdminKeyDate}
            getStatus={getAdminKeyStatus}
          />
          <OverviewSettingsTab />
        </FormProvider>
      </Tabs>
    </div>
  );
}

function parseListInput(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((entry) => entry.trim())
    .filter(Boolean);
}
