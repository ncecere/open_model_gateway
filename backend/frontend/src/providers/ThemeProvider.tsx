import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import { useUserProfileQuery } from "@/apps/user/hooks/useUserData";
import type { ThemePreference } from "@/types/theme";

export type ThemeContextValue = {
  theme: ThemePreference;
  resolvedTheme: "light" | "dark";
  isLoading: boolean;
  setThemePreference: (preference: ThemePreference) => void;
};

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

type ThemeProviderProps = {
  children: React.ReactNode;
  isAuthenticated: boolean;
};

export function ThemeProvider({ children, isAuthenticated }: ThemeProviderProps) {
  const profileQuery = useUserProfileQuery({ enabled: isAuthenticated });
  const [preference, setPreference] = useState<ThemePreference>("system");
  const [resolvedTheme, setResolvedTheme] = useState<"light" | "dark">(
    getSystemTheme(),
  );

  const applyTheme = useCallback((pref: ThemePreference) => {
    const nextTheme = resolveTheme(pref);
    setResolvedTheme(nextTheme);
    if (typeof document !== "undefined") {
      document.documentElement.classList.toggle("dark", nextTheme === "dark");
    }
  }, []);

  useEffect(() => {
    applyTheme(preference);
  }, [preference, applyTheme]);

  useEffect(() => {
    const serverPref = profileQuery.data?.theme_preference as
      | ThemePreference
      | undefined;
    if (serverPref && serverPref !== preference) {
      setPreference(serverPref);
    }
  }, [profileQuery.data?.theme_preference, preference]);

  useEffect(() => {
    if (!isAuthenticated) {
      setPreference("system");
    }
  }, [isAuthenticated]);

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return;
    }
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => {
      if (preference === "system") {
        applyTheme("system");
      }
    };
    media.addEventListener("change", handler);
    return () => media.removeEventListener("change", handler);
  }, [preference, applyTheme]);

  const setThemePreference = useCallback((pref: ThemePreference) => {
    setPreference(pref);
  }, []);

  const value = useMemo(
    () => ({
      theme: preference,
      resolvedTheme,
      isLoading: profileQuery.isFetching && isAuthenticated,
      setThemePreference,
    }),
    [preference, resolvedTheme, profileQuery.isFetching, isAuthenticated, setThemePreference],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return ctx;
}

function getSystemTheme(): "light" | "dark" {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return "light";
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function resolveTheme(pref: ThemePreference): "light" | "dark" {
  return pref === "system" ? getSystemTheme() : pref;
}
