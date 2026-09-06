import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { HistoryRange } from "@/types";

export function useMonitorHistory(monitorId: string, range: HistoryRange) {
  return useQuery({
    queryKey: ["monitors", monitorId, "history", range],
    queryFn: () => api.getMonitorHistory(monitorId, range),
    enabled: Boolean(monitorId),
    // Keep the previous range's data on screen while the new one loads,
    // instead of a loading flash. The chart holds its shape and just
    // dims slightly (driven by isFetching in the chart component).
    placeholderData: keepPreviousData,
    refetchInterval: 60_000,
  });
}
