export type AlertStatus = "active" | "resolved";

export type Severity = "info" | "warning" | "critical";

export type RuleType =
  | "node_offline"
  | "service_unhealthy"
  | "cpu_above_threshold"
  | "memory_above_threshold"
  | "disk_above_threshold";

export type AlertRule = {
  id: number;
  project_id: number;
  node_id: number | null;
  service_id: number | null;
  rule_type: RuleType;
  threshold_value: number | null;
  severity: Severity;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type Alert = {
  id: number;
  project_id: number;
  alert_rule_id: number;
  node_id: number | null;
  service_id: number | null;
  title: string;
  message: string;
  status: AlertStatus;
  triggered_at: string;
  resolved_at: string | null;
};

export type AlertRuleCreateInput = {
  project_id: number;
  node_id?: number | null;
  service_id?: number | null;
  rule_type: RuleType;
  threshold_value?: number | null;
  severity: Severity;
};

export type AlertRuleUpdateInput = {
  threshold_value?: number | null;
  severity?: Severity;
  is_enabled?: boolean;
  node_id?: number | null;
  service_id?: number | null;
};
