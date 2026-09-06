import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { CreateMonitorInput, UpdateMonitorInput } from "@/types";

const monitorsKey = ["monitors"] as const;

export function useMonitors() {
  return useQuery({
    queryKey: monitorsKey,
    queryFn: api.listMonitors,
  });
}

export function useMonitor(id: string) {
  return useQuery({
    queryKey: [...monitorsKey, id],
    queryFn: () => api.getMonitor(id),
    enabled: Boolean(id),
  });
}

export function useCreateMonitor() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateMonitorInput) => api.createMonitor(input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: monitorsKey }),
  });
}

export function useUpdateMonitor(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateMonitorInput) => api.updateMonitor(id, input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: monitorsKey }),
  });
}

export function useDeleteMonitor() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteMonitor(id),
    // Only invalidate the list (exact match). Touching the deleted
    // monitor's own ["monitors", id] query, via invalidate or remove,
    // makes any still-mounted detail-page observer refetch it immediately,
    // which just 404s a moment before the navigate-away unmounts it.
    onSuccess: () => queryClient.invalidateQueries({ queryKey: monitorsKey, exact: true }),
  });
}
