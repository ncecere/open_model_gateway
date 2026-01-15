import { XCircle, X } from "lucide-react";
import { Button } from "@/components/ui/button";

export interface BulkUserKeyActionBarProps {
  selectedCount: number;
  onRevoke: () => void;
  onClear: () => void;
  isLoading?: boolean;
  disabled?: boolean;
}

export function BulkUserKeyActionBar({
  selectedCount,
  onRevoke,
  onClear,
  isLoading,
  disabled,
}: BulkUserKeyActionBarProps) {
  if (selectedCount === 0) return null;

  return (
    <div className="fixed bottom-6 left-1/2 z-50 -translate-x-1/2">
      <div className="flex items-center gap-3 rounded-lg border bg-background px-4 py-3 shadow-lg">
        <span className="text-sm font-medium">
          {selectedCount} key{selectedCount !== 1 ? "s" : ""} selected
        </span>
        <Button
          variant="destructive"
          size="sm"
          onClick={onRevoke}
          disabled={isLoading || disabled}
        >
          <XCircle className="mr-2 h-4 w-4" />
          Revoke selected
        </Button>
        <Button variant="ghost" size="sm" onClick={onClear} disabled={isLoading}>
          <X className="mr-2 h-4 w-4" />
          Clear
        </Button>
      </div>
    </div>
  );
}
