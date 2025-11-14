import { BrowserRouter } from "react-router-dom";

import { QueryProvider } from "../../providers/QueryProvider";
import { Toaster } from "@/components/ui/toaster";
import { Toaster as SonnerToaster } from "@/components/ui/sonner";
import { UserAuthProvider } from "./auth/UserAuthProvider";
import { UserRoutes } from "./routes";

export function UserApp() {
  return (
    <QueryProvider>
      <UserAuthProvider>
        <BrowserRouter>
          <UserRoutes />
        </BrowserRouter>
        <Toaster />
        <SonnerToaster richColors position="top-right" />
      </UserAuthProvider>
    </QueryProvider>
  );
}

export default UserApp;
