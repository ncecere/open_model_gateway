import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Eye, MoreHorizontal, RefreshCcw } from "lucide-react";

import { type PersonalTenantRecord, type TenantStatus } from "@/api/tenants";
import { getUserTenants, type UserTenantMembership } from "@/api/users";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { BudgetMeter } from "@/ui/kit/BudgetMeter";
import { Separator } from "@/components/ui/separator";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DataTable, type DataTableColumn } from "@/ui/kit/DataTable";
import {
  usePersonalTenantsQuery,
  usePersonalTenantFilters,
} from "@/features/users/hooks/usePersonalTenants";

export function UsersPage() {
  const personalTenantsQuery = usePersonalTenantsQuery();
  const records: PersonalTenantRecord[] = personalTenantsQuery.data?.personal_tenants ?? [];
  const { searchTerm, setSearchTerm, statusFilter, setStatusFilter, filteredRecords } =
    usePersonalTenantFilters(records);
  const [selectedUser, setSelectedUser] = useState<PersonalTenantRecord | null>(null);

  const userTenantsQuery = useQuery<UserTenantMembership[]>({
    queryKey: ["user-tenants", selectedUser?.user_id],
    queryFn: () => getUserTenants(selectedUser!.user_id),
    enabled: Boolean(selectedUser?.user_id),
  });

  const personalTenantColumns = useMemo(() => {
    return [
      {
        header: "User",
        cell: (record: PersonalTenantRecord) => (
          <div className="flex flex-col">
            <span className="font-medium">{record.user_name}</span>
            <span className="text-xs text-muted-foreground">{record.user_email}</span>
          </div>
        ),
      },
      {
        header: "Status",
        cell: (record: PersonalTenantRecord) => (
          <span className="capitalize">{record.status}</span>
        ),
      },
      {
        header: "Budget",
        cell: (record: PersonalTenantRecord) => (
          <BudgetMeter used={record.budget_used_usd ?? 0} limit={record.budget_limit_usd ?? 0} />
        ),
        cellClassName: "min-w-[220px]",
      },
      {
        header: "Tenants",
        cell: (record: PersonalTenantRecord) => (
          <Badge variant="secondary">{record.membership_count ?? 1} tenants</Badge>
        ),
      },
      {
        header: "Created",
        cell: (record: PersonalTenantRecord) => (
          <span className="text-sm text-muted-foreground">
            {new Date(record.created_at).toLocaleDateString()}
          </span>
        ),
      },
      {
        header: "Tenant ID",
        cell: (record: PersonalTenantRecord) => (
          <span className="font-mono text-xs">{record.tenant_id}</span>
        ),
      },
      {
        header: "Actions",
        headerClassName: "text-right",
        cellClassName: "text-right",
        cell: (record: PersonalTenantRecord) => (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" aria-label="Open user actions">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              <DropdownMenuItem onSelect={() => setSelectedUser(record)}>
                <Eye className="mr-2 h-4 w-4" /> View details
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ),
      },
    ];
  }, []);

  const membershipColumns: DataTableColumn<UserTenantMembership>[] = useMemo(
    () => [
      {
        header: "Tenant",
        cell: (membership) => membership.tenant_name,
      },
      {
        header: "Role",
        cell: (membership) => <span className="capitalize">{membership.role}</span>,
      },
      {
        header: "Status",
        cell: (membership) => <span className="capitalize">{membership.status}</span>,
      },
      {
        header: "Joined",
        cell: (membership) =>
          new Date(membership.joined_at).toLocaleDateString(),
      },
    ],
    [],
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Users</h1>
          <p className="text-sm text-muted-foreground">
            Personal tenants seeded for each user. Use this view to audit
            defaults, usage, and budget consumption.
          </p>
        </div>
        <Button
          variant="outline"
          size="icon"
          onClick={() => personalTenantsQuery.refetch()}
          disabled={personalTenantsQuery.isFetching}
          title="Refresh"
        >
          <RefreshCcw className="h-4 w-4" />
        </Button>
      </div>

      <Separator />

      <Card>
        <CardHeader className="gap-4">
          <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
            <div>
              <CardTitle>Personal tenants</CardTitle>
              <p className="text-sm text-muted-foreground">
                {records.length} users with seeded personal tenants
              </p>
            </div>
            <div className="flex w-full flex-col gap-2 md:max-w-xl md:flex-row">
              <Input
                placeholder="Search by name or email"
                value={searchTerm}
                onChange={(event) => setSearchTerm(event.target.value)}
                className="w-full"
              />
              <Select
                value={statusFilter}
                onValueChange={(value) =>
                  setStatusFilter(value as "all" | TenantStatus)
                }
              >
                <SelectTrigger className="w-full md:w-[200px]">
                  <SelectValue placeholder="Filter status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="suspended">Suspended</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <DataTable
            data={filteredRecords}
            columns={personalTenantColumns}
            getKey={(record) => record.tenant_id}
            isLoading={personalTenantsQuery.isLoading}
            emptyState="No users match the current filters."
          />
        </CardContent>
      </Card>

      <Dialog open={Boolean(selectedUser)} onOpenChange={(open) => !open && setSelectedUser(null)}>
        <DialogContent className="max-w-3xl">
          {selectedUser ? (
            <>
              <DialogHeader>
                <DialogTitle>{selectedUser.user_name || selectedUser.user_email}</DialogTitle>
                <DialogDescription>{selectedUser.user_email}</DialogDescription>
              </DialogHeader>
              <div className="space-y-6 py-2">
                <div className="grid gap-4 md:grid-cols-3">
                  <StatCard
                    label="Status"
                    value={selectedUser.status}
                  />
                  <StatCard
                    label="Personal tenant created"
                    value={new Date(selectedUser.created_at).toLocaleDateString()}
                  />
                  <StatCard
                    label="Tenants"
                    value={(selectedUser.membership_count ?? 1).toString()}
                  />
                </div>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm font-medium text-muted-foreground">
                      Budget overview
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <BudgetMeter
                      used={selectedUser.budget_used_usd ?? 0}
                      limit={selectedUser.budget_limit_usd ?? 0}
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
                <Button variant="outline" onClick={() => setSelectedUser(null)}>
                  Close
                </Button>
              </DialogFooter>
            </>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <Card>
      <CardHeader>
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
