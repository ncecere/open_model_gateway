import { type ReactNode } from "react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { RowSelectionState } from "@/hooks/useRowSelection";

export type DataTableColumn<T> = {
  header: ReactNode;
  cell: (item: T) => ReactNode;
  headerClassName?: string;
  cellClassName?: string;
};

export type SelectableDataTableProps<T> = {
  data: T[];
  columns: DataTableColumn<T>[];
  getKey: (item: T, index: number) => string;
  selection: RowSelectionState<T>;
  isLoading?: boolean;
  emptyState?: ReactNode;
  className?: string;
  dense?: boolean;
  /** Whether to show the selection column. */
  showSelection?: boolean;
};

/**
 * An enhanced DataTable with row selection support.
 * Integrates with useRowSelection hook for state management.
 */
export function SelectableDataTable<T>({
  data,
  columns,
  getKey,
  selection,
  isLoading,
  emptyState,
  className,
  dense,
  showSelection = true,
}: SelectableDataTableProps<T>) {
  if (isLoading) {
    return <Skeleton className="h-48 w-full" />;
  }

  if (!data.length) {
    return (
      <div className={cn("flex h-32 items-center justify-center rounded-md border", className)}>
        <p className="text-sm text-muted-foreground">
          {emptyState ?? "No records found."}
        </p>
      </div>
    );
  }

  return (
    <div className={cn("overflow-x-auto", className)}>
      <Table>
        <TableHeader>
          <TableRow>
            {showSelection && (
              <TableHead className="w-12">
                <Checkbox
                  checked={selection.isAllSelected}
                  indeterminate={selection.isIndeterminate}
                  onCheckedChange={() => selection.toggleAll()}
                  aria-label="Select all"
                />
              </TableHead>
            )}
            {columns.map((column, index) => (
              <TableHead key={index} className={cn(column.headerClassName)}>
                {column.header}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((item, index) => {
            const id = getKey(item, index);
            const isSelected = selection.isSelected(id);
            return (
              <TableRow
                key={id}
                className={cn(
                  dense ? "h-10" : undefined,
                  isSelected && "bg-accent/50"
                )}
              >
                {showSelection && (
                  <TableCell className="w-12">
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => selection.toggleRow(id)}
                      aria-label={`Select row ${index + 1}`}
                    />
                  </TableCell>
                )}
                {columns.map((column, columnIndex) => (
                  <TableCell key={columnIndex} className={cn(column.cellClassName)}>
                    {column.cell(item)}
                  </TableCell>
                ))}
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
