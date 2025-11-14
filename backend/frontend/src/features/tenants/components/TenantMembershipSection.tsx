import type { MembershipRecord, TenantRecord } from "@/api/tenants";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Users, UserRoundPlus, Trash2 } from "lucide-react";

const dateFormatter = new Intl.DateTimeFormat(undefined, {
  year: "numeric",
  month: "short",
  day: "numeric",
});

type TenantMembershipSectionProps = {
  tenants: TenantRecord[];
  selectedTenantId?: string;
  onTenantChange: (tenantId: string) => void;
  memberships: MembershipRecord[];
  isLoading: boolean;
  onInviteClick: () => void;
  onRemoveMember: (membership: MembershipRecord) => void;
  isRemoving: boolean;
};

export function TenantMembershipSection({
  tenants,
  selectedTenantId,
  onTenantChange,
  memberships,
  isLoading,
  onInviteClick,
  onRemoveMember,
  isRemoving,
}: TenantMembershipSectionProps) {
  const selectedTenant = tenants.find((tenant) => tenant.id === selectedTenantId);

  return (
    <Card>
      <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <Users className="h-4 w-4 text-muted-foreground" />
          <div>
            <CardTitle>Memberships</CardTitle>
            <p className="text-sm text-muted-foreground">
              {selectedTenantId
                ? selectedTenant?.name ?? selectedTenantId
                : "Select a tenant to view members"}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Select
            value={selectedTenantId}
            onValueChange={onTenantChange}
            disabled={tenants.length === 0}
          >
            <SelectTrigger className="w-[220px]">
              <SelectValue placeholder="Select tenant" />
            </SelectTrigger>
            <SelectContent>
              {tenants.map((tenant) => (
                <SelectItem key={tenant.id} value={tenant.id}>
                  {tenant.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            size="sm"
            onClick={onInviteClick}
            disabled={!selectedTenantId}
          >
            <UserRoundPlus className="mr-2 h-4 w-4" /> Invite member
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {!selectedTenantId ? (
          <p className="text-sm text-muted-foreground">
            Select a tenant to view memberships.
          </p>
        ) : isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : memberships.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No members yet. Invite a collaborator to get started.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Email</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Added</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {memberships.map((member) => (
                <TableRow key={member.user_id}>
                  <TableCell className="font-medium">{member.email}</TableCell>
                  <TableCell className="capitalize">{member.role}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {dateFormatter.format(new Date(member.created_at))}
                  </TableCell>
                  <TableCell className="text-right">
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                          variant="outline"
                          size="icon"
                          disabled={isRemoving}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Remove member</AlertDialogTitle>
                          <AlertDialogDescription>
                            This will revoke {member.email}'s access to tenant resources.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() => onRemoveMember(member)}
                          >
                            Remove
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
