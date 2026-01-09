import { useState, useEffect } from "react";
import { Key } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useToast } from "@/hooks/use-toast";
import type { CreateUserAPIKeyRequest } from "@/api/user/api-keys";

export type CreateApiKeyDialogProps = {
  mode: "personal" | "tenant";
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (payload: CreateUserAPIKeyRequest) => Promise<void>;
  isSubmitting: boolean;
  budgetLimit?: number | null;
  tenantName?: string;
  disabled?: boolean;
  disabledReason?: string;
};

export function CreateApiKeyDialog({
  mode,
  open,
  onOpenChange,
  onSubmit,
  isSubmitting,
  budgetLimit,
  tenantName,
  disabled,
  disabledReason,
}: CreateApiKeyDialogProps) {
  const { toast } = useToast();
  const [name, setName] = useState("");
  const [budgetUsd, setBudgetUsd] = useState("");
  const [warningThreshold, setWarningThreshold] = useState("");
  const [rpm, setRpm] = useState("");
  const [tpm, setTpm] = useState("");
  const [parallel, setParallel] = useState("");

  useEffect(() => {
    if (!open) {
      setName("");
      setBudgetUsd("");
      setWarningThreshold("");
      setRpm("");
      setTpm("");
      setParallel("");
    }
  }, [open]);

  const handleOpenChange = (nextOpen: boolean) => {
    if (disabled && nextOpen) {
      toast({
        variant: "destructive",
        title: disabledReason ?? "Cannot create key",
      });
      return;
    }
    onOpenChange(nextOpen);
  };

  const handleSubmit = async () => {
    if (!name.trim()) {
      toast({ variant: "destructive", title: "Name is required" });
      return;
    }

    const payload: CreateUserAPIKeyRequest = { name: name.trim() };

    const budgetValue = Number(budgetUsd);
    const thresholdValue = Number(warningThreshold);

    if (
      warningThreshold &&
      (!Number.isFinite(thresholdValue) || thresholdValue <= 0 || thresholdValue > 1)
    ) {
      toast({
        variant: "destructive",
        title: "Warning threshold must be between 0 and 1",
      });
      return;
    }

    if (budgetUsd && Number.isFinite(budgetValue)) {
      if (budgetLimit && budgetValue > budgetLimit) {
        toast({
          variant: "destructive",
          title: `Budget exceeds limit ($${budgetLimit.toFixed(2)})`,
        });
        return;
      }
      payload.quota = {
        budget_usd: budgetValue,
        warning_threshold: Number.isFinite(thresholdValue) ? thresholdValue : undefined,
      };
    } else if (warningThreshold) {
      payload.quota = {
        warning_threshold: Number.isFinite(thresholdValue) ? thresholdValue : undefined,
      };
    }

    const rpmValue = rpm.trim() ? Number.parseInt(rpm.trim(), 10) : undefined;
    const tpmValue = tpm.trim() ? Number.parseInt(tpm.trim(), 10) : undefined;
    const parallelValue = parallel.trim() ? Number.parseInt(parallel.trim(), 10) : undefined;

    if (rpmValue !== undefined || tpmValue !== undefined || parallelValue !== undefined) {
      if (
        !rpmValue ||
        !tpmValue ||
        !parallelValue ||
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
      payload.rate_limits = {
        requests_per_minute: rpmValue,
        tokens_per_minute: tpmValue,
        parallel_requests: parallelValue,
      };
    }

    await onSubmit(payload);
  };

  const isPersonal = mode === "personal";
  const title = isPersonal ? "Create personal API key" : "Create tenant API key";
  const description = isPersonal
    ? "Keys are scoped to your personal tenant and inherit default budgets."
    : tenantName
      ? `Keys are scoped to ${tenantName}.`
      : "Select a tenant to continue.";

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button disabled={disabled}>
          <Key className="mr-2 size-4" />
          {isPersonal ? "Create key" : "Issue key"}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="key-name">Name</Label>
            <Input
              id="key-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={isPersonal ? "My personal key" : "Shared key"}
            />
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="key-budget">Budget (USD)</Label>
              <Input
                id="key-budget"
                value={budgetUsd}
                onChange={(e) => setBudgetUsd(e.target.value)}
                placeholder={budgetLimit ? `${budgetLimit.toFixed(2)}` : "Budget"}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="key-threshold">Warning threshold</Label>
              <Input
                id="key-threshold"
                value={warningThreshold}
                onChange={(e) => setWarningThreshold(e.target.value)}
                placeholder="0.8"
              />
            </div>
          </div>
          <div className="space-y-2">
            <Label>Rate limits (optional)</Label>
            <div className="grid gap-4 md:grid-cols-3">
              <Input
                value={rpm}
                onChange={(e) => setRpm(e.target.value)}
                placeholder="RPM"
                aria-label="Requests per minute"
              />
              <Input
                value={tpm}
                onChange={(e) => setTpm(e.target.value)}
                placeholder="TPM"
                aria-label="Tokens per minute"
              />
              <Input
                value={parallel}
                onChange={(e) => setParallel(e.target.value)}
                placeholder="Parallel"
                aria-label="Parallel requests"
              />
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            Issue key
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
