import { useEffect, useRef } from "react";
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
  const { isAuthenticated, user, refresh } = useUserAuth();
  const profileQuery = useUserProfileQuery({ enabled: isAuthenticated });
  const mismatchRef = useRef(false);
  const { refetch } = profileQuery;
  const profileEmail = profileQuery.data?.email?.toLowerCase();
  const sessionEmail = user?.email?.toLowerCase();

  useEffect(() => {
    if (
      !isAuthenticated ||
      !profileEmail ||
      !sessionEmail ||
      profileEmail === sessionEmail ||
      mismatchRef.current
    ) {
      return;
    }
    mismatchRef.current = true;
    refresh()
      .catch(() => {
        // ignore refresh failures, next navigation/login will handle it
      })
      .finally(() => {
        mismatchRef.current = false;
        void refetch();
      });
  }, [isAuthenticated, profileEmail, sessionEmail, refresh, refetch]);

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
