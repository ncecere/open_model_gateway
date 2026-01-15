import { Power, PowerOff, Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";

interface BulkTenantActionBarProps {
  selectedCount: number;
  onActivate: () => void;
  onSuspend: () => void;
  onDelete: () => void;
  onClear: () => void;
  isLoading?: boolean;
}

export function BulkTenantActionBar({
  selectedCount,
  onActivate,
  onSuspend,
  onDelete,
  onClear,
  isLoading = false,
}: BulkTenantActionBarProps) {
  if (selectedCount === 0) {
    return null;
  }

  return (
    <div className="flex items-center justify-between rounded-lg border bg-muted/50 px-4 py-3">
      <div className="flex items-center gap-4">
        <span className="text-sm font-medium">
          {selectedCount} tenant{selectedCount === 1 ? "" : "s"} selected
        </span>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={onActivate}
            disabled={isLoading}
          >
            <Power className="mr-1.5 h-4 w-4" />
            Activate
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={onSuspend}
            disabled={isLoading}
          >
            <PowerOff className="mr-1.5 h-4 w-4" />
            Suspend
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={onDelete}
            disabled={isLoading}
            className="text-destructive hover:text-destructive"
          >
            <Trash2 className="mr-1.5 h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>
      <Button variant="ghost" size="sm" onClick={onClear} disabled={isLoading}>
        <X className="mr-1.5 h-4 w-4" />
        Clear
      </Button>
    </div>
  );
}
