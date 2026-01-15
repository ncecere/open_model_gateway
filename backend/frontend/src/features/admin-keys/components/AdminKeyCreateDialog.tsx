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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { AdminKeyScope } from "../types";
import type { AdminKeyFormState } from "../types";

export interface AdminKeyCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  form: AdminKeyFormState;
  onChange: (form: AdminKeyFormState) => void;
  onSubmit: () => void;
  loading: boolean;
}

export function AdminKeyCreateDialog({
  open,
  onOpenChange,
  form,
  onChange,
  onSubmit,
  loading,
}: AdminKeyCreateDialogProps) {
  const isValid = form.name.trim().length > 0 && Number(form.expiresDays) > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create admin token</DialogTitle>
          <DialogDescription>
            Tokens expire automatically. Choose a scope and expiry window.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="admin-key-name">Name</Label>
            <Input
              id="admin-key-name"
              value={form.name}
              onChange={(e) => onChange({ ...form, name: e.target.value })}
              placeholder="Billing automation"
            />
          </div>
          <div className="space-y-2">
            <Label>Scope</Label>
            <Select
              value={form.scope}
              onValueChange={(value) =>
                onChange({ ...form, scope: value as AdminKeyScope })
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="Select scope" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="admin">Admin (current user)</SelectItem>
                <SelectItem value="system">System</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Admin scope is tied to your account. System scope has elevated
              access.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="admin-key-expiry">Expires in (days)</Label>
            <Input
              id="admin-key-expiry"
              type="number"
              min="1"
              value={form.expiresDays}
              onChange={(e) =>
                onChange({ ...form, expiresDays: e.target.value })
              }
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={onSubmit} disabled={loading || !isValid}>
            {loading ? "Creating..." : "Create token"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
