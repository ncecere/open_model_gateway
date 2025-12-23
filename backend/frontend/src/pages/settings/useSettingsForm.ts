import { useCallback, useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";

import type { BudgetDefaults } from "@/api/budgets";
import type { RateLimitDefaults } from "@/api/rate-limits";
import type {
  AlertSettingsPayload,
  BatchSettingsPayload,
  FileSettingsPayload,
} from "@/api/runtime-settings";

export type SettingsFormValues = {
  budgetDefaultUsd: string;
  budgetWarningThreshold: string;
  budgetRefreshSchedule: string;
  budgetAlertCooldown: string;
  budgetAlertEmails: string;
  budgetAlertWebhooks: string;
  rateRequestsPerMinute: string;
  rateTokensPerMinute: string;
  rateParallelKey: string;
  rateParallelTenant: string;
  fileMaxSizeMb: string;
  fileDefaultTtlSeconds: string;
  fileMaxTtlSeconds: string;
  batchMaxRequests: string;
  batchMaxConcurrency: string;
  batchDefaultTtlSeconds: string;
  batchMaxTtlSeconds: string;
  smtpHost: string;
  smtpPort: string;
  smtpUsername: string;
  smtpPassword: string;
  smtpFrom: string;
  smtpUseTLS: boolean;
  smtpSkipVerify: boolean;
  smtpTimeout: string;
  webhookTimeout: string;
  webhookRetries: string;
};

const emptyValues: SettingsFormValues = {
  budgetDefaultUsd: "",
  budgetWarningThreshold: "",
  budgetRefreshSchedule: "calendar_month",
  budgetAlertCooldown: "3600",
  budgetAlertEmails: "",
  budgetAlertWebhooks: "",
  rateRequestsPerMinute: "",
  rateTokensPerMinute: "",
  rateParallelKey: "",
  rateParallelTenant: "",
  fileMaxSizeMb: "",
  fileDefaultTtlSeconds: "",
  fileMaxTtlSeconds: "",
  batchMaxRequests: "",
  batchMaxConcurrency: "",
  batchDefaultTtlSeconds: "",
  batchMaxTtlSeconds: "",
  smtpHost: "",
  smtpPort: "",
  smtpUsername: "",
  smtpPassword: "",
  smtpFrom: "",
  smtpUseTLS: true,
  smtpSkipVerify: false,
  smtpTimeout: "",
  webhookTimeout: "",
  webhookRetries: "",
};

function applyFormValues(
  setValue: (
    name: keyof SettingsFormValues,
    value: SettingsFormValues[keyof SettingsFormValues],
    options?: { shouldDirty?: boolean; shouldTouch?: boolean },
  ) => void,
  values: Partial<SettingsFormValues>,
) {
  (Object.entries(values) as Array<[keyof SettingsFormValues, SettingsFormValues[keyof SettingsFormValues]]>).forEach(
    ([key, value]) => {
      setValue(key, value, { shouldDirty: false, shouldTouch: false });
    },
  );
}

function mapBudgetDefaults(defaults: BudgetDefaults | undefined): Partial<SettingsFormValues> {
  if (!defaults) {
    return {};
  }
  return {
    budgetDefaultUsd: defaults.default_usd.toString(),
    budgetWarningThreshold: defaults.warning_threshold_perc.toString(),
    budgetRefreshSchedule: defaults.refresh_schedule,
    budgetAlertCooldown: (defaults.alert.cooldown_seconds ?? 3600).toString(),
    budgetAlertEmails: (defaults.alert.emails ?? []).join(", "),
    budgetAlertWebhooks: (defaults.alert.webhooks ?? []).join(", "),
  };
}

function mapRateLimits(rateLimits: RateLimitDefaults | undefined): Partial<SettingsFormValues> {
  if (!rateLimits) {
    return {};
  }
  return {
    rateRequestsPerMinute: rateLimits.requests_per_minute.toString(),
    rateTokensPerMinute: rateLimits.tokens_per_minute.toString(),
    rateParallelKey: rateLimits.parallel_requests_key.toString(),
    rateParallelTenant: rateLimits.parallel_requests_tenant.toString(),
  };
}

function mapFileSettings(settings: FileSettingsPayload | undefined): Partial<SettingsFormValues> {
  if (!settings) {
    return {};
  }
  return {
    fileMaxSizeMb: settings.max_size_mb.toString(),
    fileDefaultTtlSeconds: settings.default_ttl_seconds.toString(),
    fileMaxTtlSeconds: settings.max_ttl_seconds.toString(),
  };
}

function mapBatchSettings(settings: BatchSettingsPayload | undefined): Partial<SettingsFormValues> {
  if (!settings) {
    return {};
  }
  return {
    batchMaxRequests: settings.max_requests.toString(),
    batchMaxConcurrency: settings.max_concurrency.toString(),
    batchDefaultTtlSeconds: settings.default_ttl_seconds.toString(),
    batchMaxTtlSeconds: settings.max_ttl_seconds.toString(),
  };
}

function mapAlertSettings(settings: AlertSettingsPayload | undefined): Partial<SettingsFormValues> {
  if (!settings) {
    return {};
  }
  return {
    smtpHost: settings.smtp?.host ?? "",
    smtpPort: settings.smtp?.port != null ? settings.smtp.port.toString() : "",
    smtpUsername: settings.smtp?.username ?? "",
    smtpPassword: settings.smtp?.password ?? "",
    smtpFrom: settings.smtp?.from ?? "",
    smtpUseTLS: Boolean(settings.smtp?.use_tls),
    smtpSkipVerify: Boolean(settings.smtp?.skip_tls_verify),
    smtpTimeout:
      settings.smtp?.connect_timeout_seconds != null
        ? settings.smtp.connect_timeout_seconds.toString()
        : "",
    webhookTimeout:
      settings.webhook?.timeout_seconds != null
        ? settings.webhook.timeout_seconds.toString()
        : "",
    webhookRetries:
      settings.webhook?.max_retries != null
        ? settings.webhook.max_retries.toString()
        : "",
  };
}

export function useSettingsForm({
  budgetDefaults,
  rateLimitDefaults,
  fileSettings,
  batchSettings,
  alertSettings,
}: {
  budgetDefaults?: BudgetDefaults;
  rateLimitDefaults?: RateLimitDefaults;
  fileSettings?: FileSettingsPayload;
  batchSettings?: BatchSettingsPayload;
  alertSettings?: AlertSettingsPayload;
}) {
  const form = useForm<SettingsFormValues>({
    defaultValues: emptyValues,
    mode: "onChange",
  });

  const resetBudget = useCallback(() => {
    applyFormValues(form.setValue, mapBudgetDefaults(budgetDefaults));
  }, [budgetDefaults, form.setValue]);

  const resetRateLimits = useCallback(() => {
    applyFormValues(form.setValue, mapRateLimits(rateLimitDefaults));
  }, [rateLimitDefaults, form.setValue]);

  const resetFiles = useCallback(() => {
    applyFormValues(form.setValue, mapFileSettings(fileSettings));
  }, [fileSettings, form.setValue]);

  const resetBatches = useCallback(() => {
    applyFormValues(form.setValue, mapBatchSettings(batchSettings));
  }, [batchSettings, form.setValue]);

  const resetAlerts = useCallback(() => {
    applyFormValues(form.setValue, mapAlertSettings(alertSettings));
  }, [alertSettings, form.setValue]);

  useEffect(() => {
    resetBudget();
  }, [resetBudget]);

  useEffect(() => {
    resetRateLimits();
  }, [resetRateLimits]);

  useEffect(() => {
    resetFiles();
  }, [resetFiles]);

  useEffect(() => {
    resetBatches();
  }, [resetBatches]);

  useEffect(() => {
    resetAlerts();
  }, [resetAlerts]);

  const resetAll = useCallback(() => {
    form.reset(emptyValues, { keepDirty: false });
    resetBudget();
    resetRateLimits();
    resetFiles();
    resetBatches();
    resetAlerts();
  }, [form, resetBudget, resetRateLimits, resetFiles, resetBatches, resetAlerts]);

  return useMemo(
    () => ({
      form,
      resetBudget,
      resetRateLimits,
      resetFiles,
      resetBatches,
      resetAlerts,
      resetAll,
    }),
    [form, resetBudget, resetRateLimits, resetFiles, resetBatches, resetAlerts, resetAll],
  );
}
