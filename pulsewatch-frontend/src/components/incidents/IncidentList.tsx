import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/Badge";
import { Card } from "@/components/ui/Card";
import { StatusDot } from "@/components/ui/StatusDot";
import { formatIncidentDuration } from "@/lib/utils";
import type { Incident } from "@/types";

interface IncidentListProps {
  incidents: Incident[];
  showMonitorName?: boolean;
}

export function IncidentList({ incidents, showMonitorName }: IncidentListProps) {
  return (
    <div className="flex flex-col gap-3">
      {incidents.map((incident) => {
        const resolved = Boolean(incident.resolved_at);
        return (
          <Card key={incident.id} className="flex items-start gap-3 p-4">
            <StatusDot
              status={resolved ? "up" : "down"}
              pulse={!resolved}
              className="mt-1"
            />
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                {showMonitorName && (
                  <Link
                    to={`/monitors/${incident.monitor_id}`}
                    className="rounded-sm text-sm font-semibold text-fg transition-colors hover:text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
                  >
                    {incident.monitor_name}
                  </Link>
                )}
                <Badge variant={resolved ? "neutral" : "down"}>
                  {resolved ? "Resolved" : "Ongoing"}
                </Badge>
                <span className="text-xs text-fg-muted">
                  {formatIncidentDuration(incident.started_at, incident.resolved_at)}
                </span>
              </div>

              <p className="mt-1 text-xs text-fg-muted">
                Started {new Date(incident.started_at).toLocaleString()}
                {incident.resolved_at &&
                  ` · Resolved ${new Date(incident.resolved_at).toLocaleString()}`}
              </p>

              {incident.cause && (
                <p className="mt-1 truncate font-mono text-xs text-fg-muted">
                  {incident.cause}
                </p>
              )}
            </div>
          </Card>
        );
      })}
    </div>
  );
}
