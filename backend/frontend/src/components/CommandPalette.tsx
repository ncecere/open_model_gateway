"use client";

import * as React from "react";
import { useNavigate } from "react-router-dom";
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
  Sun,
  Moon,
  Sparkles,
  FolderKanban,
  ScrollText,
  Server,
  type LucideIcon,
} from "lucide-react";

import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandShortcut,
} from "@/components/ui/command";
import { useTheme } from "@/providers/ThemeProvider";

interface CommandAction {
  id: string;
  label: string;
  icon: LucideIcon;
  shortcut?: string;
  action: () => void;
  keywords?: string[];
}

interface CommandPaletteProps {
  /** The portal type determines which navigation items are shown */
  portal: "admin" | "user";
}

export function CommandPalette({ portal }: CommandPaletteProps) {
  const [open, setOpen] = React.useState(false);
  const navigate = useNavigate();
  const { resolvedTheme, setThemePreference } = useTheme();

  // Navigation actions based on portal type
  const navigationActions: CommandAction[] = React.useMemo(() => {
    if (portal === "admin") {
      return [
        {
          id: "dashboard",
          label: "Dashboard",
          icon: LayoutDashboard,
          action: () => navigate("/"),
          keywords: ["home", "overview"],
        },
        {
          id: "models",
          label: "Models",
          icon: Boxes,
          action: () => navigate("/models"),
          keywords: ["llm", "gpt", "claude", "catalog"],
        },
        {
          id: "tenants",
          label: "Tenants",
          icon: Building2,
          action: () => navigate("/tenants"),
          keywords: ["organization", "team", "company"],
        },
        {
          id: "users",
          label: "Users",
          icon: Users,
          action: () => navigate("/users"),
          keywords: ["members", "accounts"],
        },
        {
          id: "keys",
          label: "API Keys",
          icon: Key,
          action: () => navigate("/keys"),
          keywords: ["tokens", "authentication", "api"],
        },
        {
          id: "provider-health",
          label: "Provider Health",
          icon: Activity,
          action: () => navigate("/provider-health"),
          keywords: ["status", "monitoring", "health"],
        },
        {
          id: "usage",
          label: "Usage",
          icon: BarChart3,
          action: () => navigate("/usage"),
          keywords: ["analytics", "metrics", "stats"],
        },
        {
          id: "files",
          label: "Files",
          icon: FileText,
          action: () => navigate("/files"),
          keywords: ["documents", "uploads"],
        },
        {
          id: "batches",
          label: "Batches",
          icon: Layers,
          action: () => navigate("/batches"),
          keywords: ["jobs", "async", "processing"],
        },
        {
          id: "system-status",
          label: "System Status",
          icon: Server,
          action: () => navigate("/system-status"),
          keywords: ["health", "monitoring", "uptime"],
        },
        {
          id: "audit-log",
          label: "Audit Log",
          icon: ScrollText,
          action: () => navigate("/audit-log"),
          keywords: ["activity", "history", "events"],
        },
        {
          id: "settings",
          label: "Settings",
          icon: Settings,
          action: () => navigate("/settings"),
          keywords: ["config", "preferences"],
        },
      ];
    }

    // User portal
    return [
      {
        id: "dashboard",
        label: "Dashboard",
        icon: LayoutDashboard,
        action: () => navigate("/"),
        keywords: ["home", "overview"],
      },
      {
        id: "models",
        label: "Models",
        icon: Sparkles,
        action: () => navigate("/models"),
        keywords: ["llm", "gpt", "claude", "catalog"],
      },
      {
        id: "tenants",
        label: "Tenants",
        icon: Building2,
        action: () => navigate("/tenants"),
        keywords: ["organization", "team", "company"],
      },
      {
        id: "api-keys",
        label: "API Keys",
        icon: Key,
        action: () => navigate("/api-keys"),
        keywords: ["tokens", "authentication", "api"],
      },
      {
        id: "usage",
        label: "Usage",
        icon: Activity,
        action: () => navigate("/usage"),
        keywords: ["analytics", "metrics", "stats"],
      },
      {
        id: "files",
        label: "Files",
        icon: FileText,
        action: () => navigate("/files"),
        keywords: ["documents", "uploads"],
      },
      {
        id: "batches",
        label: "Batches",
        icon: FolderKanban,
        action: () => navigate("/batches"),
        keywords: ["jobs", "async", "processing"],
      },
    ];
  }, [navigate, portal]);

  // Theme actions
  const themeActions: CommandAction[] = React.useMemo(
    () => [
      {
        id: "toggle-theme",
        label: resolvedTheme === "dark" ? "Switch to Light Mode" : "Switch to Dark Mode",
        icon: resolvedTheme === "dark" ? Sun : Moon,
        action: () => setThemePreference(resolvedTheme === "dark" ? "light" : "dark"),
        keywords: ["theme", "dark", "light", "mode", "appearance"],
      },
    ],
    [resolvedTheme, setThemePreference]
  );

  // Quick actions
  const quickActions: CommandAction[] = React.useMemo(() => {
    if (portal === "admin") {
      return [
        {
          id: "create-tenant",
          label: "Create New Tenant",
          icon: Building2,
          action: () => navigate("/tenants?action=create"),
          keywords: ["new", "add", "tenant"],
        },
        {
          id: "create-user",
          label: "Create New User",
          icon: Users,
          action: () => navigate("/users?action=create"),
          keywords: ["new", "add", "user"],
        },
        {
          id: "create-key",
          label: "Create API Key",
          icon: Key,
          action: () => navigate("/keys?action=create"),
          keywords: ["new", "add", "key", "token"],
        },
      ];
    }

    return [
      {
        id: "create-key",
        label: "Create API Key",
        icon: Key,
        action: () => navigate("/api-keys?action=create"),
        keywords: ["new", "add", "key", "token"],
      },
    ];
  }, [navigate, portal]);

  // Global keyboard shortcut
  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const runAction = (action: CommandAction) => {
    setOpen(false);
    action.action();
  };

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <CommandInput placeholder="Type a command or search..." />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        <CommandGroup heading="Navigation">
          {navigationActions.map((action) => (
            <CommandItem
              key={action.id}
              value={`${action.label} ${action.keywords?.join(" ") ?? ""}`}
              onSelect={() => runAction(action)}
            >
              <action.icon className="mr-2 h-4 w-4" />
              <span>{action.label}</span>
              {action.shortcut && (
                <CommandShortcut>{action.shortcut}</CommandShortcut>
              )}
            </CommandItem>
          ))}
        </CommandGroup>

        <CommandGroup heading="Quick Actions">
          {quickActions.map((action) => (
            <CommandItem
              key={action.id}
              value={`${action.label} ${action.keywords?.join(" ") ?? ""}`}
              onSelect={() => runAction(action)}
            >
              <action.icon className="mr-2 h-4 w-4" />
              <span>{action.label}</span>
            </CommandItem>
          ))}
        </CommandGroup>

        <CommandGroup heading="Preferences">
          {themeActions.map((action) => (
            <CommandItem
              key={action.id}
              value={`${action.label} ${action.keywords?.join(" ") ?? ""}`}
              onSelect={() => runAction(action)}
            >
              <action.icon className="mr-2 h-4 w-4" />
              <span>{action.label}</span>
            </CommandItem>
          ))}
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}

// Hook to trigger command palette from anywhere
export function useCommandPalette() {
  const trigger = React.useCallback(() => {
    const event = new KeyboardEvent("keydown", {
      key: "k",
      metaKey: true,
      bubbles: true,
    });
    document.dispatchEvent(event);
  }, []);

  return { trigger };
}
