import { Plus, Search, X } from "lucide-react";
import { AnimatePresence } from "motion/react";
import { useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/Button";
import { MonitorFormModal } from "@/components/monitors/MonitorFormModal";
import { MonitorGrid } from "@/components/monitors/MonitorGrid";
import { MonitorStats } from "@/components/monitors/MonitorStats";
import { useMonitors } from "@/hooks/useMonitors";

export function Dashboard() {
  const { data: monitors, isLoading, isError, error } = useMonitors();
  const [isAddOpen, setIsAddOpen] = useState(false);
  const [search, setSearch] = useState("");
  const addButtonRef = useRef<HTMLButtonElement>(null);

  const trimmedSearch = search.trim().toLowerCase();
  const filteredMonitors = useMemo(() => {
    if (!trimmedSearch) return monitors;
    return monitors?.filter((m) => m.name.toLowerCase().includes(trimmedSearch));
  }, [monitors, trimmedSearch]);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-fg">Monitors</h1>
          <p className="mt-1 text-sm text-fg-muted">
            Live uptime and response time for every tracked endpoint.
          </p>
        </div>

        <div className="flex shrink-0 items-center gap-3">
          <div className="relative w-64 shrink-0">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-fg-muted" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search monitors…"
              aria-label="Search monitors by name"
              className="w-full rounded-lg border border-border bg-bg py-2 pr-8 pl-9 text-sm text-fg placeholder:text-fg-muted transition-colors focus-visible:border-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
            />
            {search && (
              <button
                type="button"
                aria-label="Clear search"
                onClick={() => setSearch("")}
                className="absolute top-1/2 right-2 -translate-y-1/2 rounded-md p-1 text-fg-muted transition-colors hover:bg-surface-2 hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
              >
                <X className="size-3.5" />
              </button>
            )}
          </div>

          <Button ref={addButtonRef} onClick={() => setIsAddOpen(true)}>
            <Plus className="size-4" />
            Add Monitor
          </Button>
        </div>
      </div>

      {/* Stats reflect every monitor, independent of the search box above,
          so filtering the grid never makes the overview lie about the
          whole fleet. */}
      <MonitorStats monitors={monitors} isLoading={isLoading} />

      <MonitorGrid
        monitors={filteredMonitors}
        isLoading={isLoading}
        isError={isError}
        error={error}
        isSearchFiltered={Boolean(trimmedSearch)}
      />

      <AnimatePresence>
        {isAddOpen && (
          <MonitorFormModal
            onClose={() => {
              setIsAddOpen(false);
              addButtonRef.current?.focus();
            }}
          />
        )}
      </AnimatePresence>
    </div>
  );
}
