import { deduplicateAlerts } from "@/app/alerts/alert-presentation.ts";
import type { Alert, AlertHistoryFilters, AlertRule } from "@/types/alert.ts";
import type { ServiceAlert } from "@/types/service.ts";

export const SERVICE_OUTAGE_HISTORY_LIMIT = 10;

export function serviceOutageHistoryFilters(serviceId: number): AlertHistoryFilters {
  return { serviceId, ruleType: "service_unhealthy" };
}

export function serviceOutageAlerts(alerts: readonly ServiceAlert[]) {
  return alerts.filter((alert) => alert.rule_type === "service_unhealthy");
}

export function otherServiceAlerts(alerts: readonly ServiceAlert[]) {
  return alerts.filter((alert) => alert.rule_type !== "service_unhealthy");
}

export function resolvedServiceOutages(alerts: readonly Alert[], serviceId: number) {
  return deduplicateAlerts([...alerts]).filter(
    (alert) =>
      alert.status === "resolved" &&
      alert.rule_type === "service_unhealthy" &&
      alert.service_id === serviceId,
  );
}

export function hasApplicableServiceOutageRule(rules: readonly AlertRule[], serviceId: number) {
  return rules.some(
    (rule) =>
      rule.is_enabled &&
      rule.rule_type === "service_unhealthy" &&
      (rule.target_scope === "all" ||
        (rule.target_scope === "specific" && rule.service_id === serviceId)),
  );
}
