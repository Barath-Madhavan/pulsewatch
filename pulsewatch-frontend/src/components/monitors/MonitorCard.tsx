import { Pause, Pencil, Play, Trash2 } from "lucide-react";
import { AnimatePresence } from "motion/react";
import { useRef, useState } from "react";
import { Link } from "react-router-dom";
import { AnimatedNumber } from "@/components/ui/AnimatedNumber";
import { Badge } from "@/components/ui/Badge";
import { Card } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { StatusDot } from "@/components/ui/StatusDot";
import { UptimeSparkline } from "@/components/charts/UptimeSparkline";
import { MonitorFormModal } from "@/components/monitors/MonitorFormModal";
import { useDeleteMonitor, useUpdateMonitor } from "@/hooks/useMonitors";
import { useMonitorHistory } from "@/hooks/useMonitorHistory";
import { useLiveStatus } from "@/hooks/useWebSocket";
import { formatRelativeTime } from "@/lib/utils";
import type { Monitor } from "@/types";

interface MonitorCardProps {
  monitor: Monitor;
}

export function MonitorCard({ monitor }: MonitorCardProps) {
  const deleteMonitor = useDeleteMonitor();
  const updateMonitor = useUpdateMonitor(monitor.id);
  const live = useLiveStatus(monitor.id);
  const { data: history } = useMonitorHistory(monitor.id, "24h");
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isConfirmingDelete, setIsConfirmingDelete] = useState(false);
  const editButtonRef = useRef<HTMLButtonElement>(null);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  return (
    <Card interactive className="flex flex-col gap-3 p-4">
      <div className="flex items-start justify-between gap-2">
        <Link
          to={`/monitors/${monitor.id}`}
          className="flex min-w-0 items-center gap-2.5 rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
        >
          <StatusDot status={live?.status ?? "pending"} />
          <span className="truncate text-sm font-semibold text-fg">
            {monitor.name}
          </span>
        </Link>

        <div className="flex shrink-0 items-center gap-0.5">
          <button
            ref={editButtonRef}
            type="button"
            aria-label={`Edit ${monitor.name}`}
            onClick={() => setIsEditOpen(true)}
            className="rounded-md p-1.5 text-fg-muted transition-colors hover:bg-surface-2 hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
          >
            <Pencil className="size-4" />
          </button>
          <button
            type="button"
            aria-label={monitor.is_active ? `Pause ${monitor.name}` : `Resume ${monitor.name}`}
            onClick={() => updateMonitor.mutate({ is_active: !monitor.is_active })}
            disabled={updateMonitor.isPending}
            className="rounded-md p-1.5 text-fg-muted transition-colors hover:bg-surface-2 hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60 disabled:opacity-50"
          >
            {monitor.is_active ? <Pause className="size-4" /> : <Play className="size-4" />}
          </button>
          <button
            ref={deleteButtonRef}
            type="button"
            aria-label={`Delete ${monitor.name}`}
            onClick={() => setIsConfirmingDelete(true)}
            className="rounded-md p-1.5 text-fg-muted transition-colors hover:bg-down/10 hover:text-down focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-down/60"
          >
            <Trash2 className="size-4" />
          </button>
        </div>
      </div>

      <p className="truncate font-mono text-xs text-fg-muted">{monitor.url}</p>

      <div className="flex flex-wrap items-center gap-1.5">
        <Badge>{monitor.method}</Badge>
        <Badge>every {monitor.interval_seconds}s</Badge>
        {!monitor.is_active && <Badge variant="degraded">Paused</Badge>}
        {history && history.points.length > 0 && (
          <Badge variant="neutral">
            <AnimatedNumber value={history.uptime_percentage} decimals={1} suffix="%" /> (24h)
          </Badge>
        )}
      </div>

      {live && (
        <p className="font-mono text-xs text-fg-muted">
          {live.status_code > 0 && `${live.status_code} · `}
          <AnimatedNumber value={live.response_time_ms} suffix="ms" /> ·{" "}
          {formatRelativeTime(live.checked_at)}
        </p>
      )}

      {history && history.points.length >= 2 && (
        <UptimeSparkline points={history.points} />
      )}

      <AnimatePresence>
        {isEditOpen && (
          <MonitorFormModal
            monitor={monitor}
            onClose={() => {
              setIsEditOpen(false);
              editButtonRef.current?.focus();
            }}
          />
        )}
      </AnimatePresence>

      <AnimatePresence>
        {isConfirmingDelete && (
          <ConfirmDialog
            title="Delete monitor?"
            message={`This permanently deletes "${monitor.name}" along with its entire check history and incident log. This can't be undone.`}
            confirmLabel="Delete"
            danger
            isConfirming={deleteMonitor.isPending}
            onConfirm={() =>
              deleteMonitor.mutate(monitor.id, { onSuccess: () => setIsConfirmingDelete(false) })
            }
            onCancel={() => {
              setIsConfirmingDelete(false);
              deleteButtonRef.current?.focus();
            }}
          />
        )}
      </AnimatePresence>
    </Card>
  );
}
