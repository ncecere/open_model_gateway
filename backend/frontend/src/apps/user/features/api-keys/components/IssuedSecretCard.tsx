import { Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export type IssuedSecret = {
  scope: "personal" | "tenant";
  tenantName?: string;
  name: string;
  prefix: string;
  secret: string;
  token: string;
};

export type IssuedSecretCardProps = {
  issued: IssuedSecret;
  onCopy: (value: string) => void;
};

export function IssuedSecretCard({ issued, onCopy }: IssuedSecretCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Save this secret</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        <p>
          {issued.scope === "tenant"
            ? `Issued for ${issued.tenantName ?? "selected tenant"}.`
            : "Issued for your personal tenant."}{" "}
          This is the only time the full secret will be shown.
        </p>
        <div className="rounded-md border p-3">
          <p className="text-xs text-muted-foreground">Token</p>
          <div className="mt-1 flex items-center justify-between gap-4">
            <code className="truncate">{issued.token}</code>
            <Button variant="ghost" size="sm" onClick={() => onCopy(issued.token)}>
              <Copy className="mr-1 size-4" /> Copy
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
