import type { ReactElement } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import { DashboardPage } from "../pages/DashboardPage";
import { KeysPage } from "../pages/KeysPage";
import { LoginPage } from "../pages/LoginPage";
import { ModelsPage } from "../pages/ModelsPage";
import { ProviderHealthPage } from "../pages/ProviderHealthPage";
import { TenantsPage } from "../pages/TenantsPage";
import { UsersPage } from "../pages/UsersPage";
import { UsagePage } from "../pages/UsagePage";
import { BatchesPage } from "../pages/BatchesPage";
import { FilesPage } from "../pages/FilesPage";
import { EnterpriseLayout } from "../components/layouts";
import { OIDCRedirectPage } from "../pages/OIDCRedirectPage";
import { SettingsPage } from "../pages/SettingsPage";

function ProtectedRoute({ children }: { children: ReactElement }) {
  const { isAuthenticated } = useAuth();
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  return children;
}

function GuestRoute({ children }: { children: ReactElement }) {
  const { isAuthenticated } = useAuth();
  if (isAuthenticated) {
    return <Navigate to="/" replace />;
  }
  return children;
}

export function AppRoutes() {
  return (
    <Routes>
      <Route
        path="/login"
        element={
          <GuestRoute>
            <LoginPage />
          </GuestRoute>
        }
      />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <EnterpriseLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<DashboardPage />} />
        <Route path="tenants" element={<TenantsPage />} />
        <Route path="users" element={<UsersPage />} />
        <Route path="keys" element={<KeysPage />} />
        <Route path="models" element={<ModelsPage />} />
        <Route path="provider-health" element={<ProviderHealthPage />} />
        <Route path="usage" element={<UsagePage />} />
        <Route path="files" element={<FilesPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="batches" element={<BatchesPage />} />
      </Route>
      <Route path="/auth/oidc/callback" element={<OIDCRedirectPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
