import { AlertCircle, ArrowLeft, Pause, Pencil, Play, Trash2 } from "lucide-react";
import { AnimatePresence } from "motion/react";
import { useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ResponseTimeChart } from "@/components/charts/ResponseTimeChart";
import { IncidentList } from "@/components/incidents/IncidentList";
import { MonitorFormModal } from "@/components/monitors/MonitorFormModal";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { Pagination } from "@/components/ui/Pagination";
import { Skeleton } from "@/components/ui/Skeleton";
import { StatusDot } from "@/components/ui/StatusDot";
import {
  DEFAULT_INCIDENT_PAGE_SIZE,
  INCIDENT_PAGE_SIZE_OPTIONS,
  useMonitorIncidents,
} from "@/hooks/useIncidents";
import { useDeleteMonitor, useMonitor, useUpdateMonitor } from "@/hooks/useMonitors";
import { useMonitorHistory } from "@/hooks/useMonitorHistory";
import { useLiveStatus } from "@/hooks/useWebSocket";
import { cn, formatRelativeTime } from "@/lib/utils";
import type { HistoryRange } from "@/types";

const statusLabel: Record<string, string> = {
  up: "Up",
  down: "Down",
  degraded: "Degraded",
};

const RANGES: { value: HistoryRange; label: string }[] = [
  { value: "24h", label: "24h" },
  { value: "7d", label: "7d" },
  { value: "30d", label: "30d" },
];

