import type { ReactNode, FormEvent } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export type FormDialogProps = {
  /** Whether the dialog is open */
  open: boolean;
  /** Callback when the dialog open state changes */
  onOpenChange: (open: boolean) => void;
  /** Dialog title */
  title: ReactNode;
  /** Optional description below the title */
  description?: ReactNode;
  /** Form content */
  children: ReactNode;
  /** Content width class (defaults to sm:max-w-lg) */
  maxWidth?: string;
  /** Additional content class */
  contentClassName?: string;
  /** Class name for the body wrapper (defaults to "space-y-4 py-4") */
  bodyClassName?: string;
  /** Whether the form is submitting */
  isSubmitting?: boolean;
  /** Submit button text (defaults to "Save") */
  submitText?: string;
  /** Submitting button text (defaults to "Saving...") */
  submittingText?: string;
  /** Cancel button text (defaults to "Cancel") */
  cancelText?: string;
  /** Whether the submit button is disabled (beyond isSubmitting) */
  submitDisabled?: boolean;
  /** Called when the form is submitted */
  onSubmit: () => void;
  /** Optional extra footer content (appears before action buttons) */
  extraFooter?: ReactNode;
  /** Whether to use a form element (set false for non-form content like tabs) */
  useForm?: boolean;
};

/**
 * FormDialog provides a standard dialog for form-based interactions.
 * It includes Cancel and Submit buttons with loading state support.
 */
export function FormDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  maxWidth = "sm:max-w-lg",
  contentClassName,
  bodyClassName = "space-y-4 py-4",
  isSubmitting = false,
  submitText = "Save",
  submittingText = "Saving...",
  cancelText = "Cancel",
  submitDisabled = false,
  onSubmit,
  extraFooter,
  useForm = true,
}: FormDialogProps) {
  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    onSubmit();
  };

  const header = (
    <DialogHeader>
      <DialogTitle>{title}</DialogTitle>
      {description && <DialogDescription>{description}</DialogDescription>}
    </DialogHeader>
  );

  const body = <div className={bodyClassName}>{children}</div>;

  const footer = (
    <DialogFooter>
      {extraFooter}
      <Button
        type={useForm ? "button" : undefined}
        variant="outline"
        onClick={() => onOpenChange(false)}
        disabled={isSubmitting}
      >
        {cancelText}
      </Button>
      <Button
        type={useForm ? "submit" : undefined}
        onClick={useForm ? undefined : onSubmit}
        disabled={isSubmitting || submitDisabled}
      >
        {isSubmitting ? submittingText : submitText}
      </Button>
    </DialogFooter>
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={cn(maxWidth, contentClassName)}>
        {useForm ? (
          <form onSubmit={handleSubmit}>
            {header}
            {body}
            {footer}
          </form>
        ) : (
          <>
            {header}
            {body}
            {footer}
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
