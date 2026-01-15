import { useCallback } from "react";
import { Eye, MoreHorizontal, XCircle } from "lucide-react";
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
import { statusToneClass, toneFromStatus } from "@/ui/kit/status";
import type {
  AdminKeyRecord,
  AdminKeyScopeFilter,
  AdminKeyStatusFilter,
} from "../types";
import { getKeyStatus, isKeyActive } from "../types";

export interface AdminKeyDirectoryCardProps {
  keys: AdminKeyRecord[];
  filteredKeys: AdminKeyRecord[];
  searchTerm: string;
  setSearchTerm: (term: string) => void;
  scopeFilter: AdminKeyScopeFilter;
  setScopeFilter: (filter: AdminKeyScopeFilter) => void;
  statusFilter: AdminKeyStatusFilter;
  setStatusFilter: (filter: AdminKeyStatusFilter) => void;
  isLoading: boolean;
  selectedIds: Set<string>;
  onSelectionChange: (ids: Set<string>) => void;
  onViewDetails: (record: AdminKeyRecord) => void;
  onRevoke: (record: AdminKeyRecord) => void;
  onCreate: () => void;
}

export function AdminKeyDirectoryCard({
  keys,
  filteredKeys,
  searchTerm,
  setSearchTerm,
  scopeFilter,
  setScopeFilter,
  statusFilter,
  setStatusFilter,
  isLoading,
  selectedIds,
  onSelectionChange,
  onViewDetails,
  onRevoke,
  onCreate,
}: AdminKeyDirectoryCardProps) {
  const allSelected =
    filteredKeys.length > 0 &&
    filteredKeys.filter(isKeyActive).every((k) => selectedIds.has(k.id));
  const someSelected = filteredKeys.some((k) => selectedIds.has(k.id));
  const activeFilteredKeys = filteredKeys.filter(isKeyActive);

  const toggleAll = useCallback(() => {
    if (allSelected) {
      const next = new Set(selectedIds);
      activeFilteredKeys.forEach((k) => next.delete(k.id));
      onSelectionChange(next);
    } else {
      const next = new Set(selectedIds);
      activeFilteredKeys.forEach((k) => next.add(k.id));
      onSelectionChange(next);
    }
  }, [allSelected, activeFilteredKeys, onSelectionChange, selectedIds]);

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

  const formatDate = (dateStr?: string | null) => {
    if (!dateStr) return "—";
    return new Date(dateStr).toLocaleDateString();
  };

  return (
    <Card>
      <CardHeader className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="space-y-1">
          <CardTitle>Admin access tokens</CardTitle>
          <p className="text-sm text-muted-foreground">
            {keys.length} token{keys.length !== 1 ? "s" : ""} total
            {filteredKeys.length !== keys.length &&
              ` · ${filteredKeys.length} matching filters`}
          </p>
        </div>
        <Button onClick={onCreate}>Create token</Button>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Filters */}
        <div className="flex flex-col gap-2 md:flex-row">
          <Input
            placeholder="Search by name or prefix..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full md:max-w-xs"
          />
          <Select
            value={scopeFilter}
            onValueChange={(v) => setScopeFilter(v as AdminKeyScopeFilter)}
          >
            <SelectTrigger className="w-full md:w-[150px]">
              <SelectValue placeholder="Scope" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All scopes</SelectItem>
              <SelectItem value="admin">Admin</SelectItem>
              <SelectItem value="system">System</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={statusFilter}
            onValueChange={(v) => setStatusFilter(v as AdminKeyStatusFilter)}
          >
            <SelectTrigger className="w-full md:w-[150px]">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="expired">Expired</SelectItem>
              <SelectItem value="revoked">Revoked</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Table */}
        <AdminKeyTable
          keys={filteredKeys}
          isLoading={isLoading}
          hasAnyKeys={keys.length > 0}
          selectedIds={selectedIds}
          allSelected={allSelected}
          someSelected={someSelected}
          toggleAll={toggleAll}
          toggleOne={toggleOne}
          onViewDetails={onViewDetails}
          onRevoke={onRevoke}
          formatDate={formatDate}
        />
      </CardContent>
    </Card>
  );
}

interface AdminKeyTableProps {
  keys: AdminKeyRecord[];
  isLoading: boolean;
  hasAnyKeys: boolean;
  selectedIds: Set<string>;
  allSelected: boolean;
  someSelected: boolean;
  toggleAll: () => void;
  toggleOne: (id: string) => void;
  onViewDetails: (record: AdminKeyRecord) => void;
  onRevoke: (record: AdminKeyRecord) => void;
  formatDate: (dateStr?: string | null) => string;
}

function AdminKeyTable({
  keys,
  isLoading,
  hasAnyKeys,
  selectedIds,
  allSelected,
  someSelected,
  toggleAll,
  toggleOne,
  onViewDetails,
  onRevoke,
  formatDate,
}: AdminKeyTableProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
    );
  }

  if (!hasAnyKeys) {
    return (
      <p className="text-sm text-muted-foreground">
        No admin tokens created yet.
      </p>
    );
  }

  if (keys.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No tokens match the current search or filters.
      </p>
    );
  }

  return (
    <div className="rounded-lg border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-12">
              <Checkbox
                checked={allSelected}
                onCheckedChange={toggleAll}
                aria-label="Select all active tokens"
                className={someSelected && !allSelected ? "opacity-50" : ""}
              />
            </TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Scope</TableHead>
            <TableHead>Prefix</TableHead>
            <TableHead>Owner</TableHead>
            <TableHead>Expires</TableHead>
            <TableHead>Last used</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="w-12 text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {keys.map((key) => {
            const status = getKeyStatus(key);
            const statusTone = toneFromStatus(status);
            const isSelected = selectedIds.has(key.id);
            const canSelect = isKeyActive(key);

            return (
              <TableRow
                key={key.id}
                className={isSelected ? "bg-muted/50" : undefined}
              >
                <TableCell>
                  <Checkbox
                    checked={isSelected}
                    onCheckedChange={() => toggleOne(key.id)}
                    disabled={!canSelect}
                    aria-label={`Select ${key.name}`}
                  />
                </TableCell>
                <TableCell className="font-medium">{key.name}</TableCell>
                <TableCell>
                  <Badge variant="secondary">{key.scope}</Badge>
                </TableCell>
                <TableCell className="font-mono text-sm">
                  sk-{key.prefix}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {key.owner_name || key.owner_email || "System"}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {formatDate(key.expires_at)}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {key.last_used_at ? formatDate(key.last_used_at) : "Never"}
                </TableCell>
                <TableCell>
                  <Badge className={statusToneClass(statusTone)}>
                    {status.charAt(0).toUpperCase() + status.slice(1)}
                  </Badge>
                </TableCell>
                <TableCell className="text-right">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label="Open token actions"
                      >
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuLabel>Actions</DropdownMenuLabel>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem onSelect={() => onViewDetails(key)}>
                        <Eye className="mr-2 h-4 w-4" /> View details
                      </DropdownMenuItem>
                      {canSelect && (
                        <>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            onSelect={() => onRevoke(key)}
                          >
                            <XCircle className="mr-2 h-4 w-4" /> Revoke
                          </DropdownMenuItem>
                        </>
                      )}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
