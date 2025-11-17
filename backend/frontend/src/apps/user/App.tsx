import { BrowserRouter } from "react-router-dom";

import { QueryProvider } from "../../providers/QueryProvider";
import { ThemeProvider } from "../../providers/ThemeProvider";
import { Toaster } from "@/components/ui/toaster";
import { Toaster as SonnerToaster } from "@/components/ui/sonner";
import { UserAuthProvider } from "./auth/UserAuthProvider";
import { useUserAuth } from "./hooks";
import { useUserProfileQuery } from "./hooks/useUserData";
import { UserRoutes } from "./routes";

export function UserApp() {
  return (
    <QueryProvider>
      <UserAuthProvider>
        <UserThemeBoundary />
      </UserAuthProvider>
    </QueryProvider>
  );
}

function UserThemeBoundary() {
  const { isAuthenticated } = useUserAuth();
  const profileQuery = useUserProfileQuery({ enabled: isAuthenticated });

  return (
    <ThemeProvider
      isAuthenticated={isAuthenticated}
      profileQuery={profileQuery}
    >
      <BrowserRouter>
        <UserRoutes />
      </BrowserRouter>
      <Toaster />
      <SonnerToaster richColors position="top-right" />
    </ThemeProvider>
  );
}

export default UserApp;