export function MonitorDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: monitor, isLoading, isError, error } = useMonitor(id ?? "");
  const deleteMonitor = useDeleteMonitor();
  const updateMonitor = useUpdateMonitor(id ?? "");
  const live = useLiveStatus(id ?? "");
  const [range, setRange] = useState<HistoryRange>("24h");
  const { data: history, isFetching: isHistoryFetching } = useMonitorHistory(
    id ?? "",
    range,
  );
  const [incidentPage, setIncidentPage] = useState(1);
  const [incidentPageSize, setIncidentPageSize] = useState<number>(
    DEFAULT_INCIDENT_PAGE_SIZE,
  );
  const { data: incidentData, isFetching: isFetchingIncidents } = useMonitorIncidents(
    id ?? "",
    incidentPage,
    incidentPageSize,
  );
  const incidents = incidentData?.incidents ?? [];
  const incidentTotal = incidentData?.total ?? 0;

  function handleIncidentPageSizeChange(size: number) {
    setIncidentPageSize(size);
    setIncidentPage(1);
  }
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isConfirmingDelete, setIsConfirmingDelete] = useState(false);
  const editButtonRef = useRef<HTMLButtonElement>(null);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-40 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (isError || !monitor) {
    return (
      <Card className="flex animate-fade-in flex-col items-center gap-3 p-8 text-center">
        <AlertCircle className="size-6 text-down" />
        <p className="text-sm text-fg-muted">
          {isError ? (error as Error).message : "Monitor not found."}
        </p>
        <Link
          to="/"
          className="inline-flex items-center gap-1.5 rounded-sm text-sm text-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
        >
          <ArrowLeft className="size-4" />
          Back to monitors
        </Link>
      </Card>
    );
  }

  return (
    <div className="flex animate-fade-in flex-col gap-4">
      <Link
        to="/"
        className="inline-flex w-fit items-center gap-1.5 rounded-sm text-sm text-fg-muted transition-colors hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
      >
        <ArrowLeft className="size-4" />
        Back to monitors
      </Link>

      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 items-center gap-3">
          <StatusDot status={live?.status ?? "pending"} size="md" />
          <h1 className="min-w-0 text-2xl font-semibold tracking-tight text-fg break-words">
            {monitor.name}
          </h1>
          {!monitor.is_active && <Badge variant="degraded">Paused</Badge>}
        </div>

        <div className="flex shrink-0 items-center gap-2 self-start">
          <Button ref={editButtonRef} variant="secondary" onClick={() => setIsEditOpen(true)}>
            <Pencil className="size-4" />
            Edit
          </Button>
          <Button
            variant="secondary"
            onClick={() => updateMonitor.mutate({ is_active: !monitor.is_active })}
            disabled={updateMonitor.isPending}
          >
            {monitor.is_active ? (
              <>
                <Pause className="size-4" />
                Pause
              </>
            ) : (
              <>
                <Play className="size-4" />
                Resume
              </>
            )}
          </Button>
          <Button
            ref={deleteButtonRef}
            variant="danger"
            onClick={() => setIsConfirmingDelete(true)}
          >
            <Trash2 className="size-4" />
            Delete
          </Button>
        </div>
      </div>

      <Card className="flex flex-col gap-4 p-4">
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Field label="URL" value={monitor.url} mono />
          <Field label="Method" value={monitor.method} />
          <Field label="Check interval" value={`${monitor.interval_seconds}s`} />
          <Field label="Timeout" value={`${monitor.timeout_seconds}s`} />
          <Field
            label="Expected status"
            value={String(monitor.expected_status_code)}
          />
          <Field
            label="Created"
            value={new Date(monitor.created_at).toLocaleString()}
          />
        </div>

        <div className="border-t border-border pt-4">
          <p className="mb-3 text-sm font-semibold text-fg">Latest check</p>
          {live ? (
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
              <Field
                label="Status"
                value={statusLabel[live.status] ?? live.status}
              />
              <Field label="HTTP status" value={String(live.status_code || "N/A")} />
              <Field label="Response time" value={`${live.response_time_ms}ms`} />
              <Field label="Checked" value={formatRelativeTime(live.checked_at)} />
              {live.error && (
                <div className="col-span-2 sm:col-span-4">
                  <Field label="Error" value={live.error} mono />
                </div>
              )}
            </div>
          ) : (
            <p className="text-sm text-fg-muted">
              Waiting for the first check to come in…
            </p>
          )}
        </div>
      </Card>

      <Card className="p-4">
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
            <p className="text-sm font-semibold text-fg">Response time</p>
            {history && history.points.length > 0 && (
              <p className="text-xs text-fg-muted">
                {history.uptime_percentage.toFixed(2)}% uptime ({range})
              </p>
            )}
          </div>

          <div className="inline-flex w-fit rounded-lg border border-border p-0.5">
            {RANGES.map(({ value, label }) => (
              <button
                key={value}
                type="button"
                onClick={() => setRange(value)}
                className={cn(
                  "rounded-md px-2.5 py-1 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60",
                  range === value
                    ? "bg-accent/10 text-accent"
                    : "text-fg-muted hover:text-fg",
                )}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <ResponseTimeChart
          points={history?.points ?? []}
          range={range}
          isFetching={isHistoryFetching}
        />
      </Card>

      {incidents.length > 0 && (
        <div className="flex flex-col gap-3">
          <p className="text-sm font-semibold text-fg">Incident history</p>
          <div className={cn("transition-opacity", isFetchingIncidents && "opacity-60")}>
            <IncidentList incidents={incidents} />
          </div>
          <Pagination
            page={incidentPage}
            pageSize={incidentPageSize}
            total={incidentTotal}
            pageSizeOptions={INCIDENT_PAGE_SIZE_OPTIONS}
            onPageChange={setIncidentPage}
            onPageSizeChange={handleIncidentPageSizeChange}
          />
        </div>
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
              deleteMonitor.mutate(monitor.id, { onSuccess: () => navigate("/") })
            }
            onCancel={() => {
              setIsConfirmingDelete(false);
              deleteButtonRef.current?.focus();
            }}
          />
        )}
      </AnimatePresence>
    </div>
  );
}

function Field({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div>
      <p className="text-xs text-fg-muted">{label}</p>
      <p className={mono ? "font-mono text-sm text-fg break-all" : "text-sm text-fg"}>
        {value}
      </p>
    </div>
  );
}
