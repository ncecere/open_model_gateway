import { BrowserRouter } from "react-router-dom";

import { AuthProvider } from "../../auth/AuthProvider";
import { QueryProvider } from "../../providers/QueryProvider";
import { ThemeProvider } from "../../providers/ThemeProvider";
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
  const { isAuthenticated } = useAuth();

  return (
    <ThemeProvider isAuthenticated={isAuthenticated}>
      <BrowserRouter basename="/admin/ui">
        <AppRoutes />
      </BrowserRouter>
      <Toaster />
      <SonnerToaster richColors position="top-right" />
    </ThemeProvider>
  );
}

export default AdminApp;
