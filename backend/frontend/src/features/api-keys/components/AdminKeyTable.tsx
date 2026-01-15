import { useCallback } from "react";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ActionsMenu, EmptyState, TableSkeleton } from "@/components/tables";
import type { ApiKeyRecord } from "@/api/tenants";
import { formatScheduleLabel } from "../utils";
import { shortDateFormatter as dateFormatter } from "@/lib/formatters";
import { Eye, Trash2 } from "lucide-react";

type AdminKeyTableProps = {
  allKeys: ApiKeyRecord[];
  filteredKeys: ApiKeyRecord[];
  isLoading: boolean;
  onViewDetails: (key: ApiKeyRecord) => void;
  onRequestRevoke: (key: ApiKeyRecord) => void;
  revokeDisabled?: boolean;
  formatBudgetValue: (key: ApiKeyRecord) => string;
  formatWarningThresholdValue: (key: ApiKeyRecord) => string;
  selectedIds?: Set<string>;
  onSelectionChange?: (ids: Set<string>) => void;
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
  selectedIds,
  onSelectionChange,
}: AdminKeyTableProps) {
  const activeFilteredKeys = filteredKeys.filter((k) => !k.revoked);
  const allSelected =
    activeFilteredKeys.length > 0 &&
    activeFilteredKeys.every((k) => selectedIds?.has(k.id));
  const someSelected = filteredKeys.some((k) => selectedIds?.has(k.id));

  const toggleAll = useCallback(() => {
    if (!onSelectionChange || !selectedIds) return;
    if (allSelected) {
      const next = new Set(selectedIds);
      activeFilteredKeys.forEach((k) => next.delete(k.id));
      onSelectionChange(next);
    } else {
      const next = new Set(selectedIds);
      activeFilteredKeys.forEach((k) => next.add(k.id));
      onSelectionChange(next);
    }
  }, [allSelected, activeFilteredKeys, onSelectionChange, selectedIds]);

  const toggleOne = useCallback(
    (id: string) => {
      if (!onSelectionChange || !selectedIds) return;
      const next = new Set(selectedIds);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      onSelectionChange(next);
    },
    [onSelectionChange, selectedIds]
  );

  const showCheckboxes = Boolean(selectedIds && onSelectionChange);

  if (isLoading) {
    return <TableSkeleton rows={3} />;
  }

  if (!filteredKeys.length) {
    const hasFilters = allKeys.length > 0;
    return (
      <EmptyState
        message={hasFilters ? "No keys match your filters" : "No keys issued yet"}
        description={hasFilters ? "Try adjusting your filter criteria." : undefined}
      />
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {showCheckboxes && (
            <TableHead className="w-12">
              <Checkbox
                checked={allSelected}
                onCheckedChange={toggleAll}
                aria-label="Select all active keys"
                className={someSelected && !allSelected ? "opacity-50" : ""}
              />
            </TableHead>
          )}
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
        {filteredKeys.map((key) => {
          const isSelected = selectedIds?.has(key.id) ?? false;
          const canSelect = !key.revoked;

          return (
          <TableRow key={key.id} className={isSelected ? "bg-muted/50" : undefined}>
            {showCheckboxes && (
              <TableCell>
                <Checkbox
                  checked={isSelected}
                  onCheckedChange={() => toggleOne(key.id)}
                  disabled={!canSelect}
                  aria-label={`Select ${key.name}`}
                />
              </TableCell>
            )}
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
              <ActionsMenu
                variant="outline"
                actions={[
                  {
                    label: "View details",
                    icon: <Eye className="h-4 w-4" />,
                    onClick: () => onViewDetails(key),
                  },
                  {
                    label: "Revoke key",
                    icon: <Trash2 className="h-4 w-4" />,
                    onClick: () => onRequestRevoke(key),
                    disabled: key.revoked || revokeDisabled,
                    destructive: true,
                  },
                ]}
              />
            </TableCell>
          </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
