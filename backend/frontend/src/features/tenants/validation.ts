export type TenantFormValues = {
  name: string;
  budgetUsd: string;
  warningThreshold: string;
  alertCooldown: string;
  requestsPerMinute: string;
  tokensPerMinute: string;
  parallelRequests: string;
  selectedModels: string[];
};

export type ValidationError = {
  field: string;
  message: string;
  description?: string;
};

export type ParsedTenantValues = {
  trimmedName: string;
  trimmedBudget: string;
  trimmedThreshold: string;
  trimmedCooldown: string;
  trimmedRPM: string;
  trimmedTPM: string;
  trimmedParallel: string;
  budgetValue: number;
  thresholdValue: number;
  cooldownValue: number;
  rpmValue: number;
  tpmValue: number;
  parallelValue: number;
  hasRateOverride: boolean;
};

export function parseTenantFormValues(values: TenantFormValues): ParsedTenantValues {
  const trimmedName = values.name.trim();
  const trimmedBudget = values.budgetUsd.trim();
  const trimmedThreshold = values.warningThreshold.trim();
  const trimmedCooldown = values.alertCooldown.trim();
  const trimmedRPM = values.requestsPerMinute.trim();
  const trimmedTPM = values.tokensPerMinute.trim();
  const trimmedParallel = values.parallelRequests.trim();

  return {
    trimmedName,
    trimmedBudget,
    trimmedThreshold,
    trimmedCooldown,
    trimmedRPM,
    trimmedTPM,
    trimmedParallel,
    budgetValue: Number.parseFloat(trimmedBudget),
    thresholdValue: Number.parseFloat(trimmedThreshold),
    cooldownValue: Number.parseInt(trimmedCooldown, 10),
    rpmValue: Number.parseInt(trimmedRPM, 10),
    tpmValue: Number.parseInt(trimmedTPM, 10),
    parallelValue: Number.parseInt(trimmedParallel, 10),
    hasRateOverride: trimmedRPM.length > 0 || trimmedTPM.length > 0 || trimmedParallel.length > 0,
  };
}

export function validateTenantForm(
  values: TenantFormValues,
  options: {
    requireName?: boolean;
    requireModels?: boolean;
    modelCatalogLoading?: boolean;
    modelCatalogEmpty?: boolean;
  } = {},
): ValidationError | null {
  const {
    requireName = true,
    requireModels = true,
    modelCatalogLoading = false,
    modelCatalogEmpty = false,
  } = options;

  const parsed = parseTenantFormValues(values);

  if (requireName && !parsed.trimmedName) {
    return { field: "name", message: "Name is required" };
  }

  if (parsed.trimmedBudget && (!Number.isFinite(parsed.budgetValue) || parsed.budgetValue <= 0)) {
    return { field: "budgetUsd", message: "Budget override must be a positive number" };
  }

  if (
    parsed.trimmedBudget &&
    parsed.trimmedThreshold &&
    (!Number.isFinite(parsed.thresholdValue) || parsed.thresholdValue <= 0 || parsed.thresholdValue > 1)
  ) {
    return { field: "warningThreshold", message: "Warning threshold must be between 0 and 1" };
  }

  if (
    parsed.trimmedBudget &&
    parsed.trimmedCooldown &&
    (!Number.isFinite(parsed.cooldownValue) || parsed.cooldownValue <= 0)
  ) {
    return { field: "alertCooldown", message: "Cooldown must be a positive integer (seconds)" };
  }

  const rateError = validateRateLimits(parsed);
  if (rateError) {
    return rateError;
  }

  if (requireModels && modelCatalogEmpty) {
    return {
      field: "models",
      message: modelCatalogLoading
        ? "Model catalog is still loading"
        : "Add at least one model to the catalog first",
      description: modelCatalogLoading ? "Please wait a moment and try again." : undefined,
    };
  }

  if (requireModels && values.selectedModels.length === 0) {
    return { field: "selectedModels", message: "Select at least one model" };
  }

  return null;
}

export function validateRateLimits(parsed: ParsedTenantValues): ValidationError | null {
  if (!parsed.hasRateOverride) {
    return null;
  }

  if (!parsed.trimmedRPM || !parsed.trimmedTPM || !parsed.trimmedParallel) {
    return {
      field: "rateLimits",
      message: "Provide RPM, TPM, and parallel values",
      description: "All three fields are required for tenant overrides.",
    };
  }

  if (!Number.isFinite(parsed.rpmValue) || parsed.rpmValue <= 0 || !Number.isInteger(parsed.rpmValue)) {
    return { field: "requestsPerMinute", message: "RPM must be a positive integer" };
  }

  if (!Number.isFinite(parsed.tpmValue) || parsed.tpmValue <= 0 || !Number.isInteger(parsed.tpmValue)) {
    return { field: "tokensPerMinute", message: "TPM must be a positive integer" };
  }

  if (!Number.isFinite(parsed.parallelValue) || parsed.parallelValue <= 0 || !Number.isInteger(parsed.parallelValue)) {
    return { field: "parallelRequests", message: "Parallel requests must be a positive integer" };
  }

  return null;
}

export function parseListInput(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((entry) => entry.trim())
    .filter(Boolean);
}

export function normalizeAliases(aliases: string[]): string[] {
  return [...new Set(aliases)].sort((a, b) => a.localeCompare(b));
}

export function aliasSelectionsEqual(a: string[], b: string[]): boolean {
  const left = normalizeAliases(a);
  const right = normalizeAliases(b);
  if (left.length !== right.length) {
    return false;
  }
  return left.every((value, index) => value === right[index]);
}
