import { type FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";

export function LoginPage() {
  const navigate = useNavigate();
  const { loginLocal, beginOIDC, methods } = useAuth();
  const [email, setEmail] = useState("admin@example.com");
  const [secret, setSecret] = useState("");
  const [error, setError] = useState<string | undefined>(undefined);
  const localEnabled = methods.includes("local");
  const oidcEnabled = methods.includes("oidc");

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!localEnabled) {
      return;
    }
    loginLocal(email, secret)
      .then(() => navigate("/", { replace: true }))
      .catch(() => setError("Invalid credentials"));
  };

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-br from-background via-background to-muted">
      <Card className="w-full max-w-md shadow-xl">
        <CardHeader className="space-y-2 text-center">
          <CardTitle className="text-2xl font-semibold">
            Open Model Gateway Admin
          </CardTitle>
          <CardDescription>
            Sign in with admin credentials to manage tenants and routes.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {localEnabled ? (
            <form className="space-y-4" onSubmit={handleSubmit}>
              <div className="space-y-2 text-left">
                <label className="text-sm font-medium" htmlFor="email">
                  Email
                </label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoFocus
                />
              </div>
              <div className="space-y-2 text-left">
                <label className="text-sm font-medium" htmlFor="password">
                  Password / API Secret
                </label>
                <Input
                  id="password"
                  type="password"
                  value={secret}
                  onChange={(e) => setSecret(e.target.value)}
                  required
                />
              </div>
              {error ? (
                <p className="text-sm text-destructive">{error}</p>
              ) : null}
              <Button type="submit" className="w-full">
                Continue
              </Button>
            </form>
          ) : null}
          {localEnabled && oidcEnabled ? (
            <div className="my-6 flex items-center gap-2 text-xs uppercase tracking-wide text-muted-foreground">
              <Separator className="flex-1" />
              <span>or</span>
              <Separator className="flex-1" />
            </div>
          ) : null}
          {oidcEnabled ? (
            <Button
              type="button"
              variant="outline"
              className="w-full"
              onClick={beginOIDC}
            >
              Continue with SSO
            </Button>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
