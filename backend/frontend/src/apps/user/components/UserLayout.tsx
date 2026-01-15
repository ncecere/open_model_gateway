import { useState } from "react";
import { Link, NavLink, Outlet } from "react-router-dom";
import {
  Activity,
  Building2,
  FileText,
  FolderKanban,
  Key,
  LayoutDashboard,
  Menu,
  Moon,
  Sparkles,
  Sun,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/providers/ThemeProvider";
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

type NavItem = {
  path: string;
  label: string;
  icon: LucideIcon;
};

const navItems: NavItem[] = [
  { path: "/", label: "Dashboard", icon: LayoutDashboard },
  { path: "/models", label: "Models", icon: Sparkles },
  { path: "/tenants", label: "Tenants", icon: Building2 },
  { path: "/api-keys", label: "API Keys", icon: Key },
  { path: "/usage", label: "Usage", icon: Activity },
  { path: "/files", label: "Files", icon: FileText },
  { path: "/batches", label: "Batches", icon: FolderKanban },
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
  const { resolvedTheme, setThemePreference } = useTheme();

  return (
    <div className="flex h-screen overflow-hidden bg-background text-foreground">
      <Sidebar className="h-full border-r border-sidebar-border bg-sidebar">
        <SidebarHeader className="justify-between border-b border-sidebar-border px-4 py-4">
          <Link
            to="/"
            className="flex items-center gap-2 text-lg font-semibold tracking-tight"
          >
            <img
              src={LogoMark}
              alt="Open Model Gateway"
              className="h-6 w-6"
            />
            <span className="text-sidebar-foreground">User Portal</span>
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
        <SidebarContent className="px-2 py-4">
          <SidebarMenu>
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <SidebarMenuItem key={item.path}>
                  <NavLink
                    to={item.path}
                    end={item.path === "/"}
                    className={({ isActive }) =>
                      cn(
                        "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all duration-150",
                        isActive
                          ? "bg-sidebar-accent text-sidebar-accent-foreground border-l-2 border-accent"
                          : "text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground",
                      )
                    }
                    onClick={sidebar.close}
                  >
                    <Icon className="h-4 w-4 shrink-0" />
                    {item.label}
                  </NavLink>
                </SidebarMenuItem>
              );
            })}
          </SidebarMenu>
        </SidebarContent>
        <SidebarFooter className="space-y-2 border-t border-sidebar-border px-4 py-4">
          {user?.is_super_admin ? (
            <Button asChild variant="outline" size="sm" className="w-full text-xs">
              <a href="/admin/ui">Open Admin Portal</a>
            </Button>
          ) : null}
          <p className="text-xs text-sidebar-foreground/50">&copy; {new Date().getFullYear()} Open Model Gateway</p>
        </SidebarFooter>
      </Sidebar>
      <SidebarInset className="flex h-full flex-1 flex-col overflow-hidden">
        <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b bg-card/50 backdrop-blur-sm px-4 md:px-6">
          <div className="flex items-center gap-2">
            <SidebarTrigger className="md:hidden">
              <Menu className="size-5" />
            </SidebarTrigger>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setThemePreference(resolvedTheme === "dark" ? "light" : "dark")}
              className="rounded-full"
            >
              {resolvedTheme === "dark" ? (
                <Sun className="h-4 w-4" />
              ) : (
                <Moon className="h-4 w-4" />
              )}
            </Button>
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
          </div>
        </header>
        <main className="flex-1 overflow-y-auto bg-background-subtle px-4 py-6 md:px-8">
          <div className="mx-auto w-full max-w-6xl animate-in fade-in slide-in-from-bottom-2 duration-300">
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
