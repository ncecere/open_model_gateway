import { useEffect, useState } from "react";
import { Key } from "lucide-react";

import type {
  CreateApiKeyRequest,
  CreateApiKeyResponse,
  TenantRecord,
} from "@/api/tenants";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { useToast } from "@/hooks/use-toast";

export type TenantBudgetInfo = {
  limit: number | null;
  warning: number | null;
};

export type RateLimitInfo = {
  requests_per_minute: number;
  tokens_per_minute: number;
  parallel_requests: number;
};

export type AdminKeyCreateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selectedTenantId: string | undefined;
  onTenantChange: (tenantId: string) => void;
  tenants: TenantRecord[];
  tenantBudgetMap: Map<string, TenantBudgetInfo>;
  budgetDefaults: { default_usd?: number } | undefined;
  defaultKeyRateLimit: RateLimitInfo | null;
  effectiveTenantRateLimit: RateLimitInfo | null;
  isSubmitting: boolean;
  onSubmit: (
    tenantId: string,
    payload: CreateApiKeyRequest,
  ) => Promise<CreateApiKeyResponse>;
};

export function AdminKeyCreateDialog({
  open,
  onOpenChange,
  selectedTenantId,
  onTenantChange,
  tenants,
  tenantBudgetMap,
  budgetDefaults,
  defaultKeyRateLimit,
  effectiveTenantRateLimit,
  isSubmitting,
  onSubmit,
}: AdminKeyCreateDialogProps) {
  const { toast } = useToast();

  const [keyName, setKeyName] = useState("");
  const [budgetUsd, setBudgetUsd] = useState("");
  const [warningThreshold, setWarningThreshold] = useState("");
  const [requestsPerMinute, setRequestsPerMinute] = useState("");
  const [tokensPerMinute, setTokensPerMinute] = useState("");
  const [parallelRequests, setParallelRequests] = useState("");

  useEffect(() => {
    if (!open) {
      setKeyName("");
      setBudgetUsd("");
      setWarningThreshold("");
      setRequestsPerMinute("");
      setTokensPerMinute("");
      setParallelRequests("");
    }
  }, [open]);

  const selectedTenantBudgetLimit =
    (selectedTenantId && tenantBudgetMap.get(selectedTenantId)?.limit) ?? null;

  const handleCreateKey = async () => {
    if (!selectedTenantId) return;
    if (!keyName.trim()) {
      toast({ variant: "destructive", title: "Name is required" });
      return;
    }

    const payload: CreateApiKeyRequest = {
      name: keyName.trim(),
    };

    const parsedBudget = Number(budgetUsd);
    const parsedThreshold = Number(warningThreshold);
    const tenantBudgetLimit =
      tenantBudgetMap.get(selectedTenantId)?.limit ??
      budgetDefaults?.default_usd ??
      0;

    if (budgetUsd && Number.isFinite(parsedBudget)) {
      if (tenantBudgetLimit > 0 && parsedBudget > tenantBudgetLimit) {
        toast({
          variant: "destructive",
          title: `Budget exceeds tenant cap ($${tenantBudgetLimit.toFixed(2)})`,
        });
        return;
      }
      payload.quota = {
        budget_usd: parsedBudget,
        warning_threshold: Number.isFinite(parsedThreshold)
          ? parsedThreshold
          : undefined,
      };
    } else if (warningThreshold) {
      payload.quota = {
        warning_threshold: Number.isFinite(parsedThreshold)
          ? parsedThreshold
          : undefined,
      };
    }

    const trimmedRPM = requestsPerMinute.trim();
    const trimmedTPM = tokensPerMinute.trim();
    const trimmedParallel = parallelRequests.trim();
    const hasRateOverride =
      trimmedRPM.length > 0 || trimmedTPM.length > 0 || trimmedParallel.length > 0;

    if (hasRateOverride) {
      const rpmValue = Number.parseInt(trimmedRPM, 10);
      const tpmValue = Number.parseInt(trimmedTPM, 10);
      const parallelValue = Number.parseInt(trimmedParallel, 10);

      if (
        !Number.isFinite(rpmValue) ||
        !Number.isFinite(tpmValue) ||
        !Number.isFinite(parallelValue) ||
        rpmValue <= 0 ||
        tpmValue <= 0 ||
        parallelValue <= 0
      ) {
        toast({
          variant: "destructive",
          title: "Rate limits must be positive integers",
        });
        return;
      }

      const keyMaxRPM = defaultKeyRateLimit?.requests_per_minute ?? 0;
      const keyMaxTPM = defaultKeyRateLimit?.tokens_per_minute ?? 0;
      const keyMaxParallel = defaultKeyRateLimit?.parallel_requests ?? 0;
      const tenantMaxRPM = effectiveTenantRateLimit?.requests_per_minute ?? 0;
      const tenantMaxTPM = effectiveTenantRateLimit?.tokens_per_minute ?? 0;
      const tenantMaxParallel = effectiveTenantRateLimit?.parallel_requests ?? 0;

      if (keyMaxRPM > 0 && rpmValue > keyMaxRPM) {
        toast({
          variant: "destructive",
          title: `RPM exceeds key default (${keyMaxRPM})`,
        });
        return;
      }
      if (tenantMaxRPM > 0 && rpmValue > tenantMaxRPM) {
        toast({
          variant: "destructive",
          title: `RPM exceeds tenant cap (${tenantMaxRPM})`,
        });
        return;
      }
      if (keyMaxTPM > 0 && tpmValue > keyMaxTPM) {
        toast({
          variant: "destructive",
          title: `TPM exceeds key default (${keyMaxTPM})`,
        });
        return;
      }
      if (tenantMaxTPM > 0 && tpmValue > tenantMaxTPM) {
        toast({
          variant: "destructive",
          title: `TPM exceeds tenant cap (${tenantMaxTPM})`,
        });
        return;
      }
      if (keyMaxParallel > 0 && parallelValue > keyMaxParallel) {
        toast({
          variant: "destructive",
          title: `Parallel requests exceed key default (${keyMaxParallel})`,
        });
        return;
      }
      if (tenantMaxParallel > 0 && parallelValue > tenantMaxParallel) {
        toast({
          variant: "destructive",
          title: `Parallel requests exceed tenant cap (${tenantMaxParallel})`,
        });
        return;
      }

      payload.rate_limits = {
        requests_per_minute: rpmValue,
        tokens_per_minute: tpmValue,
        parallel_requests: parallelValue,
      };
    }

    try {
      await onSubmit(selectedTenantId, payload);
      setKeyName("");
      setBudgetUsd("");
      setWarningThreshold("");
      setRequestsPerMinute("");
      setTokensPerMinute("");
      setParallelRequests("");
      onOpenChange(false);
    } catch (error) {
      console.error(error);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>
        <Button disabled={!selectedTenantId}>
          <Key className="mr-2 h-4 w-4" /> Generate key
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Generate API key</DialogTitle>
          <DialogDescription>
            Issue a new key for the selected tenant. Secrets are shown
            once—store them securely.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="tenant-select">Tenant</Label>
            <Select
              value={selectedTenantId}
              onValueChange={onTenantChange}
              disabled={isSubmitting}
            >
              <SelectTrigger id="tenant-select">
                <SelectValue placeholder="Select tenant" />
              </SelectTrigger>
              <SelectContent>
                {tenants.map((tenant) => (
                  <SelectItem key={tenant.id} value={tenant.id}>
                    {tenant.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="key-name">Key name</Label>
            <Input
              id="key-name"
              value={keyName}
              onChange={(event) => setKeyName(event.target.value)}
              placeholder="Production gateway"
            />
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="budget">Monthly budget (USD)</Label>
              <Input
                id="budget"
                value={budgetUsd}
                onChange={(event) => setBudgetUsd(event.target.value)}
                placeholder={
                  selectedTenantBudgetLimit
                    ? `${selectedTenantBudgetLimit.toFixed(2)}`
                    : budgetDefaults?.default_usd
                      ? `${budgetDefaults.default_usd}`
                      : "Budget"
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="threshold">Warning threshold (0-1)</Label>
              <Input
                id="threshold"
                value={warningThreshold}
                onChange={(event) => setWarningThreshold(event.target.value)}
                placeholder="0.8"
              />
            </div>
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Rate limit override (optional)</Label>
              <p className="text-xs text-muted-foreground">
                Max per key: {defaultKeyRateLimit?.requests_per_minute ?? "—"}{" "}
                RPM / {defaultKeyRateLimit?.tokens_per_minute ?? "—"} TPM /{" "}
                {defaultKeyRateLimit?.parallel_requests ?? "—"} parallel
              </p>
            </div>
            {effectiveTenantRateLimit ? (
              <p className="text-xs text-muted-foreground">
                Tenant cap: {effectiveTenantRateLimit.requests_per_minute} RPM ·{" "}
                {effectiveTenantRateLimit.tokens_per_minute} TPM ·{" "}
                {effectiveTenantRateLimit.parallel_requests} parallel
              </p>
            ) : null}
            <div className="grid gap-4 md:grid-cols-3">
              <Input
                id="rpm"
                value={requestsPerMinute}
                onChange={(event) => setRequestsPerMinute(event.target.value)}
                placeholder={
                  defaultKeyRateLimit?.requests_per_minute
                    ? `${defaultKeyRateLimit.requests_per_minute}`
                    : "RPM"
                }
                aria-label="Requests per minute"
              />
              <Input
                id="tpm"
                value={tokensPerMinute}
                onChange={(event) => setTokensPerMinute(event.target.value)}
                placeholder={
                  defaultKeyRateLimit?.tokens_per_minute
                    ? `${defaultKeyRateLimit.tokens_per_minute}`
                    : "Tokens per minute"
                }
                aria-label="Tokens per minute"
              />
              <Input
                id="parallel"
                value={parallelRequests}
                onChange={(event) => setParallelRequests(event.target.value)}
                placeholder={
                  defaultKeyRateLimit?.parallel_requests
                    ? `${defaultKeyRateLimit.parallel_requests}`
                    : "Parallel requests"
                }
                aria-label="Parallel requests"
              />
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button
            onClick={handleCreateKey}
            disabled={isSubmitting || !selectedTenantId}
          >
            {isSubmitting ? "Issuing…" : "Generate"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
