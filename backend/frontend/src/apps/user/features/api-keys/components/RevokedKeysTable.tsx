import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { UserAPIKey } from "@/api/user/api-keys";

export type RevokedRow = UserAPIKey & { tenantLabel: string };

export type RevokedKeysTableProps = {
  keys: RevokedRow[];
  loading: boolean;
};

export function RevokedKeysTable({ keys, loading }: RevokedKeysTableProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Revoked keys</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="space-y-2">
            {[...Array(4)].map((_, idx) => (
              <div key={idx} className="h-10 animate-pulse rounded bg-muted" />
            ))}
          </div>
        ) : keys.length ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Prefix</TableHead>
                <TableHead>Scope</TableHead>
                <TableHead>Revoked at</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((key) => (
                <TableRow key={key.id}>
                  <TableCell>{key.name}</TableCell>
                  <TableCell>{key.prefix}</TableCell>
                  <TableCell>{key.tenantLabel}</TableCell>
                  <TableCell>
                    {key.revoked_at
                      ? new Date(key.revoked_at).toLocaleDateString()
                      : "—"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <p className="text-sm text-muted-foreground">No revoked keys yet.</p>
        )}
      </CardContent>
    </Card>
  );
}
