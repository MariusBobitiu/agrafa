export type Project = {
  id: number;
  slug: string;
  name: string;
  created_at: string;
};

export type ProjectSummary = Project & {
  current_user_role: string;
};

export type ProjectDetail = ProjectSummary & {
  node_count: number;
  service_count: number;
  active_alert_count: number;
};
