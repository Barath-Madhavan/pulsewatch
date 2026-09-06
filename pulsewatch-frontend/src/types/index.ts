export type MonitorStatus = "up" | "down" | "degraded";

export interface Monitor {
  id: string;
  name: string;
  url: string;
  method: string;
  interval_seconds: number;
  timeout_seconds: number;
  expected_status_code: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateMonitorInput {
  name: string;
  url: string;
  method?: string;
  interval_seconds?: number;
  timeout_seconds?: number;
  expected_status_code?: number;
}

export interface UpdateMonitorInput {
  name?: string;
  url?: string;
  method?: string;
  interval_seconds?: number;
  timeout_seconds?: number;
  expected_status_code?: number;
  is_active?: boolean;
}

export interface CheckResult {
  monitor_id: string;
  status: MonitorStatus;
  status_code: number;
  response_time_ms: number;
  error?: string;
  checked_at: string;
}

export interface StatusUpdateMessage {
  type: "status_update";
  payload: CheckResult;
}

export type HistoryRange = "24h" | "7d" | "30d";

export interface HistoryBucket {
  bucket_start: string;
  avg_response_time_ms: number;
  worst_status: MonitorStatus;
  total_checks: number;
}

export interface HistoryResponse {
  range: HistoryRange;
  points: HistoryBucket[];
  uptime_percentage: number;
}

export interface AlertSettings {
  email_enabled: boolean;
  email_address: string;
  webhook_enabled: boolean;
  webhook_url: string;
  updated_at: string;
  demo_mode: boolean;
}

export interface UpdateAlertSettingsInput {
  email_enabled: boolean;
  email_address: string;
  webhook_enabled: boolean;
  webhook_url: string;
}

export interface Incident {
  id: string;
  monitor_id: string;
  monitor_name: string;
  started_at: string;
  resolved_at?: string | null;
  cause?: string;
}

export interface IncidentPage {
  incidents: Incident[];
  total: number;
}
