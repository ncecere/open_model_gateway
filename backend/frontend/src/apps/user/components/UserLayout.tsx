import { useState } from "react";
import { Link, NavLink, Outlet } from "react-router-dom";
import {
  Activity,
  Building2,
  FileText,
  FolderKanban,
  Key,
  LayoutDashboard,
  LogOut,
  Menu,
  Moon,
  Search,
  Sparkles,
  Sun,
  User,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/providers/ThemeProvider";
import { CommandPalette, useCommandPalette } from "@/components/CommandPalette";
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
  const { trigger: openCommandPalette } = useCommandPalette();

  return (
    <div className="flex h-screen overflow-hidden bg-background text-foreground">
      <Sidebar className="h-full">
        <SidebarHeader className="justify-between">
          <Link
            to="/"
            className="flex items-center gap-2.5"
          >
            <img
              src={LogoMark}
              alt="Open Model Gateway"
              className="h-7 w-7 shrink-0"
            />
            <span className="truncate text-sm font-semibold tracking-tight">
              Open Model Gateway
            </span>
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
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <SidebarMenuItem key={item.path}>
                  <NavLink
                    to={item.path}
                    end={item.path === "/"}
                    className={({ isActive }) =>
                      cn(
                        "group flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all duration-fast",
                        isActive
                          ? "bg-primary/10 text-primary border-l-2 border-accent ml-[-1px]"
                          : "text-muted-foreground hover:bg-muted hover:text-foreground",
                      )
                    }
                    onClick={sidebar.close}
                  >
                    <Icon className="h-[18px] w-[18px] shrink-0 transition-colors group-hover:text-foreground" />
                    <span>{item.label}</span>
                  </NavLink>
                </SidebarMenuItem>
              );
            })}
          </SidebarMenu>
        </SidebarContent>
        <SidebarFooter className="space-y-3">
          {user?.is_super_admin ? (
            <Button asChild variant="outline" size="sm" className="w-full">
              <a href="/admin/ui">Admin Portal</a>
            </Button>
          ) : null}
          <p className="text-center text-[11px] text-muted-foreground">
            &copy; {new Date().getFullYear()} Open Model Gateway
          </p>
        </SidebarFooter>
      </Sidebar>
      <SidebarInset className="flex h-full flex-1 flex-col overflow-hidden">
        <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b bg-card px-4 sm:px-6">
          <div className="flex items-center gap-3">
            <SidebarTrigger className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground md:hidden">
              <Menu className="h-5 w-5" />
              <span className="sr-only">Open sidebar</span>
            </SidebarTrigger>
          </div>
          <div className="flex items-center gap-2">
            {/* Search / Command Palette trigger */}
            <button
              onClick={openCommandPalette}
              className="hidden items-center gap-2 rounded-md border bg-muted/50 px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground sm:flex"
            >
              <Search className="h-3.5 w-3.5" />
              <span className="text-xs">Search...</span>
              <kbd className="pointer-events-none ml-2 hidden select-none rounded border bg-background px-1.5 py-0.5 font-mono text-[10px] font-medium text-muted-foreground lg:inline-block">
                ⌘K
              </kbd>
            </button>
            <button
              onClick={openCommandPalette}
              className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground sm:hidden"
              aria-label="Open search"
            >
              <Search className="h-4 w-4" />
            </button>

            <button
              onClick={() => setThemePreference(resolvedTheme === "dark" ? "light" : "dark")}
              className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              aria-label="Toggle theme"
            >
              {resolvedTheme === "dark" ? (
                <Sun className="h-4 w-4" />
              ) : (
                <Moon className="h-4 w-4" />
              )}
            </button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  className="flex h-9 items-center gap-2 px-2"
                >
                  <div className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-primary">
                    <User className="h-4 w-4" />
                  </div>
                  <span className="hidden max-w-[150px] truncate text-sm font-medium sm:block">
                    {user?.email ?? "user@example.com"}
                  </span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                <DropdownMenuLabel className="font-normal">
                  <div className="flex flex-col space-y-1">
                    <p className="text-sm font-medium leading-none">Account</p>
                    <p className="text-xs leading-none text-muted-foreground">
                      {user?.email ?? "user@example.com"}
                    </p>
                  </div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onSelect={(event) => {
                    event.preventDefault();
                    onProfileOpenChange(true);
                  }}
                >
                  <User className="mr-2 h-4 w-4" />
                  Profile
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onSelect={() => logout()}
                  className="text-destructive focus:text-destructive"
                >
                  <LogOut className="mr-2 h-4 w-4" />
                  Sign out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </header>
        <main className="flex-1 overflow-y-auto bg-background-subtle px-4 py-6 sm:px-6 lg:px-8">
          <div className="mx-auto w-full max-w-7xl">
            <Outlet />
          </div>
        </main>
      </SidebarInset>
      <UserProfileDialog
        open={profileOpen}
        onOpenChange={onProfileOpenChange}
      />

      {/* Command Palette */}
      <CommandPalette portal="user" />
    </div>
  );
}
