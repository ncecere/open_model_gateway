import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, Copy, Pencil } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DataTable, type DataTableColumn } from "@/ui/kit/DataTable";
import { BudgetMeter } from "@/ui/kit/BudgetMeter";
import { statusToneClass, toneFromStatus } from "@/ui/kit/status";
import { getUserTenants, type UserTenantMembership } from "@/api/users";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import type { PersonalTenantRecord } from "../types";

export interface UserDetailsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  record: PersonalTenantRecord | null;
  onEdit?: (record: PersonalTenantRecord) => void;
}

export function UserDetailsDialog({
  open,
  onOpenChange,
  record,
  onEdit,
}: UserDetailsDialogProps) {
  const userTenantsQuery = useQuery<UserTenantMembership[]>({
    queryKey: ["user-tenants", record?.user_id],
    queryFn: () => getUserTenants(record!.user_id),
    enabled: open && Boolean(record?.user_id),
  });

  const membershipColumns: DataTableColumn<UserTenantMembership>[] = useMemo(
    () => [
      {
        header: "Tenant",
        cell: (membership) => membership.tenant_name,
      },
      {
        header: "Role",
        cell: (membership) => (
          <Badge variant="secondary" className="capitalize">
            {membership.role}
          </Badge>
        ),
      },
      {
        header: "Status",
        cell: (membership) => (
          <span className="capitalize">{membership.status}</span>
        ),
      },
      {
        header: "Joined",
        cell: (membership) =>
          new Date(membership.joined_at).toLocaleDateString(),
      },
    ],
    [],
  );

  if (!record) {
    return null;
  }

  const statusTone = toneFromStatus(record.status);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <div className="flex items-start justify-between">
            <div>
              <DialogTitle className="flex items-center gap-2">
                {record.user_name || record.user_email}
                <Badge className={statusToneClass(statusTone)}>
                  {record.status.charAt(0).toUpperCase() + record.status.slice(1)}
                </Badge>
              </DialogTitle>
              <DialogDescription>{record.user_email}</DialogDescription>
            </div>
            {onEdit && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => onEdit(record)}
              >
                <Pencil className="mr-2 h-4 w-4" />
                Edit
              </Button>
            )}
          </div>
        </DialogHeader>

        <div className="space-y-6 py-2">
          <div className="grid gap-4 md:grid-cols-3">
            <StatCard
              label="Status"
              value={record.status.charAt(0).toUpperCase() + record.status.slice(1)}
            />
            <StatCard
              label="Created"
              value={new Date(record.created_at).toLocaleDateString()}
            />
            <StatCard
              label="Tenants"
              value={(record.membership_count ?? 1).toString()}
            />
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <CopyableField label="User ID" value={record.user_id} />
            <CopyableField label="Tenant ID" value={record.tenant_id} />
          </div>

          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium text-muted-foreground">
                Budget overview
              </CardTitle>
            </CardHeader>
            <CardContent>
              <BudgetMeter
                used={record.budget_used_usd ?? 0}
                limit={record.budget_limit_usd ?? 0}
                warningThreshold={record.warning_threshold ?? 0.8}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Tenant memberships</CardTitle>
            </CardHeader>
            <CardContent>
              <DataTable
                data={userTenantsQuery.data ?? []}
                columns={membershipColumns}
                getKey={(membership) => `${membership.tenant_id}-${membership.role}`}
                isLoading={userTenantsQuery.isLoading}
                emptyState="No additional tenant memberships found."
                dense
              />
            </CardContent>
          </Card>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {label}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold text-foreground">{value}</p>
      </CardContent>
    </Card>
  );
}

function CopyableField({ label, value }: { label: string; value: string }) {
  const { copy, copied } = useCopyToClipboard();

  return (
    <div className="flex items-center justify-between rounded-lg border px-3 py-2">
      <div className="min-w-0 flex-1">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="truncate font-mono text-sm">{value}</p>
      </div>
      <Button
        variant="ghost"
        size="icon"
        className="ml-2 h-8 w-8 shrink-0"
        onClick={() => copy(value)}
      >
        {copied ? (
          <Check className="h-4 w-4 text-green-500" />
        ) : (
          <Copy className="h-4 w-4" />
        )}
      </Button>
    </div>
  );
}
