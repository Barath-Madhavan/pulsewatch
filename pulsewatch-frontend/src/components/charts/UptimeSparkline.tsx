import { useId, useMemo } from "react";
import { Area, ComposedChart, Line, ResponsiveContainer, YAxis } from "recharts";
import { useThemeColors } from "@/hooks/useTheme";
import type { HistoryBucket } from "@/types";

interface UptimeSparklineProps {
  points: HistoryBucket[];
}

// The chart is twice as wide as its container and scrolls left on a
// seamless loop - ambient motion layered on top of the real trend, which
// still only updates from the 60s history refetch.
//
// Data is mirrored (forward, then reversed) rather than repeated, so the
// value at the loop point always matches on both sides - a straight
// repeat would butt the trend's last value against its first, which
// rarely match, showing up as a fake spike at the seam.
export function UptimeSparkline({ points }: UptimeSparklineProps) {
  const colors = useThemeColors();
  const gradientId = useId();

  // globals.css's blanket prefers-reduced-motion rule only catches CSS
  // animations; the SMIL <animate> below needs this explicit opt-out too.
  const prefersReducedMotion = useMemo(
    () => window.matchMedia("(prefers-reduced-motion: reduce)").matches,
    [],
  );

  const mirrored = useMemo(() => [...points, ...[...points].reverse()], [points]);

  if (points.length < 2) return null;

  return (
    <div className="relative h-9 w-full overflow-hidden">
      <div className="h-full w-[200%] animate-sparkline-scroll">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={mirrored} margin={{ top: 4, right: 0, left: 0, bottom: 0 }}>
            {/* Sparklines show relative shape, not absolute magnitude, so
                the scale should zoom to the visible range rather than
                Recharts' default [0, auto] domain - anchoring at 0 squashes
                a 30-700ms trend into a sliver near the top and reads as
                flat even when the underlying values genuinely vary. */}
            <YAxis
              hide
              domain={([dataMin, dataMax]: readonly [number, number]): [number, number] => {
                const pad = Math.max(2, (dataMax - dataMin) * 0.15);
                return [dataMin - pad, dataMax + pad];
              }}
            />
            <Area
              type="monotone"
              dataKey="avg_response_time_ms"
              stroke={colors.fgMuted}
              strokeWidth={1.5}
              fill={colors.fgMuted}
              fillOpacity={0.08}
              isAnimationActive={false}
              dot={false}
            />
            {!prefersReducedMotion && (
              <>
                {/* A bright accent-colored band sweeps left-to-right along
                    the same curve on a loop, like a pulse traveling down a
                    wire, layered on top of the muted base line above. */}
                <defs>
                  <linearGradient id={gradientId} x1="-35%" y1="0" x2="0%" y2="0">
                    <stop offset="0%" stopColor={colors.accent} stopOpacity={0} />
                    <stop offset="50%" stopColor={colors.accent} stopOpacity={0.9} />
                    <stop offset="100%" stopColor={colors.accent} stopOpacity={0} />
                    <animate
                      attributeName="x1"
                      values="-35%;135%"
                      dur="2.6s"
                      repeatCount="indefinite"
                    />
                    <animate
                      attributeName="x2"
                      values="0%;170%"
                      dur="2.6s"
                      repeatCount="indefinite"
                    />
                  </linearGradient>
                </defs>
                <Line
                  type="monotone"
                  dataKey="avg_response_time_ms"
                  stroke={`url(#${gradientId})`}
                  strokeWidth={2}
                  isAnimationActive={false}
                  dot={false}
                />
              </>
            )}
          </ComposedChart>
        </ResponsiveContainer>
      </div>

      {/* Fades the scrolling line into the card's surface color at the
          right edge, so it reads as "emerging into view" rather than
          being abruptly clipped. */}
      <div
        className="pointer-events-none absolute inset-y-0 right-0 w-6 bg-gradient-to-r from-transparent to-surface"
        aria-hidden="true"
      />

      <span
        className="absolute top-1/2 right-0.5 size-1.5 -translate-y-1/2 rounded-full bg-accent text-accent animate-status-pulse"
        aria-hidden="true"
      />
    </div>
  );
}
