import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

export function OIDCRedirectPage() {
  const navigate = useNavigate();
  const { completeOIDC } = useAuth();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    completeOIDC(params)
      .then(() => navigate("/", { replace: true }))
      .catch(() => navigate("/login", { replace: true }));
  }, [completeOIDC, navigate]);

  return (
    <div className="login-wrapper">
      <div className="login-card">
        <h1>Completing sign-in…</h1>
        <p>You will be redirected shortly.</p>
      </div>
    </div>
  );
}
