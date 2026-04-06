import type { NodeState } from "./node.ts";

export type NodeSummary = {
  id: number;
  project_id: number;
  name: string;
  identifier: string;
  current_state: NodeState;
  last_seen_at: string | null;
  service_count: number;
  active_alert_count: number;
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
  node_summaries: NodeSummary[];
};
