import type { BudgetDefaults } from "@/api/budgets";
import type { TenantRecord, TenantStatus } from "@/api/tenants";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { BudgetMeter } from "@/ui/kit/BudgetMeter";
import { MoreHorizontal, Pencil, Users } from "lucide-react";

const dateFormatter = new Intl.DateTimeFormat(undefined, {
  year: "numeric",
  month: "short",
  day: "numeric",
});

interface TenantDirectoryCardProps {
  activeCount: number;
  totalCount: number;
  searchValue: string;
  onSearchValueChange: (value: string) => void;
  statusFilter: "all" | TenantStatus;
  onStatusFilterChange: (value: "all" | TenantStatus) => void;
  statusOptions: TenantStatus[];
  isLoading: boolean;
  tenants: TenantRecord[];
  displayTenants: TenantRecord[];
  onStatusChange: (tenantId: string, status: TenantStatus) => Promise<void>;
  isStatusUpdating: boolean;
  onEditTenant: (tenant: TenantRecord) => void;
  onManageMembers: (tenantId: string) => void;
  budgetDefaults?: BudgetDefaults;
}

export function TenantDirectoryCard({
  activeCount,
  totalCount,
  searchValue,
  onSearchValueChange,
  statusFilter,
  onStatusFilterChange,
  statusOptions,
  isLoading,
  tenants,
  displayTenants,
  onStatusChange,
  isStatusUpdating,
  onEditTenant,
  onManageMembers,
  budgetDefaults,
}: TenantDirectoryCardProps) {
  return (
    <Card id="tenant-directory">
      <CardHeader className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <CardTitle>Tenant directory</CardTitle>
          <p className="text-sm text-muted-foreground">
            {activeCount} active · {totalCount} total
          </p>
        </div>
        <div className="flex w-full flex-col gap-2 md:max-w-xl md:flex-row md:items-center">
          <Input
            placeholder="Search tenants"
            value={searchValue}
            onChange={(event) => onSearchValueChange(event.target.value)}
            className="w-full"
            aria-label="Search tenants"
          />
          <div className="flex flex-col gap-1">
            <Label className="sr-only" htmlFor="tenant-status-filter">
              Filter status
            </Label>
            <Select
              value={statusFilter}
              onValueChange={(value) =>
                onStatusFilterChange(value as "all" | TenantStatus)
              }
            >
              <SelectTrigger
                id="tenant-status-filter"
                className="w-full md:w-[180px]"
              >
                <SelectValue placeholder="Filter status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                {statusOptions.map((status) => (
                  <SelectItem key={status} value={status}>
                    {status}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : tenants.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No tenants found. Create your first tenant to begin.
          </p>
        ) : displayTenants.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No tenants match the current filters.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Budget</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {displayTenants.map((tenant) => (
                <TableRow key={tenant.id}>
                  <TableCell className="font-medium">{tenant.name}</TableCell>
                  <TableCell>
                    <Select
                      value={tenant.status}
                      onValueChange={(value) =>
                        onStatusChange(tenant.id, value as TenantStatus)
                      }
                      disabled={isStatusUpdating}
                    >
                      <SelectTrigger className="w-[140px]">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {statusOptions.map((status) => (
                          <SelectItem key={status} value={status}>
                            {status}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </TableCell>
                  <TableCell className="min-w-[220px]">
                    <BudgetMeter
                      used={tenant.budget_used_usd ?? 0}
                      limit={
                        tenant.budget_limit_usd ?? budgetDefaults?.default_usd ?? 0
                      }
                    />
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {dateFormatter.format(new Date(tenant.created_at))}
                  </TableCell>
                  <TableCell className="text-right">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuLabel>Actions</DropdownMenuLabel>
                        <DropdownMenuItem onSelect={() => onEditTenant(tenant)}>
                          <Pencil className="mr-2 h-4 w-4" /> Edit tenant
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onSelect={() => onManageMembers(tenant.id)}
                        >
                          <Users className="mr-2 h-4 w-4" /> Manage members
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
