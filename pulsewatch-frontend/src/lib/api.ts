import type {
  AlertSettings,
  CreateMonitorInput,
  HistoryRange,
  HistoryResponse,
  IncidentPage,
  Monitor,
  UpdateAlertSettingsInput,
  UpdateMonitorInput,
} from "@/types";

const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `Request failed with status ${res.status}`);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

export const api = {
  listMonitors: () => request<Monitor[]>("/api/monitors"),
  getMonitor: (id: string) => request<Monitor>(`/api/monitors/${id}`),
  createMonitor: (input: CreateMonitorInput) =>
    request<Monitor>("/api/monitors", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  updateMonitor: (id: string, input: UpdateMonitorInput) =>
    request<Monitor>(`/api/monitors/${id}`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),
  deleteMonitor: (id: string) =>
    request<void>(`/api/monitors/${id}`, { method: "DELETE" }),
  getMonitorHistory: (id: string, range: HistoryRange) =>
    request<HistoryResponse>(`/api/monitors/${id}/history?range=${range}`),
  getAlertSettings: () => request<AlertSettings>("/api/settings/alerts"),
  updateAlertSettings: (input: UpdateAlertSettingsInput) =>
    request<AlertSettings>("/api/settings/alerts", {
      method: "PUT",
      body: JSON.stringify(input),
    }),
  listIncidents: (limit: number, offset: number) =>
    request<IncidentPage>(`/api/incidents?limit=${limit}&offset=${offset}`),
  getMonitorIncidents: (id: string, limit: number, offset: number) =>
    request<IncidentPage>(`/api/monitors/${id}/incidents?limit=${limit}&offset=${offset}`),
};
