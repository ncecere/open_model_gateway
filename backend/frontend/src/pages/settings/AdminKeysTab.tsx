import { Copy } from "lucide-react";

import type { AdminKeyRecord, AdminKeyScope } from "@/api/admin-keys";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { TabsContent } from "@/components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

type AdminKeysTabProps = {
  adminKeys: AdminKeyRecord[];
  loading: boolean;
  createOpen: boolean;
  setCreateOpen: (open: boolean) => void;
  issuedToken: string | null;
  setIssuedToken: (token: string | null) => void;
  pendingRevoke: AdminKeyRecord | null;
  setPendingRevoke: (record: AdminKeyRecord | null) => void;
  adminKeyName: string;
  setAdminKeyName: (value: string) => void;
  adminKeyScope: AdminKeyScope;
  setAdminKeyScope: (value: AdminKeyScope) => void;
  adminKeyExpiresDays: string;
  setAdminKeyExpiresDays: (value: string) => void;
  createPending: boolean;
  revokePending: boolean;
  onCreate: () => void;
  onRevoke: (id: string) => void;
  onCopy: (value: string) => void;
  formatDate: (value?: string | null) => string;
  getStatus: (key: AdminKeyRecord) => string;
};

export function AdminKeysTab({
  adminKeys,
  loading,
  createOpen,
  setCreateOpen,
  issuedToken,
  setIssuedToken,
  pendingRevoke,
  setPendingRevoke,
  adminKeyName,
  setAdminKeyName,
  adminKeyScope,
  setAdminKeyScope,
  adminKeyExpiresDays,
  setAdminKeyExpiresDays,
  createPending,
  revokePending,
  onCreate,
  onRevoke,
  onCopy,
  formatDate,
  getStatus,
}: AdminKeysTabProps) {
  return (
    <TabsContent value="admin-keys" forceMount>
      <Card>
        <CardHeader className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div className="space-y-2">
            <CardTitle>Admin access tokens</CardTitle>
            <p className="text-sm text-muted-foreground">
              Create time-bound admin tokens for automation and operational scripts.
              Tokens are shown once; store them securely.
            </p>
          </div>
          <Button onClick={() => setCreateOpen(true)}>Create token</Button>
        </CardHeader>
        <CardContent className="space-y-4">
          {loading ? (
            <Skeleton className="h-32 w-full" />
          ) : adminKeys.length ? (
            <div className="rounded-lg border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Scope</TableHead>
                    <TableHead>Prefix</TableHead>
                    <TableHead>Owner</TableHead>
                    <TableHead>Expires</TableHead>
                    <TableHead>Last used</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="w-24 text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {adminKeys.map((key) => {
                    const status = getStatus(key);
                    return (
                      <TableRow key={key.id}>
                        <TableCell className="font-medium">{key.name}</TableCell>
                        <TableCell>
                          <Badge variant="secondary">{key.scope}</Badge>
                        </TableCell>
                        <TableCell className="font-mono">{`sk-${key.prefix}`}</TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {key.owner_name || key.owner_email || "System"}
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {formatDate(key.expires_at)}
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {formatDate(key.last_used_at)}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={status === "active" ? "secondary" : "destructive"}
                          >
                            {status}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={status !== "active"}
                            onClick={() => setPendingRevoke(key)}
                          >
                            Revoke
                          </Button>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              No admin tokens created yet.
            </p>
          )}
        </CardContent>
      </Card>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
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
                value={adminKeyName}
                onChange={(event) => setAdminKeyName(event.target.value)}
                placeholder="Billing automation"
              />
            </div>
            <div className="space-y-2">
              <Label>Scope</Label>
              <Select
                value={adminKeyScope}
                onValueChange={(value) => setAdminKeyScope(value as AdminKeyScope)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select scope" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="admin">Admin (current user)</SelectItem>
                  <SelectItem value="system">System</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="admin-key-expiry">Expires in (days)</Label>
              <Input
                id="admin-key-expiry"
                type="number"
                min="1"
                value={adminKeyExpiresDays}
                onChange={(event) => setAdminKeyExpiresDays(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              Cancel
            </Button>
            <Button onClick={onCreate} disabled={createPending}>
              {createPending ? "Creating…" : "Create token"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {issuedToken ? (
        <Dialog open onOpenChange={(open) => !open && setIssuedToken(null)}>
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
                <Input value={issuedToken} readOnly className="font-mono" />
                <Button variant="outline" size="icon" onClick={() => onCopy(issuedToken)}>
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>
            <DialogFooter>
              <Button onClick={() => setIssuedToken(null)}>Done</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}

      <AlertDialog
        open={Boolean(pendingRevoke)}
        onOpenChange={(open) => !open && setPendingRevoke(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke admin token?</AlertDialogTitle>
            <AlertDialogDescription>
              This token will stop working immediately. You cannot undo this action.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (pendingRevoke) {
                  onRevoke(pendingRevoke.id);
                }
              }}
              disabled={revokePending}
            >
              Revoke
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </TabsContent>
  );
}
