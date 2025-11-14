import type { MembershipRole, TenantRecord } from "@/api/tenants";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";

import type { MembershipDialogState } from "../hooks/useTenantDialogs";
import { MembershipSuggestions } from "./MembershipSuggestions";

type TenantMembershipDialogProps = {
  dialog: MembershipDialogState;
  isSubmitting: boolean;
  usersLoading: boolean;
  tenants: TenantRecord[];
  selectedTenantId?: string;
  onSubmit: () => void;
};

export function TenantMembershipDialog({
  dialog,
  isSubmitting,
  usersLoading,
  tenants,
  selectedTenantId,
  onSubmit,
}: TenantMembershipDialogProps) {
  const selectedTenant = tenants.find((tenant) => tenant.id === selectedTenantId);
  const roleOptions: MembershipRole[] = ["owner", "admin", "viewer", "user"];

  return (
    <Dialog open={dialog.open} onOpenChange={dialog.setOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Invite member</DialogTitle>
          <DialogDescription>
            Assign a role to grant access to {selectedTenant?.name ?? "this tenant"}.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="member-email">Email</Label>
            <Input
              id="member-email"
              value={dialog.email}
              onChange={(event) => dialog.setEmail(event.target.value)}
              placeholder="user@example.com"
              autoFocus
            />
            <MembershipSuggestions
              query={dialog.email}
              loading={usersLoading}
              suggestions={dialog.suggestions}
              onSelect={(email) => dialog.setEmail(email)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="member-role">Role</Label>
            <Select
              value={dialog.role}
              onValueChange={(value) => dialog.setRole(value as MembershipRole)}
            >
              <SelectTrigger id="member-role">
                <SelectValue placeholder="Select a role" />
              </SelectTrigger>
              <SelectContent>
                {roleOptions.map((role) => (
                  <SelectItem key={role} value={role} className="capitalize">
                    {role}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => dialog.setOpen(false)}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button onClick={onSubmit} disabled={isSubmitting}>
            {isSubmitting ? "Saving…" : "Add member"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
