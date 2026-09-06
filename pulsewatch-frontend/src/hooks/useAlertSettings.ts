import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { UpdateAlertSettingsInput } from "@/types";

const alertSettingsKey = ["alert-settings"] as const;

export function useAlertSettings() {
  return useQuery({
    queryKey: alertSettingsKey,
    queryFn: api.getAlertSettings,
  });
}

export function useUpdateAlertSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateAlertSettingsInput) => api.updateAlertSettings(input),
    onSuccess: (settings) => queryClient.setQueryData(alertSettingsKey, settings),
  });
}
