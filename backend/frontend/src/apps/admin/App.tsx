import { BrowserRouter } from "react-router-dom";

import { AuthProvider } from "../../auth/AuthProvider";
import { QueryProvider } from "../../providers/QueryProvider";
import { AppRoutes } from "../../routes";
import { Toaster } from "@/components/ui/toaster";
import { Toaster as SonnerToaster } from "@/components/ui/sonner";

export function AdminApp() {
  return (
    <QueryProvider>
      <AuthProvider>
        <BrowserRouter basename="/admin/ui">
          <AppRoutes />
        </BrowserRouter>
        <Toaster />
        <SonnerToaster richColors position="top-right" />
      </AuthProvider>
    </QueryProvider>
  );
}

export default AdminApp;
