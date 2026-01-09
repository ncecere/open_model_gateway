import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { BudgetMeter } from "@/ui/kit/BudgetMeter";
import { MoreHorizontal, Eye, Settings2 } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { TableSkeleton } from "@/components/tables";

export type TenantMembership = {
  tenant_id: string;
  name: string;
  status: string;
  role: string;
  budget_used_usd?: number | null;
  budget_limit_usd?: number | null;
  warning_threshold?: number | null;
  is_personal?: boolean;
};

export type TenantsListProps = {
  tenants: TenantMembership[];
  isLoading: boolean;
  onManage: (tenantId: string) => void;
};

export function TenantsList({ tenants, isLoading, onManage }: TenantsListProps) {
  return (
    <Card>
      <CardHeader className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <CardTitle>Tenant directory</CardTitle>
          <p className="text-xs text-muted-foreground">
            {tenants.length} membership{tenants.length === 1 ? "" : "s"}
          </p>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <TableSkeleton rows={3} />
        ) : tenants.length ? (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs uppercase text-muted-foreground">
                  <th className="pb-3 font-medium">Name</th>
                  <th className="pb-3 font-medium">Status</th>
                  <th className="pb-3 font-medium min-w-[220px]">Budget</th>
                  <th className="pb-3 font-medium min-w-[120px] pl-4">Role</th>
                  <th className="pb-3 font-medium text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {tenants.map((tenant) => (
                  <tr key={tenant.tenant_id} className="border-t text-sm">
                    <td className="py-3 font-medium">{tenant.name}</td>
                    <td className="py-3">
                      <Badge variant={tenant.status === "active" ? "secondary" : "outline"}>
                        {tenant.status}
                      </Badge>
                    </td>
                    <td className="py-3 pr-4 min-w-[220px]">
                      <BudgetMeter
                        used={tenant.budget_used_usd ?? 0}
                        limit={tenant.budget_limit_usd ?? 0}
                        warningThreshold={tenant.warning_threshold ?? 0.8}
                      />
                    </td>
                    <td className="py-3 min-w-[120px] pl-4 capitalize">{tenant.role}</td>
                    <td className="py-3 text-right">
                      <TenantRowActions
                        role={tenant.role}
                        disabled={tenant.is_personal}
                        onManage={() => onManage(tenant.tenant_id)}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            You are not part of any shared tenants.
          </p>
        )}
      </CardContent>
    </Card>
  );
}

type TenantRowActionsProps = {
  role: string;
  onManage: () => void;
  disabled?: boolean;
};

function TenantRowActions({ role, onManage, disabled }: TenantRowActionsProps) {
  const canEdit = !disabled && (role === "owner" || role === "admin");
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" disabled={disabled}>
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={onManage} disabled={disabled}>
          <Eye className="mr-2 h-4 w-4" /> Manage
        </DropdownMenuItem>
        {canEdit ? (
          <DropdownMenuItem disabled>
            <Settings2 className="mr-2 h-4 w-4" /> Admin settings
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
