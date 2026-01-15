import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { statusToneClass, toneFromStatus } from "@/ui/kit/status";
import type { AdminKeyRecord } from "../types";
import { getKeyStatus, isKeyActive } from "../types";

export interface AdminKeyDetailsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  record: AdminKeyRecord | null;
  onRevoke?: (record: AdminKeyRecord) => void;
}

export function AdminKeyDetailsDialog({
  open,
  onOpenChange,
  record,
  onRevoke,
}: AdminKeyDetailsDialogProps) {
  const { copy, copied } = useCopyToClipboard();

  if (!record) return null;

  const status = getKeyStatus(record);
  const statusTone = toneFromStatus(status);
  const canRevoke = isKeyActive(record);

  const formatDate = (dateStr?: string | null) => {
    if (!dateStr) return "—";
    return new Date(dateStr).toLocaleString();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <div className="flex items-start justify-between">
            <div>
              <DialogTitle className="flex items-center gap-2">
                {record.name}
                <Badge className={statusToneClass(statusTone)}>
                  {status.charAt(0).toUpperCase() + status.slice(1)}
                </Badge>
              </DialogTitle>
              <DialogDescription>Admin access token details</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="grid gap-4 sm:grid-cols-2">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Scope
                </CardTitle>
              </CardHeader>
              <CardContent>
                <Badge variant="secondary" className="text-sm">
                  {record.scope}
                </Badge>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Prefix
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex items-center gap-2">
                  <code className="font-mono text-sm">sk-{record.prefix}</code>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6"
                    onClick={() => copy(`sk-${record.prefix}`)}
                  >
                    {copied ? (
                      <Check className="h-3 w-3 text-green-500" />
                    ) : (
                      <Copy className="h-3 w-3" />
                    )}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Owner
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm">
                  {record.owner_name || record.owner_email || "System"}
                </p>
                {record.owner_email && record.owner_name && (
                  <p className="text-xs text-muted-foreground">
                    {record.owner_email}
                  </p>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Created by
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm">
                  {record.creator_name || record.creator_email || "System"}
                </p>
                {record.creator_email && record.creator_name && (
                  <p className="text-xs text-muted-foreground">
                    {record.creator_email}
                  </p>
                )}
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Created
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm">{formatDate(record.created_at)}</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Expires
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm">{formatDate(record.expires_at)}</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Last used
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm">
                  {record.last_used_at ? formatDate(record.last_used_at) : "Never"}
                </p>
              </CardContent>
            </Card>
          </div>

          {record.revoked_at && (
            <Card className="border-destructive/50 bg-destructive/5">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-destructive">
                  Revoked
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm">{formatDate(record.revoked_at)}</p>
              </CardContent>
            </Card>
          )}
        </div>

        <DialogFooter>
          {canRevoke && onRevoke && (
            <Button
              variant="destructive"
              onClick={() => onRevoke(record)}
            >
              Revoke
            </Button>
          )}
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
