import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { ApiKeyRecord } from "@/api/tenants";
import { formatScheduleLabel } from "../utils";
import { Eye, MoreHorizontal, Trash2 } from "lucide-react";

const dateFormatter = new Intl.DateTimeFormat(undefined, {
  year: "numeric",
  month: "short",
  day: "numeric",
});

type AdminKeyTableProps = {
  allKeys: ApiKeyRecord[];
  filteredKeys: ApiKeyRecord[];
  isLoading: boolean;
  onViewDetails: (key: ApiKeyRecord) => void;
  onRequestRevoke: (key: ApiKeyRecord) => void;
  revokeDisabled?: boolean;
  formatBudgetValue: (key: ApiKeyRecord) => string;
  formatWarningThresholdValue: (key: ApiKeyRecord) => string;
};

export function AdminKeyTable({
  allKeys,
  filteredKeys,
  isLoading,
  onViewDetails,
  onRequestRevoke,
  revokeDisabled,
  formatBudgetValue,
  formatWarningThresholdValue,
}: AdminKeyTableProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
    );
  }

  if (!filteredKeys.length) {
    const message = allKeys.length
      ? "No keys match your filters."
      : "No keys issued yet.";
    return (
      <div className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
        {message}
      </div>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Issuer</TableHead>
          <TableHead>Budget</TableHead>
          <TableHead>Created</TableHead>
          <TableHead>Last used</TableHead>
          <TableHead>Reset schedule</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {filteredKeys.map((key) => (
          <TableRow key={key.id}>
            <TableCell className="font-medium">{key.name}</TableCell>
            <TableCell>
              <Badge variant={key.revoked ? "destructive" : "secondary"}>
                {key.revoked ? "revoked" : "active"}
              </Badge>
            </TableCell>
            <TableCell>
              <div className="flex flex-col">
                <span className="font-medium">
                  {key.issuer?.label ?? key.tenant_name ?? "—"}
                </span>
                <span className="text-xs uppercase text-muted-foreground">
                  {key.issuer?.type ?? "tenant"}
                </span>
              </div>
            </TableCell>
            <TableCell className="text-sm">
              <div className="flex flex-col">
                <span>{formatBudgetValue(key)}</span>
                <span className="text-xs text-muted-foreground">
                  Warn at {formatWarningThresholdValue(key)}
                </span>
              </div>
            </TableCell>
            <TableCell className="text-sm text-muted-foreground">
              {dateFormatter.format(new Date(key.created_at))}
            </TableCell>
            <TableCell className="text-sm text-muted-foreground">
              {key.last_used_at
                ? dateFormatter.format(new Date(key.last_used_at))
                : "—"}
            </TableCell>
            <TableCell className="text-sm text-muted-foreground">
              {formatScheduleLabel(key.budget_refresh_schedule)}
            </TableCell>
            <TableCell className="text-right">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="icon">
                    <MoreHorizontal className="h-4 w-4" />
                    <span className="sr-only">Open actions</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => onViewDetails(key)}>
                    <Eye className="mr-2 h-4 w-4" />
                    View details
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    disabled={key.revoked || revokeDisabled}
                    className="text-destructive focus:text-destructive"
                    onClick={() => onRequestRevoke(key)}
                  >
                    <Trash2 className="mr-2 h-4 w-4" />
                    Revoke key
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
