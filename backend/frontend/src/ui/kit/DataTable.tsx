import { type ReactNode } from "react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export type DataTableColumn<T> = {
  header: ReactNode;
  cell: (item: T) => ReactNode;
  headerClassName?: string;
  cellClassName?: string;
};

export type DataTableProps<T> = {
  data: T[];
  columns: DataTableColumn<T>[];
  getKey: (item: T, index: number) => React.Key;
  isLoading?: boolean;
  emptyState?: ReactNode;
  className?: string;
  dense?: boolean;
};

export function DataTable<T>({
  data,
  columns,
  getKey,
  isLoading,
  emptyState,
  className,
  dense,
}: DataTableProps<T>) {
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
            {columns.map((column, index) => (
              <TableHead key={index} className={cn(column.headerClassName)}>
                {column.header}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((item, index) => (
            <TableRow key={getKey(item, index)} className={dense ? "h-10" : undefined}>
              {columns.map((column, columnIndex) => (
                <TableCell key={columnIndex} className={cn(column.cellClassName)}>
                  {column.cell(item)}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
