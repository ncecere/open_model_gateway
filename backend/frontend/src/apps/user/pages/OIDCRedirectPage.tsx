import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useUserAuth } from "../hooks/useUserAuth";

export function UserOIDCRedirectPage() {
  const navigate = useNavigate();
  const { completeOIDC } = useUserAuth();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    completeOIDC(params)
      .then(() => navigate("/", { replace: true }))
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Sign-in failed");
        setTimeout(() => navigate("/login", { replace: true }), 2000);
      });
  }, [completeOIDC, navigate]);

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-br from-background via-background to-muted">
      <div className="w-full max-w-md rounded-lg border bg-card p-8 text-center shadow-xl">
        <h1 className="text-2xl font-semibold text-card-foreground">
          {error ? "Sign-in failed" : "Completing sign-in…"}
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          {error
            ? error
            : "You will be redirected shortly once your session is ready."}
        </p>
      </div>
    </div>
  );
}
