import { useCallback } from "react";
import {
  Copy,
  Eye,
  Mail,
  MoreHorizontal,
  Pencil,
  Trash2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { BudgetMeter } from "@/ui/kit/BudgetMeter";
import { statusToneClass, toneFromStatus } from "@/ui/kit/status";
import type { PersonalTenantRecord, TenantStatus } from "../types";

export interface UserDirectoryCardProps {
  records: PersonalTenantRecord[];
  filteredRecords: PersonalTenantRecord[];
  searchTerm: string;
  setSearchTerm: (term: string) => void;
  statusFilter: "all" | TenantStatus;
  setStatusFilter: (filter: "all" | TenantStatus) => void;
  isLoading: boolean;
  selectedIds: Set<string>;
  onSelectionChange: (ids: Set<string>) => void;
  onView: (record: PersonalTenantRecord) => void;
  onEdit: (record: PersonalTenantRecord) => void;
  onClone: (record: PersonalTenantRecord) => void;
  onDelete: (record: PersonalTenantRecord) => void;
  onSendInvite: (userId: string) => void;
  invitePending?: string | null;
}

export function UserDirectoryCard({
  records,
  filteredRecords,
  searchTerm,
  setSearchTerm,
  statusFilter,
  setStatusFilter,
  isLoading,
  selectedIds,
  onSelectionChange,
  onView,
  onEdit,
  onClone,
  onDelete,
  onSendInvite,
  invitePending,
}: UserDirectoryCardProps) {
  const allSelected =
    filteredRecords.length > 0 &&
    filteredRecords.every((r) => selectedIds.has(r.tenant_id));
  const someSelected = filteredRecords.some((r) => selectedIds.has(r.tenant_id));

  const toggleAll = useCallback(() => {
    if (allSelected) {
      const next = new Set(selectedIds);
      filteredRecords.forEach((r) => next.delete(r.tenant_id));
      onSelectionChange(next);
    } else {
      const next = new Set(selectedIds);
      filteredRecords.forEach((r) => next.add(r.tenant_id));
      onSelectionChange(next);
    }
  }, [allSelected, filteredRecords, onSelectionChange, selectedIds]);

  const toggleOne = useCallback(
    (id: string) => {
      const next = new Set(selectedIds);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      onSelectionChange(next);
    },
    [onSelectionChange, selectedIds],
  );

  const handleSendInvite = useCallback(
    (event: Event, userId: string) => {
      event.preventDefault();
      onSendInvite(userId);
    },
    [onSendInvite],
  );

  return (
    <Card>
      <CardHeader className="gap-4">
        <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <div>
            <CardTitle>User directory</CardTitle>
            <p className="text-sm text-muted-foreground">
              {records.length} user{records.length !== 1 ? "s" : ""} with personal tenants
            </p>
          </div>
          <div className="flex w-full flex-col gap-2 md:max-w-xl md:flex-row">
            <Input
              placeholder="Search by name or email"
              value={searchTerm}
              onChange={(event) => setSearchTerm(event.target.value)}
              className="w-full"
            />
            <Select
              value={statusFilter}
              onValueChange={(value) =>
                setStatusFilter(value as "all" | TenantStatus)
              }
            >
              <SelectTrigger className="w-full md:w-[200px]">
                <SelectValue placeholder="Filter status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="suspended">Suspended</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        <UserTable
          records={filteredRecords}
          isLoading={isLoading}
          hasAnyRecords={records.length > 0}
          selectedIds={selectedIds}
          allSelected={allSelected}
          someSelected={someSelected}
          toggleAll={toggleAll}
          toggleOne={toggleOne}
          onView={onView}
          onEdit={onEdit}
          onClone={onClone}
          onDelete={onDelete}
          onSendInvite={handleSendInvite}
          invitePending={invitePending}
        />
      </CardContent>
    </Card>
  );
}

interface UserTableProps {
  records: PersonalTenantRecord[];
  isLoading: boolean;
  hasAnyRecords: boolean;
  selectedIds: Set<string>;
  allSelected: boolean;
  someSelected: boolean;
  toggleAll: () => void;
  toggleOne: (id: string) => void;
  onView: (record: PersonalTenantRecord) => void;
  onEdit: (record: PersonalTenantRecord) => void;
  onClone: (record: PersonalTenantRecord) => void;
  onDelete: (record: PersonalTenantRecord) => void;
  onSendInvite: (event: Event, userId: string) => void;
  invitePending?: string | null;
}

function UserTable({
  records,
  isLoading,
  hasAnyRecords,
  selectedIds,
  allSelected,
  someSelected,
  toggleAll,
  toggleOne,
  onView,
  onEdit,
  onClone,
  onDelete,
  onSendInvite,
  invitePending,
}: UserTableProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
    );
  }

  if (!hasAnyRecords) {
    return (
      <p className="text-sm text-muted-foreground">
        No users found. Create a user to get started.
      </p>
    );
  }

  if (records.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No users match the current search or filters.
      </p>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-12">
            <Checkbox
              checked={allSelected}
              onCheckedChange={toggleAll}
              aria-label="Select all"
              className={someSelected && !allSelected ? "opacity-50" : ""}
            />
          </TableHead>
          <TableHead>User</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="min-w-[220px]">Budget</TableHead>
          <TableHead>Tenants</TableHead>
          <TableHead>Created</TableHead>
          <TableHead className="w-12 text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {records.map((record) => {
          const isSelected = selectedIds.has(record.tenant_id);
          const statusTone = toneFromStatus(record.status);

          return (
            <TableRow
              key={record.tenant_id}
              className={isSelected ? "bg-muted/50" : undefined}
            >
              <TableCell>
                <Checkbox
                  checked={isSelected}
                  onCheckedChange={() => toggleOne(record.tenant_id)}
                  aria-label={`Select ${record.user_name || record.user_email}`}
                />
              </TableCell>
              <TableCell>
                <div className="flex flex-col">
                  <span className="font-medium">{record.user_name}</span>
                  <span className="text-xs text-muted-foreground">
                    {record.user_email}
                  </span>
                </div>
              </TableCell>
              <TableCell>
                <Badge className={statusToneClass(statusTone)}>
                  {record.status.charAt(0).toUpperCase() + record.status.slice(1)}
                </Badge>
              </TableCell>
              <TableCell>
                <BudgetMeter
                  used={record.budget_used_usd ?? 0}
                  limit={record.budget_limit_usd ?? 0}
                  warningThreshold={record.warning_threshold ?? 0.8}
                />
              </TableCell>
              <TableCell>
                <Badge variant="secondary">
                  {record.membership_count ?? 1} tenant
                  {(record.membership_count ?? 1) !== 1 ? "s" : ""}
                </Badge>
              </TableCell>
              <TableCell>
                <span className="text-sm text-muted-foreground">
                  {new Date(record.created_at).toLocaleDateString()}
                </span>
              </TableCell>
              <TableCell className="text-right">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label="Open user actions"
                    >
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuLabel>Actions</DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onSelect={() => onView(record)}>
                      <Eye className="mr-2 h-4 w-4" /> View details
                    </DropdownMenuItem>
                    <DropdownMenuItem onSelect={() => onEdit(record)}>
                      <Pencil className="mr-2 h-4 w-4" /> Edit
                    </DropdownMenuItem>
                    <DropdownMenuItem onSelect={() => onClone(record)}>
                      <Copy className="mr-2 h-4 w-4" /> Clone
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      disabled={
                        invitePending !== null && invitePending !== record.user_id
                      }
                      onSelect={(event) => onSendInvite(event, record.user_id)}
                    >
                      <Mail className="mr-2 h-4 w-4" /> Send invite email
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      className="text-destructive focus:text-destructive"
                      onSelect={() => onDelete(record)}
                    >
                      <Trash2 className="mr-2 h-4 w-4" /> Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
