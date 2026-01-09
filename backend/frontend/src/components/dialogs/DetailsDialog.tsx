import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { BaseDialog } from "./BaseDialog";

export type DetailsDialogProps = {
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
  /** Content width class (defaults to sm:max-w-[640px]) */
  maxWidth?: string;
  /** Additional content class */
  contentClassName?: string;
  /** Close button text (defaults to "Close") */
  closeText?: string;
  /** Optional extra footer content (appears before Close button) */
  extraFooter?: ReactNode;
};

/**
 * DetailsDialog is a read-only dialog for displaying information.
 * It includes a standard Close button in the footer.
 */
export function DetailsDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  maxWidth = "sm:max-w-[640px]",
  contentClassName,
  closeText = "Close",
  extraFooter,
}: DetailsDialogProps) {
  return (
    <BaseDialog
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
      maxWidth={maxWidth}
      contentClassName={contentClassName}
      footer={
        <>
          {extraFooter}
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {closeText}
          </Button>
        </>
      }
    >
      {children}
    </BaseDialog>
  );
}
