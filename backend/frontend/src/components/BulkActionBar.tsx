import { X, type LucideIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export interface BulkAction {
  /** Unique identifier for the action. */
  id: string;
  /** Display label for the action. */
  label: string;
  /** Icon to display. */
  icon?: LucideIcon;
  /** Variant for the button. */
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost";
  /** Whether the action is currently loading. */
  loading?: boolean;
  /** Whether the action is disabled. */
  disabled?: boolean;
  /** Handler called when action is triggered. */
  onClick: () => void;
}

interface BulkActionBarProps {
  /** Number of selected items. */
  selectedCount: number;
  /** Available bulk actions. */
  actions: BulkAction[];
  /** Callback to clear selection. */
  onClearSelection: () => void;
  /** Optional label to describe what is selected. */
  itemLabel?: string;
  /** Additional class names. */
  className?: string;
}

/**
 * A floating action bar that appears when items are selected.
 * Shows selection count and available bulk actions.
 */
export function BulkActionBar({
  selectedCount,
  actions,
  onClearSelection,
  itemLabel = "item",
  className,
}: BulkActionBarProps) {
  if (selectedCount === 0) {
    return null;
  }

  const pluralLabel = selectedCount === 1 ? itemLabel : `${itemLabel}s`;

  return (
    <div
      className={cn(
        "fixed bottom-6 left-1/2 z-50 -translate-x-1/2 transform",
        "flex items-center gap-3 rounded-lg border bg-card px-4 py-3 shadow-lg",
        "animate-in fade-in slide-in-from-bottom-4 duration-200",
        className
      )}
    >
      {/* Selection count */}
      <div className="flex items-center gap-2">
        <button
          onClick={onClearSelection}
          className="flex h-6 w-6 items-center justify-center rounded-full bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground transition-colors"
          aria-label="Clear selection"
        >
          <X className="h-3.5 w-3.5" />
        </button>
        <span className="text-sm font-medium">
          {selectedCount} {pluralLabel} selected
        </span>
      </div>

      {/* Divider */}
      <div className="h-6 w-px bg-border" />

      {/* Actions */}
      <div className="flex items-center gap-2">
        {actions.map((action) => {
          const Icon = action.icon;
          return (
            <Button
              key={action.id}
              variant={action.variant ?? "secondary"}
              size="sm"
              onClick={action.onClick}
              disabled={action.disabled}
              loading={action.loading}
              className="gap-1.5"
            >
              {Icon && <Icon className="h-3.5 w-3.5" />}
              {action.label}
            </Button>
          );
        })}
      </div>
    </div>
  );
}

/**
 * A simpler inline version of the bulk action bar.
 */
export function InlineBulkActions({
  selectedCount,
  actions,
  onClearSelection,
  itemLabel = "item",
  className,
}: BulkActionBarProps) {
  if (selectedCount === 0) {
    return null;
  }

  const pluralLabel = selectedCount === 1 ? itemLabel : `${itemLabel}s`;

  return (
    <div
      className={cn(
        "flex items-center gap-3 rounded-md border bg-accent/50 px-3 py-2",
        className
      )}
    >
      <span className="text-sm font-medium">
        {selectedCount} {pluralLabel} selected
      </span>

      <div className="h-4 w-px bg-border" />

      <div className="flex items-center gap-2">
        {actions.map((action) => {
          const Icon = action.icon;
          return (
            <Button
              key={action.id}
              variant={action.variant ?? "ghost"}
              size="sm"
              onClick={action.onClick}
              disabled={action.disabled}
              loading={action.loading}
              className="h-7 gap-1.5 text-xs"
            >
              {Icon && <Icon className="h-3 w-3" />}
              {action.label}
            </Button>
          );
        })}
      </div>

      <button
        onClick={onClearSelection}
        className="ml-auto text-xs text-muted-foreground hover:text-foreground transition-colors"
      >
        Clear
      </button>
    </div>
  );
}
