import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { DataTable, type DataTableColumn } from "@/ui/kit/DataTable";
import { getUserTenants, type UserTenantMembership } from "@/api/users";
import type { UserFormState, PersonalTenantRecord, TenantStatus } from "../types";

type TabValue = "basic" | "access" | "limits";

interface ValidationErrors {
  basic: string[];
  access: string[];
  limits: string[];
}

function validateForm(form: UserFormState, _mode: "create" | "edit"): ValidationErrors {
  const errors: ValidationErrors = {
    basic: [],
    access: [],
    limits: [],
  };

  if (!form.email.trim()) {
    errors.basic.push("Email is required");
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
    errors.basic.push("Invalid email format");
  }

  if (!form.name.trim()) {
    errors.basic.push("Name is required");
  }

  return errors;
}

function getErrorCount(errors: ValidationErrors, tab: TabValue): number {
  return errors[tab].length;
}

function hasAnyErrors(errors: ValidationErrors): boolean {
  return Object.values(errors).some((arr) => arr.length > 0);
}

export interface UserEditorDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  form: UserFormState;
  onChange: (form: UserFormState) => void;
  onSubmit: () => void;
  loading: boolean;
  mode: "create" | "edit";
  record?: PersonalTenantRecord | null;
}

export function UserEditorDialog({
  open,
  onOpenChange,
  form,
  onChange,
  onSubmit,
  loading,
  mode,
  record,
}: UserEditorDialogProps) {
  const [activeTab, setActiveTab] = useState<TabValue>("basic");
  const errors = validateForm(form, mode);

  const userTenantsQuery = useQuery<UserTenantMembership[]>({
    queryKey: ["user-tenants", record?.user_id],
    queryFn: () => getUserTenants(record!.user_id),
    enabled: mode === "edit" && Boolean(record?.user_id),
  });

  const membershipColumns: DataTableColumn<UserTenantMembership>[] = [
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
  ];

  const handleSubmit = () => {
    if (hasAnyErrors(errors)) {
      for (const tab of ["basic", "access", "limits"] as TabValue[]) {
        if (errors[tab].length > 0) {
          setActiveTab(tab);
          return;
        }
      }
      return;
    }
    onSubmit();
  };

  const renderTabTrigger = (value: TabValue, label: string) => {
    const errorCount = getErrorCount(errors, value);
    return (
      <TabsTrigger value={value} className="relative">
        {label}
        {errorCount > 0 && (
          <span className="ml-1.5 inline-flex h-4 w-4 items-center justify-center rounded-full bg-destructive text-[10px] font-medium text-destructive-foreground">
            {errorCount}
          </span>
        )}
      </TabsTrigger>
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[85vh] flex-col overflow-hidden sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {mode === "create" ? "New user" : `Edit ${form.name || form.email}`}
          </DialogTitle>
          <DialogDescription>
            {mode === "create"
              ? "Create an account and optionally send an invite email."
              : "Update user settings and status."}
          </DialogDescription>
        </DialogHeader>

        <Tabs
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as TabValue)}
          className="flex h-full min-h-0 flex-col space-y-4"
        >
          <TabsList className="grid w-full grid-cols-3">
            {renderTabTrigger("basic", "Basic Info")}
            {renderTabTrigger("access", "Access")}
            {renderTabTrigger("limits", "Limits")}
          </TabsList>

          <TabsContent
            value="basic"
            className="mt-0 flex-1 min-h-0 space-y-4 overflow-y-auto pr-1"
          >
            <BasicInfoTab
              form={form}
              onChange={onChange}
              mode={mode}
              errors={errors.basic}
            />
          </TabsContent>

          <TabsContent
            value="access"
            className="mt-0 flex-1 min-h-0 space-y-4 overflow-y-auto pr-1"
          >
            <AccessTab
              mode={mode}
              memberships={userTenantsQuery.data ?? []}
              columns={membershipColumns}
              isLoading={userTenantsQuery.isLoading}
            />
          </TabsContent>

          <TabsContent
            value="limits"
            className="mt-0 flex-1 min-h-0 space-y-4 overflow-y-auto pr-1"
          >
            <LimitsTab record={record} mode={mode} />
          </TabsContent>
        </Tabs>

        <DialogFooter className="border-t pt-4">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={loading || hasAnyErrors(errors)}
          >
            {loading
              ? mode === "create"
                ? "Creating..."
                : "Saving..."
              : mode === "create"
                ? "Create user"
                : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface BasicInfoTabProps {
  form: UserFormState;
  onChange: (form: UserFormState) => void;
  mode: "create" | "edit";
  errors: string[];
}

function BasicInfoTab({ form, onChange, mode, errors }: BasicInfoTabProps) {
  return (
    <div className="space-y-4">
      {errors.length > 0 && (
        <div className="flex items-start gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <ul className="list-inside list-disc">
            {errors.map((error, i) => (
              <li key={i}>{error}</li>
            ))}
          </ul>
        </div>
      )}

      <div className="space-y-2">
        <Label htmlFor="user-email">Email</Label>
        <Input
          id="user-email"
          type="email"
          placeholder="user@example.com"
          value={form.email}
          onChange={(e) => onChange({ ...form, email: e.target.value })}
          disabled={mode === "edit"}
        />
        {mode === "edit" && (
          <p className="text-xs text-muted-foreground">
            Email cannot be changed after creation.
          </p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="user-name">Name</Label>
        <Input
          id="user-name"
          placeholder="Full name"
          value={form.name}
          onChange={(e) => onChange({ ...form, name: e.target.value })}
          disabled={mode === "edit"}
        />
        {mode === "edit" && (
          <p className="text-xs text-muted-foreground">
            Name cannot be changed from this interface.
          </p>
        )}
      </div>

      {mode === "create" && (
        <>
          <div className="space-y-2">
            <Label htmlFor="user-password">
              Password{" "}
              <span className="text-xs text-muted-foreground">(optional)</span>
            </Label>
            <Input
              id="user-password"
              type="password"
              placeholder="Set initial password"
              value={form.password}
              onChange={(e) => onChange({ ...form, password: e.target.value })}
            />
            <p className="text-xs text-muted-foreground">
              Leave blank to rely on SSO or send invite instructions without a
              password.
            </p>
          </div>

          <div className="flex items-center justify-between rounded-md border px-3 py-2">
            <div>
              <Label className="text-sm">Send invite email</Label>
              <p className="text-xs text-muted-foreground">
                Email the user with portal links after creation.
              </p>
            </div>
            <Switch
              checked={form.sendInvite}
              onCheckedChange={(checked) =>
                onChange({ ...form, sendInvite: checked })
              }
            />
          </div>
        </>
      )}

      {mode === "edit" && (
        <div className="space-y-2">
          <Label htmlFor="user-status">Status</Label>
          <Select
            value={form.status}
            onValueChange={(value) =>
              onChange({ ...form, status: value as TenantStatus })
            }
          >
            <SelectTrigger id="user-status">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="suspended">Suspended</SelectItem>
            </SelectContent>
          </Select>
          <p className="text-xs text-muted-foreground">
            Suspended users cannot access the API or portal.
          </p>
        </div>
      )}
    </div>
  );
}

interface AccessTabProps {
  mode: "create" | "edit";
  memberships: UserTenantMembership[];
  columns: DataTableColumn<UserTenantMembership>[];
  isLoading: boolean;
}

function AccessTab({ mode, memberships, columns, isLoading }: AccessTabProps) {
  if (mode === "create") {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <p className="text-sm text-muted-foreground">
          Tenant memberships can be configured after the user is created.
        </p>
        <p className="mt-2 text-xs text-muted-foreground">
          A personal tenant will be automatically created for the new user.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h4 className="text-sm font-medium">Tenant memberships</h4>
        <p className="text-xs text-muted-foreground">
          Tenants this user belongs to. Manage memberships from the Tenants
          page.
        </p>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      ) : (
        <DataTable
          data={memberships}
          columns={columns}
          getKey={(m) => `${m.tenant_id}-${m.role}`}
          isLoading={false}
          emptyState="No tenant memberships found."
          dense
        />
      )}
    </div>
  );
}

interface LimitsTabProps {
  record?: PersonalTenantRecord | null;
  mode: "create" | "edit";
}

function LimitsTab({ record, mode }: LimitsTabProps) {
  if (mode === "create") {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <p className="text-sm text-muted-foreground">
          Budget and rate limits can be configured after the user is created.
        </p>
        <p className="mt-2 text-xs text-muted-foreground">
          Default limits from the system configuration will be applied.
        </p>
      </div>
    );
  }

  if (!record) {
    return null;
  }

  const budgetUsed = record.budget_used_usd ?? 0;
  const budgetLimit = record.budget_limit_usd ?? 0;
  const warningThreshold = record.warning_threshold ?? 0.8;
  const percentUsed = budgetLimit > 0 ? (budgetUsed / budgetLimit) * 100 : 0;

  return (
    <div className="space-y-6">
      <div>
        <h4 className="text-sm font-medium">Budget overview</h4>
        <p className="text-xs text-muted-foreground">
          Personal tenant budget consumption. Configure budgets from the Tenants
          page.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border p-4">
          <p className="text-xs text-muted-foreground">Budget used</p>
          <p className="text-2xl font-semibold">
            ${budgetUsed.toFixed(2)}
            <span className="text-sm font-normal text-muted-foreground">
              {" "}
              / ${budgetLimit.toFixed(2)}
            </span>
          </p>
        </div>

        <div className="rounded-lg border p-4">
          <p className="text-xs text-muted-foreground">Usage</p>
          <p className="text-2xl font-semibold">
            {percentUsed.toFixed(1)}%
            <span className="text-sm font-normal text-muted-foreground">
              {" "}
              consumed
            </span>
          </p>
        </div>
      </div>

      <div className="rounded-lg border p-4">
        <p className="text-xs text-muted-foreground">Warning threshold</p>
        <p className="text-lg font-medium">
          {(warningThreshold * 100).toFixed(0)}%
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          Alerts trigger when budget consumption exceeds this threshold.
        </p>
      </div>

      <div className="flex items-start gap-2 rounded-lg border bg-muted/50 p-3 text-sm text-muted-foreground">
        <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
        <p>
          To modify budget limits or rate limits, navigate to the Tenants page
          and edit the user's personal tenant directly.
        </p>
      </div>
    </div>
  );
}
