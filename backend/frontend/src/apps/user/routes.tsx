import type { ReactElement } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { useUserAuth } from "./hooks/useUserAuth";
import { UserLoginPage } from "./pages/LoginPage";
import { UserLayout } from "./components/UserLayout";
import { UserDashboardPage } from "./pages/DashboardPage";
import { UserUsagePage } from "./pages/UsagePage";
import { UserApiKeysPage } from "./pages/ApiKeysPage";
import { UserTenantsPage } from "./pages/TenantsPage";
import { UserBatchesPage } from "./pages/BatchesPage";
import { UserFilesPage } from "./pages/FilesPage";
import { UserOIDCRedirectPage } from "./pages/OIDCRedirectPage";

function ProtectedRoute({ children }: { children: ReactElement }) {
  const { isAuthenticated } = useUserAuth();
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  return children;
}

function GuestRoute({ children }: { children: ReactElement }) {
  const { isAuthenticated } = useUserAuth();
  if (isAuthenticated) {
    return <Navigate to="/" replace />;
  }
  return children;
}

export function UserRoutes() {
  return (
    <Routes>
      <Route
        path="/login"
        element={
          <GuestRoute>
            <UserLoginPage />
          </GuestRoute>
        }
      />
      <Route path="/auth/oidc/callback" element={<UserOIDCRedirectPage />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <UserLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<UserDashboardPage />} />
        <Route path="tenants" element={<UserTenantsPage />} />
        <Route path="usage" element={<UserUsagePage />} />
        <Route path="api-keys" element={<UserApiKeysPage />} />
        <Route path="files" element={<UserFilesPage />} />
        <Route path="batches" element={<UserBatchesPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
