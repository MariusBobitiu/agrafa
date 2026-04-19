import type { Alert } from "@/types/alert.ts";

export type NodeState = "online" | "offline" | "unknown";

export type MetricValue = {
  value: number;
  unit: string;
  observed_at: string;
};

export type Node = {
  id: number;
  project_id: number;
  name: string;
  identifier: string;
  current_state: NodeState;
  last_seen_at: string | null;
  metadata: Record<string, string>;
  latest_cpu: MetricValue | null;
  latest_memory: MetricValue | null;
  latest_disk: MetricValue | null;
  active_alert_count: number;
  active_alerts?: Alert[];
  service_count: number;
  created_at: string;
  updated_at: string;
};

export type NodeCreateInput = {
  project_id: number;
  name: string;
};

export type NodeUpdateInput = {
  name?: string;
};

export type NodeResponse = {
  id: number;
  project_id: number;
  name: string;
  identifier: string;
  agent_token?: string;
  created_at: string;
  updated_at: string;
};
