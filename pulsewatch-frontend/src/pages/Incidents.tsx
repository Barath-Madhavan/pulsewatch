import { AlertCircle, Inbox } from "lucide-react";
import { useState } from "react";
import { IncidentList } from "@/components/incidents/IncidentList";
import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Pagination } from "@/components/ui/Pagination";
import { Skeleton } from "@/components/ui/Skeleton";
import {
  DEFAULT_INCIDENT_PAGE_SIZE,
  INCIDENT_PAGE_SIZE_OPTIONS,
  useIncidents,
} from "@/hooks/useIncidents";
import { cn } from "@/lib/utils";

export function Incidents() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState<number>(DEFAULT_INCIDENT_PAGE_SIZE);
  const { data, isLoading, isError, error, isFetching } = useIncidents(page, pageSize);

  function handlePageSizeChange(size: number) {
    setPageSize(size);
    setPage(1);
  }

  const incidents = data?.incidents ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold tracking-tight text-fg">Incidents</h1>

      {isLoading ? (
        <div className="flex flex-col gap-3">
          {["a", "b", "c"].map((key) => (
            <Skeleton key={key} className="h-20 w-full" />
          ))}
        </div>
      ) : isError ? (
        <Card className="flex animate-fade-in flex-col items-center gap-2 p-8 text-center">
          <AlertCircle className="size-6 text-down" />
          <p className="text-sm font-medium text-down">Couldn't load incidents</p>
          <p className="mt-1 text-sm text-fg-muted">{(error as Error).message}</p>
        </Card>
      ) : incidents.length > 0 ? (
        <>
          <div className={cn("animate-fade-in transition-opacity", isFetching && "opacity-60")}>
            <IncidentList incidents={incidents} showMonitorName />
          </div>
          <Pagination
            page={page}
            pageSize={pageSize}
            total={total}
            pageSizeOptions={INCIDENT_PAGE_SIZE_OPTIONS}
            onPageChange={setPage}
            onPageSizeChange={handlePageSizeChange}
          />
        </>
      ) : (
        <EmptyState
          icon={Inbox}
          title="No incident history yet"
          description="Incidents are recorded automatically the moment a monitor goes down."
        />
      )}
    </div>
  );
}
