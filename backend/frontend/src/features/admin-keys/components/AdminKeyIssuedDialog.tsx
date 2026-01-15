import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";

export interface AdminKeyIssuedDialogProps {
  open: boolean;
  token: string | null;
  onClose: () => void;
}

export function AdminKeyIssuedDialog({
  open,
  token,
  onClose,
}: AdminKeyIssuedDialogProps) {
  const { copy, copied } = useCopyToClipboard();

  if (!token) return null;

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Admin token issued</DialogTitle>
          <DialogDescription>
            Copy the token now—this is the only time it will be shown.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2 py-2">
          <Label>Token</Label>
          <div className="flex items-center gap-2">
            <Input value={token} readOnly className="font-mono text-sm" />
            <Button
              variant="outline"
              size="icon"
              onClick={() => copy(token)}
            >
              {copied ? (
                <Check className="h-4 w-4 text-green-500" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
        <DialogFooter>
          <Button onClick={onClose}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
