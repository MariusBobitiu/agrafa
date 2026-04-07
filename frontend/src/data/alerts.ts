import { api } from "@/lib/fetch-client.ts";
import type {
  Alert,
  AlertRule,
  AlertRuleCreateInput,
  AlertRuleUpdateInput,
} from "@/types/alert.ts";

export const alertsApi = {
  listAlerts: (projectId: number): Promise<{ alerts: Alert[] }> =>
    api.get(`/alerts?project_id=${projectId}`),

  listRules: (projectId: number): Promise<{ alert_rules: AlertRule[] }> =>
    api.get(`/alert-rules?project_id=${projectId}`),

  getRule: (id: number): Promise<{ alert_rule: AlertRule }> =>
    api.get(`/alert-rules/${id}`),

  createRule: (payload: AlertRuleCreateInput): Promise<{ alert_rule: AlertRule }> =>
    api.post("/alert-rules", payload),

  updateRule: (id: number, payload: AlertRuleUpdateInput): Promise<{ alert_rule: AlertRule }> =>
    api.patch(`/alert-rules/${id}`, payload),

  deleteRule: (id: number): Promise<void> =>
    api.del(`/alert-rules/${id}`),
};
