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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { BudgetMeter } from "@/ui/kit/BudgetMeter";
import { MoreHorizontal, Pencil, Trash2 } from "lucide-react";

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
  onDeleteTenant: (tenant: TenantRecord) => void;
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
  onDeleteTenant,
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
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs uppercase text-muted-foreground">
                  <th className="pb-3 font-medium">Name</th>
                  <th className="pb-3 font-medium">Status</th>
                  <th className="pb-3 font-medium min-w-[220px]">Budget</th>
                  <th className="pb-3 font-medium">Created</th>
                  <th className="pb-3 font-medium text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {displayTenants.map((tenant) => (
                  <tr key={tenant.id} className="border-t text-sm">
                    <td className="py-3 font-medium">{tenant.name}</td>
                    <td className="py-3">
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
                    </td>
                    <td className="py-3 pr-4 min-w-[220px]">
                      <BudgetMeter
                        used={tenant.budget_used_usd ?? 0}
                        limit={
                          tenant.budget_limit_usd ?? budgetDefaults?.default_usd ?? 0
                        }
                        warningThreshold={
                          tenant.warning_threshold ??
                          budgetDefaults?.warning_threshold_perc ??
                          0.8
                        }
                      />
                    </td>
                    <td className="py-3 text-sm text-muted-foreground">
                      {dateFormatter.format(new Date(tenant.created_at))}
                    </td>
                    <td className="py-3 text-right">
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
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <DropdownMenuItem
                                onSelect={(event) => event.preventDefault()}
                                className="text-destructive focus:text-destructive"
                              >
                                <Trash2 className="mr-2 h-4 w-4" /> Delete tenant
                              </DropdownMenuItem>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Delete tenant</AlertDialogTitle>
                                <AlertDialogDescription>
                                  This will suspend the tenant and revoke access to its resources. This action can be undone by reactivating the tenant.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction onClick={() => onDeleteTenant(tenant)}>
                                  Delete tenant
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
