import { Moon, Sun } from "lucide-react";
import { useLiveStatuses, useSocketConnected } from "@/hooks/useWebSocket";
import { useMonitors } from "@/hooks/useMonitors";
import { useTheme } from "@/hooks/useTheme";
import { StatusDot, type DisplayStatus } from "@/components/ui/StatusDot";
import type { CheckResult, Monitor } from "@/types";

interface Summary {
  total: number;
  down: number;
  degraded: number;
  pending: number;
}

function summarize(
  monitors: Monitor[] | undefined,
  statuses: Map<string, CheckResult>,
): Summary {
  const active = (monitors ?? []).filter((m) => m.is_active);
  const summary: Summary = { total: active.length, down: 0, degraded: 0, pending: 0 };

  for (const m of active) {
    const s = statuses.get(m.id);
    if (!s) summary.pending += 1;
    else if (s.status === "down") summary.down += 1;
    else if (s.status === "degraded") summary.degraded += 1;
  }

  return summary;
}

// The signature element: a live-updating status pill, styled like Apple's
// system status indicators. Pulses only while something is actually down;
// restraint on every other state is what signals a mature product.
export function TopBar() {
  const { data: monitors, isLoading, isError } = useMonitors();
  const statuses = useLiveStatuses();
  const connected = useSocketConnected();
  const { resolvedTheme, setPreference } = useTheme();

  const display = getDisplay(isLoading, isError, monitors, statuses);

  return (
    <header className="sticky top-0 z-10 flex h-14 items-center justify-between gap-2 border-b border-border bg-bg/85 px-4 backdrop-blur sm:px-6">
      <div
        className="inline-flex min-w-0 items-center gap-2 rounded-full border border-border bg-surface px-3 py-1.5 shadow-sm"
        title={connected ? undefined : "Reconnecting to live updates…"}
      >
        <StatusDot status={display.status} size="sm" pulse={display.pulse} />
        <span className="truncate text-sm font-medium text-fg">{display.label}</span>
      </div>

      <button
        type="button"
        aria-label={resolvedTheme === "dark" ? "Switch to light theme" : "Switch to dark theme"}
        onClick={() => setPreference(resolvedTheme === "dark" ? "light" : "dark")}
        className="shrink-0 rounded-full p-2 text-fg-muted transition-colors hover:bg-surface-2 hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
      >
        {resolvedTheme === "dark" ? <Sun className="size-[18px]" /> : <Moon className="size-[18px]" />}
      </button>
    </header>
  );
}

function getDisplay(
  isLoading: boolean,
  isError: boolean,
  monitors: Monitor[] | undefined,
  statuses: Map<string, CheckResult>,
): { status: DisplayStatus; label: string; pulse: boolean } {
  if (isLoading) {
    return { status: "pending", label: "Loading monitors…", pulse: false };
  }

  if (isError) {
    return { status: "down", label: "Can't reach backend", pulse: true };
  }

  const summary = summarize(monitors, statuses);

  if (summary.total === 0) {
    const label = monitors && monitors.length > 0 ? "All monitors paused" : "No monitors yet";
    return { status: "pending", label, pulse: false };
  }

  if (summary.down > 0) {
    return {
      status: "down",
      label: `${summary.down} ${summary.down === 1 ? "issue" : "issues"} detected`,
      pulse: true,
    };
  }

  if (summary.degraded > 0) {
    return {
      status: "degraded",
      label: `${summary.degraded} ${summary.degraded === 1 ? "monitor" : "monitors"} degraded`,
      pulse: false,
    };
  }

  if (summary.pending > 0) {
    return { status: "pending", label: "Waiting for first checks…", pulse: false };
  }

  return { status: "up", label: "All systems operational", pulse: false };
}
