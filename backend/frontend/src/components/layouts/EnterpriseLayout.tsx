import { Link, NavLink, Outlet } from "react-router-dom";
import {
  LayoutDashboard,
  Boxes,
  Building2,
  Users,
  Key,
  Activity,
  BarChart3,
  FileText,
  Layers,
  Settings,
  Menu,
  ChevronLeft,
  Sun,
  Moon,
  LogOut,
  User,
  Search,
  ScrollText,
  Server,
  type LucideIcon,
} from "lucide-react";
import { useAuth } from "@/hooks/useAuth";
import { useTheme } from "@/providers/ThemeProvider";
import { Button } from "@/components/ui/button";
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
import { Breadcrumb } from "./Breadcrumb";
import LogoMark from "@/assets/system/open_model_gateway.svg";
import * as React from "react";

type AuthUser = ReturnType<typeof useAuth>["user"];
type LogoutFn = () => Promise<void> | void;

interface NavItem {
  path: string;
  label: string;
  icon: LucideIcon;
}

interface NavGroup {
  label: string;
  items: NavItem[];
}

const navGroups: NavGroup[] = [
  {
    label: "Overview",
    items: [{ path: "/", label: "Dashboard", icon: LayoutDashboard }],
  },
  {
    label: "Resources",
    items: [
      { path: "/models", label: "Models", icon: Boxes },
      { path: "/tenants", label: "Tenants", icon: Building2 },
      { path: "/users", label: "Users", icon: Users },
      { path: "/keys", label: "API Keys", icon: Key },
    ],
  },
  {
    label: "Operations",
    items: [
      { path: "/provider-health", label: "Provider Health", icon: Activity },
      { path: "/usage", label: "Usage", icon: BarChart3 },
    ],
  },
  {
    label: "Storage",
    items: [
      { path: "/files", label: "Files", icon: FileText },
      { path: "/batches", label: "Batches", icon: Layers },
    ],
  },
  {
    label: "System",
    items: [
      { path: "/system-status", label: "System Status", icon: Server },
      { path: "/audit-log", label: "Audit Log", icon: ScrollText },
      { path: "/settings", label: "Settings", icon: Settings },
    ],
  },
];

// Sidebar context for collapse state
interface SidebarContextValue {
  isCollapsed: boolean;
  isMobileOpen: boolean;
  setCollapsed: (value: boolean) => void;
  openMobile: () => void;
  closeMobile: () => void;
  toggleMobile: () => void;
}

const SidebarContext = React.createContext<SidebarContextValue | undefined>(
  undefined
);

function useSidebar() {
  const ctx = React.useContext(SidebarContext);
  if (!ctx) {
    throw new Error("useSidebar must be used within EnterpriseLayout");
  }
  return ctx;
}

export function EnterpriseLayout() {
  const { user, logout } = useAuth();

  // Desktop collapse state (persisted)
  const [isCollapsed, setCollapsedState] = React.useState(() => {
    const stored = localStorage.getItem("sidebar-collapsed");
    return stored === "true";
  });

  const setCollapsed = React.useCallback((value: boolean) => {
    setCollapsedState(value);
    localStorage.setItem("sidebar-collapsed", String(value));
  }, []);

  // Mobile open state
  const [isMobileOpen, setIsMobileOpen] = React.useState(false);

  const openMobile = React.useCallback(() => setIsMobileOpen(true), []);
  const closeMobile = React.useCallback(() => setIsMobileOpen(false), []);
  const toggleMobile = React.useCallback(
    () => setIsMobileOpen((prev) => !prev),
    []
  );

  const value = React.useMemo<SidebarContextValue>(
    () => ({
      isCollapsed,
      isMobileOpen,
      setCollapsed,
      openMobile,
      closeMobile,
      toggleMobile,
    }),
    [isCollapsed, isMobileOpen, setCollapsed, openMobile, closeMobile, toggleMobile]
  );

  return (
    <SidebarContext.Provider value={value}>
      <div className="flex h-screen overflow-hidden bg-background">
        <EnterpriseSidebar user={user} />
        <div className="flex flex-1 flex-col overflow-hidden">
          <EnterpriseHeader user={user} logout={logout} />
          <main className="flex-1 overflow-y-auto bg-background-subtle">
            <div className="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
              <Outlet />
            </div>
          </main>
        </div>
      </div>

      {/* Mobile overlay */}
      {isMobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm md:hidden"
          onClick={closeMobile}
        />
      )}

      {/* Command Palette */}
      <CommandPalette portal="admin" />
    </SidebarContext.Provider>
  );
}

