import { useEffect, useState } from "react";
import { Activity, Pause, Play, RefreshCcw } from "lucide-react";

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
import { formatInterval, formatRelativeTime, type LiveUpdateState } from "@/hooks/useLiveUpdates";

interface LiveIndicatorProps {
  /** Live update state from useLiveUpdates hook. */
  state: LiveUpdateState;
  /** Whether data is currently being fetched. */
  isFetching?: boolean;
  /** Callback to manually refresh data. */
  onRefresh?: () => void;
  /** Additional class names. */
  className?: string;
}

/**
 * A visual indicator for real-time data updates.
 * Shows live status, last update time, and controls for toggling/configuring.
 */
export function LiveIndicator({
  state,
  isFetching = false,
  onRefresh,
  className,
}: LiveIndicatorProps) {
  const { isLive, interval, toggleLive, setInterval, intervalOptions, lastUpdated } = state;

  // Update the relative time display every second
  const [, setTick] = useState(0);
  useEffect(() => {
    const timer = window.setInterval(() => setTick((t) => t + 1), 1000);
    return () => window.clearInterval(timer);
  }, []);

  return (
    <div className={cn("flex items-center gap-2", className)}>
      {/* Live status indicator */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className={cn(
              "gap-2 text-xs font-medium transition-colors",
              isLive && "border-success/50 text-success hover:border-success"
            )}
          >
            {isLive ? (
              <>
                <span className="relative flex h-2 w-2">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-success opacity-75" />
                  <span className="relative inline-flex h-2 w-2 rounded-full bg-success" />
                </span>
                <span>Live</span>
                <span className="text-muted-foreground">({formatInterval(interval)})</span>
              </>
            ) : (
              <>
                <Pause className="h-3 w-3" />
                <span>Paused</span>
              </>
            )}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuLabel className="text-xs font-normal text-muted-foreground">
            Last updated: {formatRelativeTime(lastUpdated)}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={toggleLive}>
            {isLive ? (
              <>
                <Pause className="mr-2 h-4 w-4" />
                Pause updates
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" />
                Resume updates
              </>
            )}
          </DropdownMenuItem>
          {onRefresh && (
            <DropdownMenuItem onClick={onRefresh} disabled={isFetching}>
              <RefreshCcw className={cn("mr-2 h-4 w-4", isFetching && "animate-spin")} />
              Refresh now
            </DropdownMenuItem>
          )}
          <DropdownMenuSeparator />
          <DropdownMenuLabel className="text-xs font-normal text-muted-foreground">
            Refresh interval
          </DropdownMenuLabel>
          {intervalOptions.map((option) => (
            <DropdownMenuItem
              key={option.value}
              onClick={() => setInterval(option.value)}
              className={cn(
                interval === option.value && "bg-accent"
              )}
            >
              <Activity className="mr-2 h-4 w-4 opacity-0" />
              {option.label}
              {interval === option.value && (
                <span className="ml-auto text-xs text-muted-foreground">Active</span>
              )}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

/**
 * A minimal live indicator that just shows a pulsing dot.
 */
export function LiveDot({
  isLive,
  className,
}: {
  isLive: boolean;
  className?: string;
}) {
  if (!isLive) return null;

  return (
    <span className={cn("relative flex h-2 w-2", className)}>
      <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-success opacity-75" />
      <span className="relative inline-flex h-2 w-2 rounded-full bg-success" />
    </span>
  );
}
