import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal } from "lucide-react";
import { cn } from "@/lib/utils";

export type ActionItem = {
  /** Label for the action */
  label: string;
  /** Icon to display before the label */
  icon?: ReactNode;
  /** Called when the action is clicked */
  onClick: () => void;
  /** Whether the action is disabled */
  disabled?: boolean;
  /** Whether this is a destructive action (red text) */
  destructive?: boolean;
};

export type ActionsMenuProps = {
  /** Array of action items */
  actions: ActionItem[];
  /** Optional label at the top of the menu */
  label?: string;
  /** Whether to show a separator after the label */
  showLabelSeparator?: boolean;
  /** Alignment of the dropdown content */
  align?: "start" | "center" | "end";
  /** Button variant */
  variant?: "ghost" | "outline";
  /** Additional class name for the trigger button */
  className?: string;
};

/**
 * ActionsMenu provides a dropdown menu for row-level actions in tables.
 */
export function ActionsMenu({
  actions,
  label,
  showLabelSeparator = true,
  align = "end",
  variant = "ghost",
  className,
}: ActionsMenuProps) {
  if (!actions.length) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant={variant} size="icon" className={className}>
          <MoreHorizontal className="h-4 w-4" />
          <span className="sr-only">Open actions</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align}>
        {label && (
          <>
            <DropdownMenuLabel>{label}</DropdownMenuLabel>
            {showLabelSeparator && <DropdownMenuSeparator />}
          </>
        )}
        {actions.map((action, index) => (
          <DropdownMenuItem
            key={index}
            disabled={action.disabled}
            className={cn(
              action.destructive && "text-destructive focus:text-destructive"
            )}
            onClick={action.onClick}
          >
            {action.icon && <span className="mr-2">{action.icon}</span>}
            {action.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
