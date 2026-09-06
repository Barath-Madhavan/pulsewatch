import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

export const INCIDENT_PAGE_SIZE_OPTIONS = [10, 25, 50, 100] as const;
export const DEFAULT_INCIDENT_PAGE_SIZE = 10;

export function useIncidents(page: number, pageSize: number) {
  return useQuery({
    queryKey: ["incidents", page, pageSize],
    queryFn: () => api.listIncidents(pageSize, (page - 1) * pageSize),
    // Hold the previous page on screen (dimmed by the caller via isFetching)
    // while the next one loads, instead of a loading flash.
    placeholderData: keepPreviousData,
    refetchInterval: 30_000,
  });
}

export function useMonitorIncidents(monitorId: string, page: number, pageSize: number) {
  return useQuery({
    queryKey: ["monitors", monitorId, "incidents", page, pageSize],
    queryFn: () => api.getMonitorIncidents(monitorId, pageSize, (page - 1) * pageSize),
    enabled: Boolean(monitorId),
    placeholderData: keepPreviousData,
    refetchInterval: 30_000,
  });
}
