import { cn } from "@/lib/utils";
import type { MonitorStatus } from "@/types";

export type DisplayStatus = MonitorStatus | "pending";

const statusClasses: Record<DisplayStatus, string> = {
  up: "bg-up text-up",
  down: "bg-down text-down",
  degraded: "bg-degraded text-degraded",
  pending: "bg-fg-muted text-fg-muted",
};

interface StatusDotProps {
  status: DisplayStatus;
  size?: "sm" | "md";
  pulse?: boolean;
  className?: string;
}

// Pulses only on an active outage by default. Restraint on every other
// state is what signals a mature product rather than a demo reel.
export function StatusDot({ status, size = "md", pulse, className }: StatusDotProps) {
  const shouldPulse = pulse ?? status === "down";
  const dimension = size === "sm" ? "size-1.5" : "size-2.5";

  return (
    <span
      className={cn(
        "inline-block shrink-0 rounded-full",
        dimension,
        statusClasses[status],
        shouldPulse && "animate-status-pulse",
        className,
      )}
      aria-hidden="true"
    />
  );
}
