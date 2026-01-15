import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";

interface EmptyStateAction {
  label: string;
  onClick: () => void;
  variant?: "default" | "outline" | "secondary";
  icon?: LucideIcon;
}

interface EmptyStateProps {
  /** The illustration to display. */
  illustration?: "empty-box" | "no-results" | "no-data" | "error" | "offline" | "keys" | "users" | "files";
  /** Custom illustration (React component). */
  customIllustration?: ReactNode;
  /** Main heading text. */
  title: string;
  /** Supporting description text. */
  description?: string;
  /** Primary action button. */
  action?: EmptyStateAction;
  /** Secondary action button. */
  secondaryAction?: EmptyStateAction;
  /** Additional class names. */
  className?: string;
  /** Size variant. */
  size?: "sm" | "default" | "lg";
}

/**
 * A polished empty state component with SVG illustrations.
 * Use this to display meaningful feedback when there's no data to show.
 */
export function EmptyState({
  illustration = "empty-box",
  customIllustration,
  title,
  description,
  action,
  secondaryAction,
  className,
  size = "default",
}: EmptyStateProps) {
  const sizeClasses = {
    sm: "py-8",
    default: "py-12",
    lg: "py-16",
  };

  const illustrationSize = {
    sm: "w-24 h-24",
    default: "w-32 h-32",
    lg: "w-40 h-40",
  };

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center",
        sizeClasses[size],
        className
      )}
    >
      {/* Illustration */}
      <div className={cn("mb-6 text-muted-foreground/50", illustrationSize[size])}>
        {customIllustration ?? <Illustration type={illustration} />}
      </div>

      {/* Text content */}
      <div className="max-w-sm space-y-2">
        <h3 className="text-lg font-semibold tracking-tight">{title}</h3>
        {description && (
          <p className="text-sm text-muted-foreground">{description}</p>
        )}
      </div>

      {/* Actions */}
      {(action || secondaryAction) && (
        <div className="mt-6 flex items-center gap-3">
          {action && (
            <Button
              variant={action.variant ?? "default"}
              onClick={action.onClick}
              className="gap-2"
            >
              {action.icon && <action.icon className="h-4 w-4" />}
              {action.label}
            </Button>
          )}
          {secondaryAction && (
            <Button
              variant={secondaryAction.variant ?? "outline"}
              onClick={secondaryAction.onClick}
              className="gap-2"
            >
              {secondaryAction.icon && <secondaryAction.icon className="h-4 w-4" />}
              {secondaryAction.label}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

function Illustration({ type }: { type: EmptyStateProps["illustration"] }) {
  switch (type) {
    case "empty-box":
      return <EmptyBoxIllustration />;
    case "no-results":
      return <NoResultsIllustration />;
    case "no-data":
      return <NoDataIllustration />;
    case "error":
      return <ErrorIllustration />;
    case "offline":
      return <OfflineIllustration />;
    case "keys":
      return <KeysIllustration />;
    case "users":
      return <UsersIllustration />;
    case "files":
      return <FilesIllustration />;
    default:
      return <EmptyBoxIllustration />;
  }
}

// Minimal, elegant line illustrations
function EmptyBoxIllustration() {
  return (
    <svg viewBox="0 0 120 120" fill="none" className="w-full h-full">
      <rect
        x="20"
        y="40"
        width="80"
        height="60"
        rx="4"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M20 55 L60 75 L100 55"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M60 75 L60 100"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <path
        d="M30 40 L60 20 L90 40"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="60" cy="60" r="3" fill="currentColor" opacity="0.5" />
    </svg>
  );
}

function NoResultsIllustration() {
  return (
    <svg viewBox="0 0 120 120" fill="none" className="w-full h-full">
      <circle
        cx="50"
        cy="50"
        r="25"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <path
        d="M68 68 L90 90"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <path
        d="M40 50 L60 50"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <circle cx="90" cy="90" r="6" stroke="currentColor" strokeWidth="2" />
    </svg>
  );
}

function NoDataIllustration() {
  return (
    <svg viewBox="0 0 120 120" fill="none" className="w-full h-full">
      <rect
        x="25"
        y="20"
        width="70"
        height="80"
        rx="4"
        stroke="currentColor"
        strokeWidth="2"
      />
      <path
        d="M35 40 L85 40"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.5"
      />
      <path
        d="M35 55 L75 55"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.5"
      />
      <path
        d="M35 70 L65 70"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.5"
      />
      <circle cx="60" cy="60" r="20" stroke="currentColor" strokeWidth="2" strokeDasharray="4 4" />
    </svg>
  );
}

function ErrorIllustration() {
  return (
    <svg viewBox="0 0 120 120" fill="none" className="w-full h-full">
      <circle
        cx="60"
        cy="60"
        r="35"
        stroke="currentColor"
        strokeWidth="2"
      />
      <path
        d="M60 40 L60 65"
        stroke="currentColor"
        strokeWidth="3"
        strokeLinecap="round"
      />
      <circle cx="60" cy="78" r="3" fill="currentColor" />
    </svg>
  );
}

function OfflineIllustration() {
  return (
    <svg viewBox="0 0 120 120" fill="none" className="w-full h-full">
      <path
        d="M30 70 Q60 50 90 70"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.5"
      />
      <path
        d="M40 80 Q60 65 80 80"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.7"
      />
      <path
        d="M50 90 Q60 82 70 90"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <path
        d="M25 95 L95 25"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  );
}

function KeysIllustration() {
  return (
    <svg viewBox="0 0 120 120" fill="none" className="w-full h-full">
      <circle
        cx="45"
        cy="60"
        r="20"
        stroke="currentColor"
        strokeWidth="2"
      />
      <circle cx="45" cy="60" r="8" stroke="currentColor" strokeWidth="2" />
      <path
        d="M65 60 L100 60"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <path
        d="M85 60 L85 75"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <path
        d="M95 60 L95 70"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  );
}

function UsersIllustration() {
  return (
    <svg viewBox="0 0 120 120" fill="none" className="w-full h-full">
      <circle cx="60" cy="45" r="15" stroke="currentColor" strokeWidth="2" />
      <path
        d="M35 95 Q35 70 60 70 Q85 70 85 95"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <circle cx="30" cy="50" r="10" stroke="currentColor" strokeWidth="2" opacity="0.5" />
      <circle cx="90" cy="50" r="10" stroke="currentColor" strokeWidth="2" opacity="0.5" />
    </svg>
  );
}

function FilesIllustration() {
  return (
    <svg viewBox="0 0 120 120" fill="none" className="w-full h-full">
      <path
        d="M30 25 L30 95 L90 95 L90 40 L75 25 L30 25"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinejoin="round"
      />
      <path
        d="M75 25 L75 40 L90 40"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinejoin="round"
      />
      <path
        d="M40 55 L80 55"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.5"
      />
      <path
        d="M40 70 L70 70"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.5"
      />
      <path
        d="M40 85 L60 85"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.5"
      />
    </svg>
  );
}

export { EmptyBoxIllustration, NoResultsIllustration, NoDataIllustration, ErrorIllustration, OfflineIllustration, KeysIllustration, UsersIllustration, FilesIllustration };
