import { LineChart as LineChartIcon } from "lucide-react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { type ChartColors, useThemeColors } from "@/hooks/useTheme";
import { cn } from "@/lib/utils";
import type { HistoryBucket, HistoryRange, MonitorStatus } from "@/types";

const statusLabel: Record<MonitorStatus, string> = {
  up: "Up",
  down: "Down",
  degraded: "Degraded",
};

function statusColor(status: MonitorStatus, colors: ChartColors): string {
  return status === "down" ? colors.down : status === "degraded" ? colors.degraded : colors.up;
}

function formatAxisTick(iso: string, range: HistoryRange): string {
  const d = new Date(iso);
  if (range === "24h") {
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }
  if (range === "7d") {
    return d.toLocaleDateString([], { weekday: "short", hour: "2-digit" });
  }
  return d.toLocaleDateString([], { month: "short", day: "numeric" });
}

interface TooltipPayloadItem {
  payload: HistoryBucket;
}

function ChartTooltip({
  active,
  payload,
  colors,
}: {
  active?: boolean;
  payload?: TooltipPayloadItem[];
  colors: ChartColors;
}) {
  if (!active || !payload?.length) return null;
  const point = payload[0].payload;

  return (
    <div className="rounded-lg border border-border bg-surface px-3 py-2 shadow-lg">
      <p className="text-xs text-fg-muted">
        {new Date(point.bucket_start).toLocaleString()}
      </p>
      <p className="text-sm font-semibold text-fg">
        {Math.round(point.avg_response_time_ms)}ms
      </p>
      <p className="flex items-center gap-1.5 text-xs text-fg-muted">
        <span
          className="inline-block size-2 rounded-full"
          style={{ backgroundColor: statusColor(point.worst_status, colors) }}
        />
        {statusLabel[point.worst_status]}
        {point.total_checks > 1 && ` · ${point.total_checks} checks`}
      </p>
    </div>
  );
}

// Only anomalous buckets get a visible marker. A clean line tells the
// trend, and status color is reserved for pointing at what broke.
function AnomalyDot(props: {
  cx?: number;
  cy?: number;
  payload?: HistoryBucket;
  colors: ChartColors;
}) {
  const { cx, cy, payload, colors } = props;
  if (!payload || payload.worst_status === "up" || cx == null || cy == null) {
    return null;
  }
  return (
    <circle
      cx={cx}
      cy={cy}
      r={4}
      fill={statusColor(payload.worst_status, colors)}
      stroke={colors.surface}
      strokeWidth={2}
    />
  );
}

interface ResponseTimeChartProps {
  points: HistoryBucket[];
  range: HistoryRange;
  isFetching?: boolean;
}

export function ResponseTimeChart({ points, range, isFetching }: ResponseTimeChartProps) {
  const colors = useThemeColors();

  if (points.length === 0) {
    return (
      <div className="flex animate-fade-in flex-col items-center justify-center gap-2 py-16 text-center">
        <div className="flex size-12 items-center justify-center rounded-full bg-surface-2">
          <LineChartIcon className="size-6 text-fg-muted" />
        </div>
        <p className="text-sm font-medium text-fg">No history yet</p>
        <p className="text-sm text-fg-muted">
          Check results will start appearing here shortly.
        </p>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "h-[280px] w-full transition-opacity",
        isFetching && "opacity-60",
      )}
    >
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={points} margin={{ top: 8, right: 20, left: 0, bottom: 0 }}>
          <CartesianGrid
            stroke={colors.border}
            strokeDasharray="0"
            vertical={false}
          />
          <XAxis
            dataKey="bucket_start"
            tickFormatter={(v: string) => formatAxisTick(v, range)}
            tick={{ fill: colors.fgMuted, fontSize: 12 }}
            axisLine={{ stroke: colors.border }}
            tickLine={false}
            minTickGap={32}
          />
          <YAxis
            domain={[0, "dataMax"]}
            allowDataOverflow
            tick={{ fill: colors.fgMuted, fontSize: 12 }}
            axisLine={false}
            tickLine={false}
            width={44}
            tickFormatter={(v: number) => `${Math.round(v)}`}
          />
          <Tooltip content={<ChartTooltip colors={colors} />} cursor={{ stroke: colors.border }} />
          <Area
            type="monotone"
            dataKey="avg_response_time_ms"
            stroke={colors.accent}
            strokeWidth={2}
            fill={colors.accent}
            fillOpacity={0.1}
            dot={<AnomalyDot colors={colors} />}
            activeDot={{ r: 5, fill: colors.accent, stroke: colors.surface, strokeWidth: 2 }}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
