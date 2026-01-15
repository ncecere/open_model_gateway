import { useState, useMemo, useCallback } from "react";
import { RefreshCcw, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
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
import type { UserAPIKey } from "@/api/user/api-keys";

export type BudgetMeta = {
  limit: number | null;
  used: number;
  warning: number;
  schedule: string;
};

export type KeyTableProps = {
  title: string;
  loading: boolean;
  keys: UserAPIKey[];
  variant: "active" | "revoked";
  allowRevoke?: boolean;
  allowRotate?: boolean;
  onRevoke?: (id: string) => void;
  onRotate?: (key: UserAPIKey) => void;
  getBudgetMeta: (key: UserAPIKey) => BudgetMeta;
  formatResetValue: (key: UserAPIKey) => string;
  selectedIds?: Set<string>;
  onSelectionChange?: (ids: Set<string>) => void;
};

export function KeyTable({
  title,
  loading,
  keys,
  variant,
  allowRevoke = false,
  allowRotate = false,
  onRevoke,
  onRotate,
  getBudgetMeta,
  formatResetValue,
  selectedIds,
  onSelectionChange,
}: KeyTableProps) {
  const [searchTerm, setSearchTerm] = useState("");

  const filteredKeys = useMemo(() => {
    if (!searchTerm.trim()) return keys;
    const term = searchTerm.toLowerCase();
    return keys.filter(
      (key) =>
        key.name.toLowerCase().includes(term) ||
        key.prefix.toLowerCase().includes(term)
    );
  }, [keys, searchTerm]);

  const hasKeys = keys.length > 0;
  const hasFilteredKeys = filteredKeys.length > 0;
  const showActions = variant === "active" && (allowRevoke || allowRotate);
  const showBudgetColumns = variant === "active";
  const showCheckboxes = Boolean(selectedIds && onSelectionChange) && variant === "active";

  const allSelected =
    filteredKeys.length > 0 &&
    filteredKeys.every((k) => selectedIds?.has(k.id));
  const someSelected = filteredKeys.some((k) => selectedIds?.has(k.id));

  const toggleAll = useCallback(() => {
    if (!onSelectionChange || !selectedIds) return;
    if (allSelected) {
      const next = new Set(selectedIds);
      filteredKeys.forEach((k) => next.delete(k.id));
      onSelectionChange(next);
    } else {
      const next = new Set(selectedIds);
      filteredKeys.forEach((k) => next.add(k.id));
      onSelectionChange(next);
    }
  }, [allSelected, filteredKeys, onSelectionChange, selectedIds]);

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

  const formatLastUsed = (dateStr?: string | null) => {
    if (!dateStr) return "Never";
    return new Date(dateStr).toLocaleDateString();
  };

  return (
    <Card>
      <CardHeader className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="space-y-1">
          <CardTitle>{title}</CardTitle>
          {hasKeys && (
            <p className="text-sm text-muted-foreground">
              {filteredKeys.length === keys.length
                ? `${keys.length} key${keys.length !== 1 ? "s" : ""}`
                : `${filteredKeys.length} of ${keys.length} keys`}
            </p>
          )}
        </div>
        {hasKeys && (
          <Input
            placeholder="Search by name or prefix..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full md:max-w-xs"
          />
        )}
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="space-y-2">
            {[...Array(4)].map((_, idx) => (
              <div key={idx} className="h-10 animate-pulse rounded bg-muted" />
            ))}
          </div>
        ) : hasKeys ? (
          hasFilteredKeys ? (
          <Table>
            <TableHeader>
              <TableRow>
                {showCheckboxes && (
                  <TableHead className="w-12">
                    <Checkbox
                      checked={allSelected}
                      onCheckedChange={toggleAll}
                      aria-label="Select all keys"
                      className={someSelected && !allSelected ? "opacity-50" : ""}
                    />
                  </TableHead>
                )}
                <TableHead>Name</TableHead>
                <TableHead>Prefix</TableHead>
                {showBudgetColumns ? (
                  <>
                    <TableHead>Budget</TableHead>
                    <TableHead>Reset schedule</TableHead>
                  </>
                ) : null}
                <TableHead>Last used</TableHead>
                {variant === "active" && showActions ? (
                  <TableHead className="text-right">Actions</TableHead>
                ) : null}
                {variant === "revoked" ? <TableHead>Revoked at</TableHead> : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredKeys.map((key) => {
                const budgetMeta = getBudgetMeta(key);
                const hasBudget =
                  typeof budgetMeta.limit === "number" && budgetMeta.limit > 0;
                const isSelected = selectedIds?.has(key.id) ?? false;
                return (
                  <TableRow key={key.id} className={isSelected ? "bg-muted/50" : undefined}>
                    {showCheckboxes && (
                      <TableCell>
                        <Checkbox
                          checked={isSelected}
                          onCheckedChange={() => toggleOne(key.id)}
                          aria-label={`Select ${key.name}`}
                        />
                      </TableCell>
                    )}
                    <TableCell>{key.name}</TableCell>
                    <TableCell>{key.prefix}</TableCell>
                    {showBudgetColumns ? (
                      <>
                        <TableCell className="min-w-[220px] align-top">
                          {hasBudget ? (
                            <div className="space-y-1.5">
                              <BudgetMeter
                                used={budgetMeta.used}
                                limit={budgetMeta.limit ?? 0}
                                warningThreshold={budgetMeta.warning}
                              />
                              <p className="text-xs text-muted-foreground">
                                Warn at {Math.round(budgetMeta.warning * 100)}%
                              </p>
                            </div>
                          ) : (
                            <p className="text-sm text-muted-foreground">
                              Inherits tenant defaults
                            </p>
                          )}
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {formatResetValue(key)}
                        </TableCell>
                      </>
                    ) : null}
                    <TableCell className="text-sm text-muted-foreground">
                      {formatLastUsed(key.last_used_at)}
                    </TableCell>
                    {variant === "active" && showActions ? (
                      <TableCell className="flex justify-end gap-2">
                        {allowRotate && onRotate ? (
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => onRotate(key)}
                            title="Rotate API key"
                          >
                            <RefreshCcw className="size-4" />
                          </Button>
                        ) : null}
                        {allowRevoke && onRevoke ? (
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="text-destructive"
                              >
                                <Trash2 className="size-4" />
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Revoke API key</AlertDialogTitle>
                                <AlertDialogDescription>
                                  This action cannot be undone. Requests using this
                                  key will immediately fail.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction onClick={() => onRevoke(key.id)}>
                                  Revoke
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        ) : null}
                      </TableCell>
                    ) : null}
                    {variant === "revoked" ? (
                      <TableCell>
                        {key.revoked_at
                          ? new Date(key.revoked_at).toLocaleDateString()
                          : "—"}
                      </TableCell>
                    ) : null}
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
          ) : (
            <p className="text-sm text-muted-foreground">
              No keys match the current search.
            </p>
          )
        ) : (
          <p className="text-sm text-muted-foreground">No data yet.</p>
        )}
      </CardContent>
    </Card>
  );
}
