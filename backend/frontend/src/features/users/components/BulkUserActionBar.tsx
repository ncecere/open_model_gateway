import { CheckCircle, Loader2, Trash2, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";

export interface BulkUserActionBarProps {
  selectedCount: number;
  onActivate: () => void;
  onSuspend: () => void;
  onDelete: () => void;
  onClear: () => void;
  isLoading?: boolean;
}

export function BulkUserActionBar({
  selectedCount,
  onActivate,
  onSuspend,
  onDelete,
  onClear,
  isLoading = false,
}: BulkUserActionBarProps) {
  if (selectedCount === 0) {
    return null;
  }

  return (
    <div className="fixed bottom-6 left-1/2 z-50 flex -translate-x-1/2 items-center gap-3 rounded-lg border bg-background px-4 py-3 shadow-lg">
      <span className="text-sm font-medium">
        {selectedCount} user{selectedCount !== 1 ? "s" : ""} selected
      </span>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={onActivate}
          disabled={isLoading}
        >
          {isLoading ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <CheckCircle className="mr-2 h-4 w-4" />
          )}
          Activate
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={onSuspend}
          disabled={isLoading}
        >
          {isLoading ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <XCircle className="mr-2 h-4 w-4" />
          )}
          Suspend
        </Button>
        <Button
          variant="destructive"
          size="sm"
          onClick={onDelete}
          disabled={isLoading}
        >
          <Trash2 className="mr-2 h-4 w-4" />
          Delete
        </Button>
      </div>
      <Button variant="ghost" size="sm" onClick={onClear} disabled={isLoading}>
        Clear
      </Button>
    </div>
  );
}
