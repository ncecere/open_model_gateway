import { BrowserRouter } from "react-router-dom";

import { AuthProvider } from "../../auth/AuthProvider";
import { QueryProvider } from "../../providers/QueryProvider";
import { ThemeProvider } from "../../providers/ThemeProvider";
import { DirectoryProvider } from "../../providers/DirectoryProvider";
import { useAuth } from "../../hooks/useAuth";
import { AppRoutes } from "../../routes";
import { Toaster } from "@/components/ui/toaster";
import { Toaster as SonnerToaster } from "@/components/ui/sonner";

export function AdminApp() {
  return (
    <QueryProvider>
      <AuthProvider>
        <AdminThemeBoundary />
      </AuthProvider>
    </QueryProvider>
  );
}

function AdminThemeBoundary() {
  const { isAuthenticated, user } = useAuth();
  const profileQuery = user
    ? { data: { theme_preference: user.theme_preference }, isFetching: false }
    : undefined;

  return (
    <ThemeProvider
      isAuthenticated={isAuthenticated}
      profileQuery={profileQuery}
    >
      <DirectoryProvider>
        <BrowserRouter basename="/admin/ui">
          <AppRoutes />
        </BrowserRouter>
      </DirectoryProvider>
      <Toaster />
      <SonnerToaster richColors position="top-right" />
    </ThemeProvider>
  );
}

export default AdminApp;