function EnterpriseSidebar({ user }: { user: AuthUser }) {
  const { isCollapsed, setCollapsed, isMobileOpen, closeMobile } = useSidebar();

  return (
    <>
      {/* Desktop sidebar */}
      <aside
        className={cn(
          "hidden md:flex h-full flex-col border-r bg-card transition-all duration-slow ease-out-expo",
          isCollapsed ? "w-16" : "w-60"
        )}
      >
        <SidebarContent
          user={user}
          isCollapsed={isCollapsed}
          onCollapseToggle={() => setCollapsed(!isCollapsed)}
        />
      </aside>

      {/* Mobile sidebar */}
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-60 flex-col border-r bg-card shadow-soft-xl transition-transform duration-slow ease-out-expo md:hidden",
          isMobileOpen ? "translate-x-0" : "-translate-x-full"
        )}
      >
        <SidebarContent
          user={user}
          isCollapsed={false}
          onNavClick={closeMobile}
        />
      </aside>
    </>
  );
}

interface SidebarContentProps {
  user: AuthUser;
  isCollapsed: boolean;
  onCollapseToggle?: () => void;
  onNavClick?: () => void;
}

function SidebarContent({
  user,
  isCollapsed,
  onCollapseToggle,
  onNavClick,
}: SidebarContentProps) {
  return (
    <>
      {/* Header */}
      <div
        className={cn(
          "flex h-16 shrink-0 items-center border-b px-4",
          isCollapsed ? "justify-center" : "justify-between"
        )}
      >
        <Link
          to="/"
          className="flex items-center gap-2.5 overflow-hidden"
          onClick={onNavClick}
        >
          <img
            src={LogoMark}
            alt="Open Model Gateway"
            className="h-7 w-7 shrink-0"
          />
          {!isCollapsed && (
            <span className="truncate text-sm font-semibold tracking-tight">
              Open Model Gateway
            </span>
          )}
        </Link>
        {!isCollapsed && onCollapseToggle && (
          <button
            onClick={onCollapseToggle}
            className="hidden rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground md:flex"
          >
            <ChevronLeft className="h-4 w-4" />
            <span className="sr-only">Collapse sidebar</span>
          </button>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        <div className="space-y-6">
          {navGroups.map((group) => (
            <div key={group.label}>
              {!isCollapsed && (
                <h3 className="mb-2 px-3 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                  {group.label}
                </h3>
              )}
              <ul className="space-y-1">
                {group.items.map((item) => (
                  <li key={item.path}>
                    <NavLink
                      to={item.path}
                      end={item.path === "/"}
                      onClick={onNavClick}
                      className={({ isActive }) =>
                        cn(
                          "group flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all duration-fast",
                          isCollapsed && "justify-center px-2",
                          isActive
                            ? "bg-primary/10 text-primary border-l-2 border-accent ml-[-1px]"
                            : "text-muted-foreground hover:bg-muted hover:text-foreground"
                        )
                      }
                    >
                      <item.icon
                        className={cn(
                          "h-[18px] w-[18px] shrink-0 transition-colors",
                          "group-hover:text-foreground"
                        )}
                      />
                      {!isCollapsed && <span>{item.label}</span>}
                    </NavLink>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </nav>

      {/* Footer */}
      <div
        className={cn(
          "shrink-0 border-t p-4",
          isCollapsed && "flex justify-center"
        )}
      >
        {!isCollapsed ? (
          <div className="space-y-3">
            {user?.is_super_admin && (
              <Button asChild variant="outline" size="sm" className="w-full">
                <a href="/">User Portal</a>
              </Button>
            )}
            <p className="text-center text-[11px] text-muted-foreground">
              &copy; {new Date().getFullYear()} Open Model Gateway
            </p>
          </div>
        ) : (
          onCollapseToggle && (
            <button
              onClick={onCollapseToggle}
              className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              <ChevronLeft className="h-4 w-4 rotate-180" />
              <span className="sr-only">Expand sidebar</span>
            </button>
          )
        )}
      </div>
    </>
  );
}

function EnterpriseHeader({
  user,
  logout,
}: {
  user: AuthUser;
  logout: LogoutFn;
}) {
  const { toggleMobile } = useSidebar();
  const { resolvedTheme, setThemePreference } = useTheme();
  const { trigger: openCommandPalette } = useCommandPalette();

  const toggleTheme = () => {
    setThemePreference(resolvedTheme === "dark" ? "light" : "dark");
  };

  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b bg-card px-4 sm:px-6">
      <div className="flex items-center gap-3">
        <button
          onClick={toggleMobile}
          className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground md:hidden"
        >
          <Menu className="h-5 w-5" />
          <span className="sr-only">Open sidebar</span>
        </button>

        <Breadcrumb />
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

        {/* Theme toggle */}
        <button
          onClick={toggleTheme}
          className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          aria-label="Toggle theme"
        >
          {resolvedTheme === "dark" ? (
            <Sun className="h-4 w-4" />
          ) : (
            <Moon className="h-4 w-4" />
          )}
        </button>

        {/* User menu */}
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
                {user?.email ?? "System"}
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel className="font-normal">
              <div className="flex flex-col space-y-1">
                <p className="text-sm font-medium leading-none">Account</p>
                <p className="text-xs leading-none text-muted-foreground">
                  {user?.email ?? "system@open-model-gateway"}
                </p>
              </div>
            </DropdownMenuLabel>
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
  );
}
