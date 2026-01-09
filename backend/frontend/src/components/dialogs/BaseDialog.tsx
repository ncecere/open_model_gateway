import type { ReactNode } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

export type BaseDialogProps = {
  /** Whether the dialog is open */
  open: boolean;
  /** Callback when the dialog open state changes */
  onOpenChange: (open: boolean) => void;
  /** Dialog title */
  title: ReactNode;
  /** Optional description below the title */
  description?: ReactNode;
  /** Dialog content */
  children: ReactNode;
  /** Footer content (buttons) */
  footer?: ReactNode;
  /** Content width class (defaults to sm:max-w-lg) */
  maxWidth?: string;
  /** Additional content class */
  contentClassName?: string;
};

/**
 * BaseDialog provides a standard dialog structure with header, content, and footer.
 * Use this as a foundation for more specific dialog types.
 */
export function BaseDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
  maxWidth = "sm:max-w-lg",
  contentClassName,
}: BaseDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={cn(maxWidth, contentClassName)}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && <DialogDescription>{description}</DialogDescription>}
        </DialogHeader>
        <div className="py-2">{children}</div>
        {footer && <DialogFooter>{footer}</DialogFooter>}
      </DialogContent>
    </Dialog>
  );
}
