import { cn } from "@/lib/utils";

export type StatusTone = "success" | "warning" | "danger" | "muted";

const toneClasses: Record<StatusTone, string> = {
  success: "bg-emerald-500 text-white hover:bg-emerald-500",
  warning: "bg-amber-500/80 text-black hover:bg-amber-500",
  danger: "bg-destructive text-destructive-foreground hover:bg-destructive",
  muted: "bg-muted text-foreground",
};

export function statusToneClass(tone: StatusTone, ...className: string[]) {
  return cn(toneClasses[tone], ...className);
}

export function toneFromStatus(status?: string | null): StatusTone {
  const normalized = status?.toString().trim().toLowerCase() ?? "";
  if (
    normalized === "ok" ||
    normalized === "online" ||
    normalized === "healthy" ||
    normalized === "success" ||
    normalized === "enabled" ||
    normalized === "active" ||
    normalized === "passing"
  ) {
    return "success";
  }
  if (
    normalized === "partial" ||
    normalized === "warn" ||
    normalized === "warning" ||
    normalized === "degraded" ||
    normalized === "pending"
  ) {
    return "warning";
  }
  if (
    normalized === "offline" ||
    normalized === "error" ||
    normalized === "failed" ||
    normalized === "failure" ||
    normalized === "disabled" ||
    normalized === "critical"
  ) {
    return "danger";
  }
  return "muted";
}
