export type ServiceStatus = "healthy" | "degraded" | "unhealthy" | "unknown";
export type CheckType = "http" | "tcp";
export type ServiceExecutionMode = "managed" | "agent";

export type HealthCheckSummary = {
  observed_at: string;
  is_success: boolean;
  status_code: number | null;
  response_time_ms: number | null;
  message: string | null;
};

export type ServiceAlertSeverity = "critical" | "warning" | "info";

export type ServiceAlert = {
  id: number;
  rule_id: number;
  rule_type: string;
  severity: ServiceAlertSeverity;
  title: string;
  status: "active";
  triggered_at: string;
};

export type Service = {
  id: number;
  project_id: number;
  node_id: number;
  execution_mode: ServiceExecutionMode;
  name: string;
  check_type: CheckType;
  check_target: string;
  status: ServiceStatus;
  last_checked_at: string | null;
  consecutive_failures: number;
  active_alert_count: number;
  active_alerts?: ServiceAlert[];
  latest_health_check: HealthCheckSummary | null;
  created_at: string;
  updated_at: string;
};

export type ServiceCreateInput = {
  project_id: number;
  execution_mode: ServiceExecutionMode;
  node_id?: number;
  name: string;
  check_type: CheckType;
  check_target: string;
  check_interval_seconds?: number;
  timeout_seconds?: number;
  consecutive_failures_before_alert?: number;
};

export type ServiceUpdateInput = Partial<Omit<ServiceCreateInput, "project_id" | "node_id" | "execution_mode">>;

export type ServiceResponse = Service;
