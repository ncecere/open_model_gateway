import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Eye, MoreHorizontal, RefreshCcw } from "lucide-react";

import {
  listPersonalTenants,
  type PersonalTenantRecord,
  type TenantStatus,
} from "@/api/tenants";
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
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { BudgetMeter } from "@/ui/kit/BudgetMeter";
import { Separator } from "@/components/ui/separator";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export function UsersPage() {
  const query = useQuery({
    queryKey: ["personal-tenants"],
    queryFn: () => listPersonalTenants({ limit: 500 }),
  });

  const records: PersonalTenantRecord[] = query.data?.personal_tenants ?? [];
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | TenantStatus>("all");
  const [selectedUser, setSelectedUser] = useState<PersonalTenantRecord | null>(null);

  const filteredRecords = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return records.filter((record) => {
      const matchesStatus =
        statusFilter === "all" || record.status === statusFilter;
      if (!matchesStatus) {
        return false;
      }
      if (!term) {
        return true;
      }
      return (
        record.user_email.toLowerCase().includes(term) ||
        record.user_name.toLowerCase().includes(term)
      );
    });
  }, [records, searchTerm, statusFilter]);

  const userTenantsQuery = useQuery<UserTenantMembership[]>({
    queryKey: ["user-tenants", selectedUser?.user_id],
    queryFn: () => getUserTenants(selectedUser!.user_id),
    enabled: Boolean(selectedUser?.user_id),
  });

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
          onClick={() => query.refetch()}
          disabled={query.isFetching}
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
          {query.isLoading ? (
            <Skeleton className="h-48 w-full" />
          ) : filteredRecords.length ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>User</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Budget</TableHead>
                  <TableHead>Tenants</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Tenant ID</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredRecords.map((record) => (
                  <TableRow key={record.tenant_id}>
                    <TableCell>
                      <div className="flex flex-col">
                        <span className="font-medium">{record.user_name}</span>
                        <span className="text-xs text-muted-foreground">
                          {record.user_email}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="capitalize">
                      {record.status}
                    </TableCell>
                    <TableCell className="min-w-[220px]">
                      <BudgetMeter
                        used={record.budget_used_usd ?? 0}
                        limit={record.budget_limit_usd ?? 0}
                      />
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary">
                        {record.membership_count ?? 1} tenants
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(record.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {record.tenant_id}
                    </TableCell>
                    <TableCell className="text-right">
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
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <p className="text-sm text-muted-foreground">
              No users match the current filters.
            </p>
          )}
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
                    {userTenantsQuery.isLoading ? (
                      <Skeleton className="h-32 w-full" />
                    ) : userTenantsQuery.data && userTenantsQuery.data.length ? (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Tenant</TableHead>
                            <TableHead>Role</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Joined</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {userTenantsQuery.data.map((membership) => (
                            <TableRow key={membership.tenant_id + membership.role}>
                              <TableCell>{membership.tenant_name}</TableCell>
                              <TableCell className="capitalize">{membership.role}</TableCell>
                              <TableCell className="capitalize">{membership.status}</TableCell>
                              <TableCell>
                                {new Date(membership.joined_at).toLocaleDateString()}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    ) : (
                      <p className="text-sm text-muted-foreground">
                        No additional tenant memberships found.
                      </p>
                    )}
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
