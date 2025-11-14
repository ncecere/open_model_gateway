import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Pencil, Trash2, MoreHorizontal } from "lucide-react";
import type { ModelCatalogEntry } from "@/api/model-catalog";

const currency = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 4,
});

const dateTime = new Intl.DateTimeFormat(undefined, {
  year: "numeric",
  month: "short",
  day: "numeric",
  hour: "2-digit",
  minute: "2-digit",
});

type ModelTableProps = {
  models: ModelCatalogEntry[];
  isLoading: boolean;
  hasAnyModels: boolean;
  onEdit: (model: ModelCatalogEntry) => void;
  onDelete: (model: ModelCatalogEntry) => void;
};

export function ModelTable({
  models,
  isLoading,
  hasAnyModels,
  onEdit,
  onDelete,
}: ModelTableProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
    );
  }

  if (!hasAnyModels) {
    return (
      <p className="text-sm text-muted-foreground">
        No models configured yet. Add a catalog entry to expose a provider alias
        to clients.
      </p>
    );
  }

  if (models.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No models match the current search or filters.
      </p>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Alias</TableHead>
          <TableHead>Provider model</TableHead>
          <TableHead>Deployment</TableHead>
          <TableHead>Pricing</TableHead>
          <TableHead>Modalities</TableHead>
          <TableHead>Updated</TableHead>
          <TableHead className="w-12 text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {models.map((model) => (
          <TableRow key={model.alias}>
            <TableCell className="font-medium">
              <div className="flex flex-col">
                <span>{model.alias}</span>
                <span className="text-xs text-muted-foreground">
                  {model.enabled ? "enabled" : "disabled"}
                </span>
              </div>
            </TableCell>
            <TableCell>
              <div className="flex flex-col">
                <span>{model.provider}</span>
                <span className="text-xs text-muted-foreground">
                  {model.provider_model}
                </span>
              </div>
            </TableCell>
            <TableCell>
              <div className="flex flex-col">
                <span>{model.deployment}</span>
                <span className="text-xs text-muted-foreground">
                  {model.region || "n/a"}
                </span>
              </div>
            </TableCell>
            <TableCell className="text-sm">
              <div className="flex flex-col">
                <span>{currency.format(model.price_input)} input</span>
                <span>{currency.format(model.price_output)} output</span>
              </div>
            </TableCell>
            <TableCell>
              <div className="flex flex-wrap gap-1">
                {model.modalities.length > 0 ? (
                  model.modalities.map((modality) => (
                    <Badge key={modality} variant="secondary">
                      {modality}
                    </Badge>
                  ))
                ) : (
                  <span className="text-xs text-muted-foreground">none</span>
                )}
                {model.supports_tools ? (
                  <Badge variant="outline">tools</Badge>
                ) : null}
              </div>
            </TableCell>
            <TableCell className="text-sm text-muted-foreground">
              {dateTime.format(new Date(model.updated_at))}
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
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onSelect={() => onEdit(model)}>
                    <Pencil className="mr-2 h-4 w-4" /> Edit
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className="text-destructive focus:text-destructive"
                    onSelect={() => onDelete(model)}
                  >
                    <Trash2 className="mr-2 h-4 w-4" /> Delete
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
