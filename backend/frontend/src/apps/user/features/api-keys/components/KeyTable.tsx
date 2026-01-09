import { RefreshCcw, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
}: KeyTableProps) {
  const hasKeys = keys.length > 0;
  const showActions = variant === "active" && (allowRevoke || allowRotate);
  const showBudgetColumns = variant === "active";

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="space-y-2">
            {[...Array(4)].map((_, idx) => (
              <div key={idx} className="h-10 animate-pulse rounded bg-muted" />
            ))}
          </div>
        ) : hasKeys ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Prefix</TableHead>
                {showBudgetColumns ? (
                  <>
                    <TableHead>Budget</TableHead>
                    <TableHead>Reset schedule</TableHead>
                  </>
                ) : null}
                {variant === "active" && showActions ? (
                  <TableHead className="text-right">Actions</TableHead>
                ) : null}
                {variant === "revoked" ? <TableHead>Revoked at</TableHead> : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((key) => {
                const budgetMeta = getBudgetMeta(key);
                const hasBudget =
                  typeof budgetMeta.limit === "number" && budgetMeta.limit > 0;
                return (
                  <TableRow key={key.id}>
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
          <p className="text-sm text-muted-foreground">No data yet.</p>
        )}
      </CardContent>
    </Card>
  );
}
