import type { NodeState } from "./node.ts";

export type MetricValue = {
  value: number;
  unit: string;
  observed_at: string;
};

export type NodeSummary = {
  id: number;
  project_id: number;
  name: string;
  identifier: string;
  current_state: NodeState;
  last_seen_at: string | null;
  latest_cpu: MetricValue | null;
  latest_memory: MetricValue | null;
  latest_disk: MetricValue | null;
  service_count: number;
  active_alert_count: number;
};

export type OverviewEvent = {
  id: number;
  project_id: number;
  node_id: number | null;
  service_id: number | null;
  event_type: string;
  severity: string;
  title: string;
  details: unknown;
  occurred_at: string;
  created_at: string;
};

export type Overview = {
  total_projects: number;
  total_nodes: number;
  nodes_online: number;
  nodes_offline: number;
  total_services: number;
  services_healthy: number;
  services_degraded: number;
  services_unhealthy: number;
  active_alerts: number;
  resolved_alerts: number;
  recent_events: OverviewEvent[];
  recent_alert_events: OverviewEvent[];
  node_summaries: NodeSummary[];
};
