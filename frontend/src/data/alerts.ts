import { api } from "@/lib/fetch-client.ts";
import type {
  Alert,
  AlertHistoryFilters,
  AlertPage,
  AlertRule,
  AlertRuleCreateInput,
  AlertRuleUpdateInput,
} from "@/types/alert.ts";

type AlertsResponse = {
  alerts: Alert[];
  pagination: {
    limit: number;
    has_more: boolean;
    next_cursor: string | null;
  };
};

function alertQuery(
  projectId: number,
  params: AlertHistoryFilters & {
    status?: "active" | "resolved";
    limit?: number;
    before?: string;
  } = {},
) {
  const search = new URLSearchParams({ project_id: String(projectId) });
  if (params.status) search.set("status", params.status);
  if (params.limit != null) search.set("limit", String(params.limit));
  if (params.before) search.set("before", params.before);
  if (params.category) search.set("category", params.category);
  if (params.severity) search.set("severity", params.severity);
  if (params.serviceId != null) search.set("service_id", String(params.serviceId));
  if (params.nodeId != null) search.set("node_id", String(params.nodeId));
  if (params.ruleType) search.set("rule_type", params.ruleType);
  return search.toString();
}

function toAlertPage(response: AlertsResponse): AlertPage {
  return {
    alerts: response.alerts,
    pagination: {
      limit: response.pagination.limit,
      hasMore: response.pagination.has_more,
      nextCursor: response.pagination.next_cursor,
    },
  };
}

export const alertsApi = {
  listAlerts: (projectId: number): Promise<{ alerts: Alert[] }> =>
    api.get(`/alerts?project_id=${projectId}`),

  listActive: async (projectId: number): Promise<AlertPage> =>
    toAlertPage(
      await api.get<AlertsResponse>(`/alerts?${alertQuery(projectId, { status: "active" })}`),
    ),

  listHistory: async (
    projectId: number,
    filters: AlertHistoryFilters,
    limit: number,
    before?: string,
    signal?: AbortSignal,
  ): Promise<AlertPage> =>
    toAlertPage(
      await api.get<AlertsResponse>(
        `/alerts?${alertQuery(projectId, { ...filters, status: "resolved", limit, before })}`,
        { signal },
      ),
    ),

  listRules: (projectId: number): Promise<{ alert_rules: AlertRule[] }> =>
    api.get(`/alert-rules?project_id=${projectId}`),

  getRule: (id: number): Promise<{ alert_rule: AlertRule }> => api.get(`/alert-rules/${id}`),

  createRule: (payload: AlertRuleCreateInput): Promise<{ alert_rule: AlertRule }> =>
    api.post("/alert-rules", payload),

  updateRule: (id: number, payload: AlertRuleUpdateInput): Promise<{ alert_rule: AlertRule }> =>
    api.patch(`/alert-rules/${id}`, payload),

  deleteRule: (id: number): Promise<void> => api.del(`/alert-rules/${id}`),
};
