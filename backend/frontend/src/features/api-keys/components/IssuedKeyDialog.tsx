import { DetailsDialog } from "@/components/dialogs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Copy } from "lucide-react";

export type IssuedSecretPayload = {
  token: string;
  secret: string;
};

type IssuedKeyDialogProps = {
  issuedKey: IssuedSecretPayload | null;
  onCopy: (value: string, label: string) => void;
  onClose: () => void;
};

export function IssuedKeyDialog({ issuedKey, onCopy, onClose }: IssuedKeyDialogProps) {
  if (!issuedKey) {
    return null;
  }

  return (
    <DetailsDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title="API key issued"
      description="Copy the secret now—this is the only time it will be shown."
      closeText="Done"
    >
      <div className="space-y-4">
        <div className="space-y-2">
          <Label>Key token</Label>
          <div className="flex items-center gap-2">
            <Input value={issuedKey.token} readOnly className="font-mono" />
            <Button
              variant="outline"
              size="icon"
              onClick={() => onCopy(issuedKey.token, "Token")}
            >
              <Copy className="h-4 w-4" />
            </Button>
          </div>
        </div>
        <div className="space-y-2">
          <Label>Secret</Label>
          <div className="flex items-center gap-2">
            <Input value={issuedKey.secret} readOnly className="font-mono" />
            <Button
              variant="outline"
              size="icon"
              onClick={() => onCopy(issuedKey.secret, "Secret")}
            >
              <Copy className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
    </DetailsDialog>
  );
}
