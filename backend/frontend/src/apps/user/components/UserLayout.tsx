import { useState } from "react";
import { Link, NavLink, Outlet } from "react-router-dom";
import { Menu } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { useUserAuth } from "../hooks/useUserAuth";
import { UserProfileDialog } from "./ProfileDialog";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar";
import LogoMark from "@/assets/system/open_model_gateway.svg";

type UserPortalUser = ReturnType<typeof useUserAuth>["user"];
type LogoutFn = () => Promise<void> | void;

const navItems = [
  { path: "/", label: "Dashboard" },
  { path: "/models", label: "Models" },
  { path: "/tenants", label: "Tenants" },
  { path: "/api-keys", label: "API Keys" },
  { path: "/usage", label: "Usage" },
  { path: "/files", label: "Files" },
  { path: "/batches", label: "Batches" },
];

export function UserLayout() {
  const { user, logout } = useUserAuth();
  const [profileOpen, setProfileOpen] = useState(false);

  return (
    <SidebarProvider>
      <UserShell
        user={user}
        logout={logout}
        profileOpen={profileOpen}
        onProfileOpenChange={setProfileOpen}
      />
    </SidebarProvider>
  );
}

type UserShellProps = {
  user: UserPortalUser;
  logout: LogoutFn;
  profileOpen: boolean;
  onProfileOpenChange: (open: boolean) => void;
};

function UserShell({
  user,
  logout,
  profileOpen,
  onProfileOpenChange,
}: UserShellProps) {
  const sidebar = useSidebar();

  return (
    <div className="flex min-h-screen bg-background text-foreground">
      <Sidebar>
        <SidebarHeader className="justify-between">
          <Link
            to="/"
            className="flex items-center gap-2 text-lg font-semibold tracking-tight"
          >
            <img
              src={LogoMark}
              alt="Open Model Gateway"
              className="h-6 w-6"
            />
            <span>Open Model Gateway</span>
          </Link>
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            onClick={sidebar.close}
          >
            <Menu className="size-5" />
            <span className="sr-only">Close sidebar</span>
          </Button>
        </SidebarHeader>
        <SidebarContent>
          <SidebarMenu>
            {navItems.map((item) => (
              <SidebarMenuItem key={item.path}>
                <NavLink
                  to={item.path}
                  end={item.path === "/"}
                  className={({ isActive }) =>
                    cn(
                      "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                      isActive
                        ? "bg-primary text-primary-foreground shadow"
                        : "text-muted-foreground hover:bg-muted hover:text-foreground",
                    )
                  }
                  onClick={sidebar.close}
                >
                  {item.label}
                </NavLink>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarContent>
        <SidebarFooter className="space-y-2">
          {user?.is_super_admin ? (
            <Button asChild variant="secondary" className="w-full text-xs">
              <a href="/admin/ui">Open Admin Portal</a>
            </Button>
          ) : null}
          <p>&copy; {new Date().getFullYear()} Open Model Gateway</p>
        </SidebarFooter>
      </Sidebar>
      <SidebarInset>
        <header className="flex h-16 items-center justify-between gap-4 border-b bg-card px-4 md:px-6">
          <div className="flex items-center gap-2">
            <SidebarTrigger className="md:hidden">
              <Menu className="size-5" />
            </SidebarTrigger>
            <span className="text-sm text-muted-foreground">User Portal</span>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                className="flex items-center gap-2 truncate max-w-[220px]"
              >
                <span className="truncate text-sm font-medium">
                  {user?.email ?? "user@example.com"}
                </span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <DropdownMenuLabel>Signed in</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onSelect={(event) => {
                  event.preventDefault();
                  onProfileOpenChange(true);
                }}
              >
                Profile
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={() => logout()}>
                Logout
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </header>
        <main className="flex-1 overflow-y-auto bg-muted/30 px-4 py-6 md:px-8">
          <div className="mx-auto w-full max-w-5xl">
            <Outlet />
          </div>
        </main>
      </SidebarInset>
      <UserProfileDialog
        open={profileOpen}
        onOpenChange={onProfileOpenChange}
      />
    </div>
  );
}
