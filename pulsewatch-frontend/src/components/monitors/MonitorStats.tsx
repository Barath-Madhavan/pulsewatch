import { AlertTriangle, Gauge, Radar, TrendingUp } from "lucide-react";
import type { ReactNode } from "react";
import { AnimatedNumber } from "@/components/ui/AnimatedNumber";
import { Card } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";
import { useLiveStatuses } from "@/hooks/useWebSocket";
import type { Monitor } from "@/types";

interface MonitorStatsProps {
  monitors: Monitor[] | undefined;
  isLoading: boolean;
}

type Tone = "up" | "down" | "degraded" | "neutral";

interface Tile {
  icon: typeof Gauge;
  label: string;
  value: ReactNode;
  tone: Tone;
}

const toneTextClasses: Record<Tone, string> = {
  up: "text-up",
  down: "text-down",
  degraded: "text-degraded",
  neutral: "text-fg",
};

const toneChipClasses: Record<Tone, string> = {
  up: "bg-up/10 text-up",
  down: "bg-down/10 text-down",
  degraded: "bg-degraded/10 text-degraded",
  neutral: "bg-accent/10 text-accent",
};

const toneBarClasses: Record<Tone, string> = {
  up: "bg-up",
  down: "bg-down",
  degraded: "bg-degraded",
  neutral: "bg-accent",
};

// Derived entirely from data already in memory (the monitor list + the
// live WebSocket status map), so it costs no extra requests.
export function MonitorStats({ monitors, isLoading }: MonitorStatsProps) {
  const statuses = useLiveStatuses();

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {["a", "b", "c", "d"].map((key) => (
          <Skeleton key={key} className="h-[84px] w-full" />
        ))}
      </div>
    );
  }

  const active = (monitors ?? []).filter((m) => m.is_active);
  let up = 0;
  let down = 0;
  let degraded = 0;
  let responseSum = 0;
  let responseCount = 0;

  for (const m of active) {
    const s = statuses.get(m.id);
    if (!s) continue;
    if (s.status === "down") {
      down += 1;
      continue;
    }
    if (s.status === "degraded") degraded += 1;
    else up += 1;
    responseSum += s.response_time_ms;
    responseCount += 1;
  }

  const reporting = up + down + degraded;
  // Optimistic default when nothing has reported in yet, matching the
  // backend's own convention (CheckResultRepo.UptimePercentage returns 100
  // when there's no data rather than treating "unknown" as "down").
  const operationalPct = reporting > 0 ? ((up + degraded) / reporting) * 100 : 100;
  const avgResponseMs = responseCount > 0 ? responseSum / responseCount : 0;

  const tiles: Tile[] = [
    {
      icon: Radar,
      label: "Monitors",
      value: <AnimatedNumber value={monitors?.length ?? 0} />,
      tone: "neutral",
    },
    {
      icon: TrendingUp,
      label: "Operational now",
      value: <AnimatedNumber value={operationalPct} suffix="%" />,
      tone: down > 0 ? "down" : degraded > 0 ? "degraded" : "up",
    },
    {
      icon: AlertTriangle,
      label: "Active issues",
      value: <AnimatedNumber value={down} />,
      tone: down > 0 ? "down" : "neutral",
    },
    {
      icon: Gauge,
      label: "Avg response",
      value: <AnimatedNumber value={avgResponseMs} suffix="ms" />,
      tone: "neutral",
    },
  ];

  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
      {tiles.map(({ icon: Icon, label, value, tone }) => (
        <Card key={label} className="relative flex items-center gap-2 overflow-hidden p-3 sm:gap-3 sm:p-4">
          <div className={`absolute inset-x-0 top-0 h-0.5 ${toneBarClasses[tone]}`} />
          <div className={`flex size-8 shrink-0 items-center justify-center rounded-lg sm:size-10 ${toneChipClasses[tone]}`}>
            <Icon className="size-4 sm:size-5" />
          </div>
          <div className="min-w-0">
            <p className="text-[11px] font-semibold tracking-wide text-fg-muted uppercase break-words">
              {label}
            </p>
            <p className={`text-lg font-bold tabular-nums sm:text-2xl ${toneTextClasses[tone]}`}>{value}</p>
          </div>
        </Card>
      ))}
    </div>
  );
}
