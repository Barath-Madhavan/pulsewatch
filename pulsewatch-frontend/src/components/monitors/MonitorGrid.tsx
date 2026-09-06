import { AlertCircle, Radar, SearchX } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import type { Monitor } from "@/types";
import { MonitorCard } from "./MonitorCard";

function MonitorCardSkeleton() {
  return (
    <Card className="flex flex-col gap-3 p-4">
      <div className="flex items-center gap-2.5">
        <Skeleton className="size-2.5 rounded-full" />
        <Skeleton className="h-4 w-32" />
      </div>
      <Skeleton className="h-3 w-full max-w-[220px]" />
      <div className="flex gap-1.5">
        <Skeleton className="h-5 w-14 rounded-full" />
        <Skeleton className="h-5 w-20 rounded-full" />
      </div>
      <Skeleton className="h-9 w-full" />
    </Card>
  );
}

interface MonitorGridProps {
  monitors: Monitor[] | undefined;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  /** True when `monitors` has already been narrowed by a search query. */
  isSearchFiltered?: boolean;
}

export function MonitorGrid({
  monitors,
  isLoading,
  isError,
  error,
  isSearchFiltered,
}: MonitorGridProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {["a", "b", "c"].map((key) => (
          <MonitorCardSkeleton key={key} />
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <Card className="flex animate-fade-in flex-col items-center gap-2 p-10 text-center">
        <AlertCircle className="size-6 text-down" />
        <p className="text-sm font-medium text-down">Couldn't load monitors</p>
        <p className="text-sm text-fg-muted">{(error as Error).message}</p>
      </Card>
    );
  }

  if (!monitors || monitors.length === 0) {
    return isSearchFiltered ? (
      <EmptyState
        icon={SearchX}
        title="No monitors match"
        description="Try a different name, or clear the search to see everything."
      />
    ) : (
      <EmptyState
        icon={Radar}
        title="No monitors yet"
        description="Add a URL to start tracking its uptime."
      />
    );
  }

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <AnimatePresence>
        {monitors.map((monitor, index) => (
          <motion.div
            key={monitor.id}
            layout
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.92, transition: { duration: 0.15 } }}
            transition={{ duration: 0.3, delay: index * 0.04, ease: [0.16, 1, 0.3, 1] }}
          >
            <MonitorCard monitor={monitor} />
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
}
