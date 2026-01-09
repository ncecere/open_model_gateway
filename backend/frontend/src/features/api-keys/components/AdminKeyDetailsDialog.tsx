import type { ApiKeyRecord } from "@/api/tenants";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { RateLimitCard } from "./RateLimitCard";
import { formatScheduleLabel } from "../utils";
import { shortDateFormatter as dateFormatter } from "@/lib/formatters";

export type AdminKeyDetailsDialogProps = {
  selectedKey: ApiKeyRecord | null;
  onClose: () => void;
  formatBudgetValue: (key: ApiKeyRecord) => string;
  formatWarningThresholdValue: (key: ApiKeyRecord) => string;
};

export function AdminKeyDetailsDialog({
  selectedKey,
  onClose,
  formatBudgetValue,
  formatWarningThresholdValue,
}: AdminKeyDetailsDialogProps) {
  return (
    <Dialog open={Boolean(selectedKey)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[640px]">
        {selectedKey ? (
          <>
            <DialogHeader>
              <DialogTitle>{selectedKey.name}</DialogTitle>
              <DialogDescription className="text-sm text-muted-foreground">
                Issuer:{" "}
                <span className="font-medium text-foreground">
                  {selectedKey.issuer?.label ?? selectedKey.tenant_name ?? "—"}
                </span>
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-6 py-2">
              <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                <span>
                  Issued {dateFormatter.format(new Date(selectedKey.created_at))}
                </span>
                <Badge variant={selectedKey.revoked ? "destructive" : "secondary"}>
                  {selectedKey.revoked ? "revoked" : "active"}
                </Badge>
                <span>
                  Last used{" "}
                  {selectedKey.last_used_at
                    ? dateFormatter.format(new Date(selectedKey.last_used_at))
                    : "—"}
                </span>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">
                    Budget limit
                  </p>
                  <p className="text-lg font-semibold text-foreground">
                    {formatBudgetValue(selectedKey)}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">
                    Warning threshold
                  </p>
                  <p className="text-lg font-semibold text-foreground">
                    {formatWarningThresholdValue(selectedKey)}
                  </p>
                </div>
              </div>
              <div className="space-y-2">
                <p className="text-sm font-medium text-muted-foreground">
                  Budget reset schedule
                </p>
                <Badge variant="outline">
                  {formatScheduleLabel(selectedKey.budget_refresh_schedule)}
                </Badge>
              </div>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <p className="text-sm font-medium text-muted-foreground">
                    Rate limits (RPM/TPM refresh every minute)
                  </p>
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <RateLimitCard
                    title="Per-key limit"
                    details={selectedKey.rate_limits?.key}
                  />
                  <RateLimitCard
                    title="Tenant limit"
                    details={selectedKey.rate_limits?.tenant}
                  />
                </div>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={onClose}>
                Close
              </Button>
            </DialogFooter>
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
