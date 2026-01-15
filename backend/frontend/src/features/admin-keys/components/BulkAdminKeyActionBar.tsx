import { Loader2, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";

export interface BulkAdminKeyActionBarProps {
  selectedCount: number;
  onRevoke: () => void;
  onClear: () => void;
  isLoading?: boolean;
}

export function BulkAdminKeyActionBar({
  selectedCount,
  onRevoke,
  onClear,
  isLoading = false,
}: BulkAdminKeyActionBarProps) {
  if (selectedCount === 0) {
    return null;
  }

  return (
    <div className="fixed bottom-6 left-1/2 z-50 flex -translate-x-1/2 items-center gap-3 rounded-lg border bg-background px-4 py-3 shadow-lg">
      <span className="text-sm font-medium">
        {selectedCount} token{selectedCount !== 1 ? "s" : ""} selected
      </span>
      <div className="flex items-center gap-2">
        <Button
          variant="destructive"
          size="sm"
          onClick={onRevoke}
          disabled={isLoading}
        >
          {isLoading ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <XCircle className="mr-2 h-4 w-4" />
          )}
          Revoke Selected
        </Button>
      </div>
      <Button variant="ghost" size="sm" onClick={onClear} disabled={isLoading}>
        Clear
      </Button>
    </div>
  );
}
